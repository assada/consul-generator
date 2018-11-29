package logging

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"

	"github.com/hashicorp/go-syslog"
	"github.com/hashicorp/logutils"
)

var Levels = []logutils.LogLevel{"TRACE", "DEBUG", "INFO", "WARN", "ERR"}

type Config struct {
	Name string `json:"name"`

	Level string `json:"level"`

	Syslog         bool   `json:"syslog"`
	SyslogFacility string `json:"syslog_facility"`

	Writer io.Writer `json:"-"`
}

func Setup(config *Config) error {
	var logOutput io.Writer

	logFilter := NewLogFilter()
	logFilter.MinLevel = logutils.LogLevel(strings.ToUpper(config.Level))
	logFilter.Writer = config.Writer
	if !ValidateLevelFilter(logFilter.MinLevel, logFilter) {
		levels := make([]string, 0, len(logFilter.Levels))
		for _, level := range logFilter.Levels {
			levels = append(levels, string(level))
		}
		return fmt.Errorf("invalid log level %q, valid log levels are %s",
			config.Level, strings.Join(levels, ", "))
	}

	if config.Syslog {
		log.Printf("[DEBUG] (logging) enabling syslog on %s", config.SyslogFacility)

		l, err := gsyslog.NewLogger(gsyslog.LOG_NOTICE, config.SyslogFacility, config.Name)
		if err != nil {
			return fmt.Errorf("error setting up syslog logger: %s", err)
		}
		syslog := &SyslogWrapper{l, logFilter}
		logOutput = io.MultiWriter(logFilter, syslog)
	} else {
		logOutput = io.MultiWriter(logFilter)
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC)
	log.SetOutput(logOutput)

	return nil
}

func NewLogFilter() *logutils.LevelFilter {
	return &logutils.LevelFilter{
		Levels:   Levels,
		MinLevel: "WARN",
		Writer:   ioutil.Discard,
	}
}

func ValidateLevelFilter(min logutils.LogLevel, filter *logutils.LevelFilter) bool {
	for _, level := range filter.Levels {
		if level == min {
			return true
		}
	}
	return false
}
