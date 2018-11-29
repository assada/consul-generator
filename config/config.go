package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/consul-template/signals"
	"github.com/hashicorp/hcl"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/mitchellh/mapstructure"

	"github.com/pkg/errors"
)

const (
	DefaultLogLevel = "WARN"

	DefaultReloadSignal = syscall.SIGHUP

	DefaultKillSignal = syscall.SIGINT
)

var (
	homePath, _ = homedir.Dir()
)

type Config struct {
	Consul       *ConsulConfig  `mapstructure:"consul"`
	KillSignal   *os.Signal     `mapstructure:"kill_signal"`
	LogLevel     *string        `mapstructure:"log_level"`
	PidFile      *string        `mapstructure:"pid_file"`
	ReloadSignal *os.Signal     `mapstructure:"reload_signal"`
	Syslog       *SyslogConfig  `mapstructure:"syslog"`
	From         *string        `mapstructure:"from"`
	To           *string        `mapstructure:"to"`
	Interval     *time.Duration `mapstructure:"interval"`
}

func (c *Config) Copy() *Config {
	var o Config

	o.Consul = c.Consul

	if c.Consul != nil {
		o.Consul = c.Consul.Copy()
	}

	o.KillSignal = c.KillSignal

	o.LogLevel = c.LogLevel

	o.From = c.From

	o.Interval = c.Interval

	o.To = c.To

	o.PidFile = c.PidFile

	o.ReloadSignal = c.ReloadSignal

	if c.Syslog != nil {
		o.Syslog = c.Syslog.Copy()
	}

	return &o
}

func (c *Config) Merge(o *Config) *Config {
	if c == nil {
		if o == nil {
			return nil
		}
		return o.Copy()
	}

	if o == nil {
		return c.Copy()
	}

	r := c.Copy()

	if o.Consul != nil {
		r.Consul = r.Consul.Merge(o.Consul)
	}

	if o.From != nil {
		r.From = o.From
	}

	if o.Interval != nil {
		r.Interval = o.Interval
	}

	if o.To != nil {
		r.To = o.To
	}

	if o.KillSignal != nil {
		r.KillSignal = o.KillSignal
	}

	if o.LogLevel != nil {
		r.LogLevel = o.LogLevel
	}

	if o.PidFile != nil {
		r.PidFile = o.PidFile
	}

	if o.ReloadSignal != nil {
		r.ReloadSignal = o.ReloadSignal
	}

	if o.Syslog != nil {
		r.Syslog = r.Syslog.Merge(o.Syslog)
	}

	return r
}

func Parse(s string) (*Config, error) {
	var shadow interface{}
	if err := hcl.Decode(&shadow, s); err != nil {
		return nil, errors.Wrap(err, "error decoding config")
	}

	parsed, ok := shadow.(map[string]interface{})
	if !ok {
		return nil, errors.New("error converting config")
	}

	flattenKeys(parsed, []string{
		"auth",
		"consul",
		"consul.auth",
		"consul.retry",
		"consul.ssl",
		"consul.transport",
		"deduplicate",
		"env",
		"exec",
		"exec.env",
		"ssl",
		"syslog",
		"from",
		"to",
		"interval",
	})

	var c Config

	var md mapstructure.Metadata
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			ConsulStringToStructFunc(),
			StringToFileModeFunc(),
			signals.StringToSignalFunc(),
			mapstructure.StringToSliceHookFunc(","),
			mapstructure.StringToTimeDurationHookFunc(),
		),
		ErrorUnused: true,
		Metadata:    &md,
		Result:      &c,
	})
	if err != nil {
		return nil, errors.Wrap(err, "mapstructure decoder creation failed")
	}
	if err := decoder.Decode(parsed); err != nil {
		return nil, errors.Wrap(err, "mapstructure decode failed")
	}

	return &c, nil
}

func Must(s string) *Config {
	c, err := Parse(s)
	if err != nil {
		log.Fatal(err)
	}
	return c
}

func TestConfig(c *Config) *Config {
	d := DefaultConfig().Merge(c)
	d.Finalize()
	return d
}

func FromFile(path string) (*Config, error) {
	c, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "from file: "+path)
	}

	config, err := Parse(string(c))
	if err != nil {
		return nil, errors.Wrap(err, "from file: "+path)
	}
	return config, nil
}

func FromPath(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, errors.Wrap(err, "missing file/folder: "+path)
	}

	stat, err := os.Stat(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed stating file: "+path)
	}

	if stat.Mode().IsDir() {
		_, err := ioutil.ReadDir(path)
		if err != nil {
			return nil, errors.Wrap(err, "failed listing dir: "+path)
		}

		var c *Config

		err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			newConfig, err := FromFile(path)
			if err != nil {
				return err
			}
			c = c.Merge(newConfig)

			return nil
		})

		if err != nil {
			return nil, errors.Wrap(err, "walk error")
		}

		return c, nil
	} else if stat.Mode().IsRegular() {
		return FromFile(path)
	}

	return nil, fmt.Errorf("unknown filetype: %q", stat.Mode().String())
}

func (c *Config) GoString() string {
	if c == nil {
		return "(*Config)(nil)"
	}

	return fmt.Sprintf("&Config{"+
		"Consul:%#v, "+
		"KillSignal:%s, "+
		"LogLevel:%s, "+
		"PidFile:%s, "+
		"ReloadSignal:%s, "+
		"Syslog:%#v, "+
		"From:%#v, "+
		"To:%#v, "+
		"Interval:%#v, "+
		"}",
		c.Consul,
		SignalGoString(c.KillSignal),
		StringGoString(c.LogLevel),
		StringGoString(c.PidFile),
		SignalGoString(c.ReloadSignal),
		c.Syslog,
		c.From,
		c.To,
		c.Interval,
	)
}

func DefaultConfig() *Config {
	return &Config{
		Consul:   DefaultConsulConfig(),
		Syslog:   DefaultSyslogConfig(),
		From:     String("/"),
		To:       String("./"),
		Interval: TimeDuration(1 * time.Second),
	}
}

func (c *Config) Finalize() {

	if c.To == nil {
		c.To = String("./")
	}

	if c.From == nil {
		c.From = String("/")
	}

	if c.Consul == nil {
		c.Consul = DefaultConsulConfig()
	}
	c.Consul.Finalize()

	if c.KillSignal == nil {
		c.KillSignal = Signal(DefaultKillSignal)
	}

	if c.LogLevel == nil {
		c.LogLevel = stringFromEnv([]string{
			"CT_LOG",
			"CONSUL_TEMPLATE_LOG",
		}, DefaultLogLevel)
	}

	if c.PidFile == nil {
		c.PidFile = String("")
	}

	if c.ReloadSignal == nil {
		c.ReloadSignal = Signal(DefaultReloadSignal)
	}

	if c.Syslog == nil {
		c.Syslog = DefaultSyslogConfig()
	}
	c.Syslog.Finalize()
}

func stringFromEnv(list []string, def string) *string {
	for _, s := range list {
		if v := os.Getenv(s); v != "" {
			return String(strings.TrimSpace(v))
		}
	}
	return String(def)
}

func stringFromFile(list []string, def string) *string {
	for _, s := range list {
		c, err := ioutil.ReadFile(s)
		if err == nil {
			return String(strings.TrimSpace(string(c)))
		}
	}
	return String(def)
}

func antiboolFromEnv(list []string, def bool) *bool {
	for _, s := range list {
		if v := os.Getenv(s); v != "" {
			b, err := strconv.ParseBool(v)
			if err == nil {
				return Bool(!b)
			}
		}
	}
	return Bool(def)
}

func boolFromEnv(list []string, def bool) *bool {
	for _, s := range list {
		if v := os.Getenv(s); v != "" {
			b, err := strconv.ParseBool(v)
			if err == nil {
				return Bool(b)
			}
		}
	}
	return Bool(def)
}

func flattenKeys(m map[string]interface{}, keys []string) {
	keyMap := make(map[string]struct{})
	for _, key := range keys {
		keyMap[key] = struct{}{}
	}

	var flatten func(map[string]interface{}, string)
	flatten = func(m map[string]interface{}, parent string) {
		for k, v := range m {
			mapKey := k
			if parent != "" {
				mapKey = parent + "." + k
			}

			if _, ok := keyMap[mapKey]; !ok {
				continue
			}

			switch typed := v.(type) {
			case []map[string]interface{}:
				if len(typed) > 0 {
					last := typed[len(typed)-1]
					flatten(last, mapKey)
					m[k] = last
				} else {
					m[k] = nil
				}
			case map[string]interface{}:
				flatten(typed, mapKey)
				m[k] = typed
			default:
				m[k] = v
			}
		}
	}

	flatten(m, "")
}
