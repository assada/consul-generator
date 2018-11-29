package main

import (
	"flag"
	"fmt"
	"github.com/Assada/consul-generator/config"
	"github.com/Assada/consul-generator/logging"
	"github.com/Assada/consul-generator/manager"
	"github.com/Assada/consul-generator/signals"
	"github.com/Assada/consul-generator/version"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"
)

const (
	ExitCodeOK int = 0

	ExitCodeError = 10 + iota
	ExitCodeInterrupt
	ExitCodeParseFlagsError
	ExitCodeRunnerError
	ExitCodeConfigError
)

type Cli struct {
	sync.Mutex

	outStream, errStream io.Writer

	signalCh chan os.Signal

	stopCh chan struct{}

	stopped bool
}

func NewCli(out, err io.Writer) *Cli {
	return &Cli{
		outStream: out,
		errStream: err,
		signalCh:  make(chan os.Signal, 1),
		stopCh:    make(chan struct{}),
	}
}

func (service *Cli) setup(conf *config.Config) (*config.Config, error) {
	if err := logging.Setup(&logging.Config{
		Name:           version.Name,
		Level:          config.StringVal(conf.LogLevel),
		Syslog:         config.BoolVal(conf.Syslog.Enabled),
		SyslogFacility: config.StringVal(conf.Syslog.Facility),
		Writer:         service.errStream,
	}); err != nil {
		return nil, err
	}

	return conf, nil
}

func (cli *Cli) Run(args []string) int {
	config, paths, once, dry, isVersion, err := cli.ParseFlags(args[1:])
	if err != nil {
		if err == flag.ErrHelp {
			fmt.Fprintf(cli.errStream, usage, version.Name)
			return 0
		}
		fmt.Fprintln(cli.errStream, err.Error())
		return ExitCodeParseFlagsError
	}

	cliConfig := config.Copy()

	config, err = loadConfigs(paths, cliConfig)
	if err != nil {
		return logError(err, ExitCodeConfigError)
	}

	config.Finalize()

	config, err = cli.setup(config)
	if err != nil {
		return logError(err, ExitCodeConfigError)
	}

	log.Printf("[INFO] %s", version.HumanVersion)

	if isVersion {
		log.Printf("[DEBUG] (cli) version flag was given, exiting now")
		fmt.Fprintf(cli.errStream, "%s\n", version.HumanVersion)
		return ExitCodeOK
	}

	runner, err := manager.NewRunner(config, dry, once)
	if err != nil {
		return logError(err, ExitCodeRunnerError)
	}
	go runner.Start()

	signal.Notify(cli.signalCh)

	for {
		select {
		case err := <-runner.ErrCh:
			code := ExitCodeRunnerError
			if typed, ok := err.(manager.ErrExitable); ok {
				code = typed.ExitStatus()
			}
			return logError(err, code)
		case <-runner.DoneCh:
			log.Printf("[INFO] (cli) received finish")
			runner.Stop()
			return ExitCodeOK
		case s := <-cli.signalCh:
			log.Printf("[DEBUG] (cli) receiving signal %q", s)

			switch s {
			case *config.ReloadSignal:
				fmt.Fprintf(cli.errStream, "Reloading configuration...\n")
				runner.Stop()

				config, err = loadConfigs(paths, cliConfig)
				if err != nil {
					return logError(err, ExitCodeConfigError)
				}
				config.Finalize()

				config, err = cli.setup(config)
				if err != nil {
					return logError(err, ExitCodeConfigError)
				}

				runner, err = manager.NewRunner(config, dry, once)
				if err != nil {
					return logError(err, ExitCodeRunnerError)
				}
				go runner.Start()
			case *config.KillSignal:
				fmt.Fprintf(cli.errStream, "Cleaning up...\n")
				runner.Stop()
				return ExitCodeInterrupt
			default:
				runner.Stop()
				return ExitCodeInterrupt
			}
		case <-cli.stopCh:
			return ExitCodeOK
		}
	}
}

func (cli *Cli) stop() {
	cli.Lock()
	defer cli.Unlock()

	if cli.stopped {
		return
	}

	close(cli.stopCh)
	cli.stopped = true
}

func (cli *Cli) ParseFlags(args []string) (*config.Config, []string, bool, bool, bool, error) {
	var dry, once, isVersion bool

	c := config.DefaultConfig()

	if s := os.Getenv("CT_LOCAL_CONFIG"); s != "" {
		envConfig, err := config.Parse(s)
		if err != nil {
			return nil, nil, false, false, false, err
		}
		c = c.Merge(envConfig)
	}

	configPaths := make([]string, 0, 6)

	flags := flag.NewFlagSet(version.Name, flag.ContinueOnError)
	flags.SetOutput(ioutil.Discard)
	flags.Usage = func() {}

	flags.Var((funcVar)(func(s string) error {
		configPaths = append(configPaths, s)
		return nil
	}), "config", "")

	flags.Var((funcVar)(func(s string) error {
		c.From = config.String(s)
		return nil
	}), "from", "")

	flags.Var((funcVar)(func(s string) error {
		c.To = config.String(s)
		return nil
	}), "to", "")

	flags.Var((funcIntVar)(func(s int) error {
		c.Interval = config.TimeDuration(time.Duration(s) * time.Second)
		return nil
	}), "interval", "")

	flags.Var((funcVar)(func(s string) error {
		c.Consul.Address = config.String(s)
		return nil
	}), "consul-addr", "")

	flags.Var((funcVar)(func(s string) error {
		a, err := config.ParseAuthConfig(s)
		if err != nil {
			return err
		}
		c.Consul.Auth = a
		return nil
	}), "consul-auth", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		c.Consul.Retry.Enabled = config.Bool(b)
		return nil
	}), "consul-retry", "")

	flags.Var((funcIntVar)(func(i int) error {
		c.Consul.Retry.Attempts = config.Int(i)
		return nil
	}), "consul-retry-attempts", "")

	flags.Var((funcDurationVar)(func(d time.Duration) error {
		c.Consul.Retry.Backoff = config.TimeDuration(d)
		return nil
	}), "consul-retry-backoff", "")

	flags.Var((funcDurationVar)(func(d time.Duration) error {
		c.Consul.Retry.MaxBackoff = config.TimeDuration(d)
		return nil
	}), "consul-retry-max-backoff", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		c.Consul.SSL.Enabled = config.Bool(b)
		return nil
	}), "consul-ssl", "")

	flags.Var((funcVar)(func(s string) error {
		c.Consul.SSL.CaCert = config.String(s)
		return nil
	}), "consul-ssl-ca-cert", "")

	flags.Var((funcVar)(func(s string) error {
		c.Consul.SSL.CaPath = config.String(s)
		return nil
	}), "consul-ssl-ca-path", "")

	flags.Var((funcVar)(func(s string) error {
		c.Consul.SSL.Cert = config.String(s)
		return nil
	}), "consul-ssl-cert", "")

	flags.Var((funcVar)(func(s string) error {
		c.Consul.SSL.Key = config.String(s)
		return nil
	}), "consul-ssl-key", "")

	flags.Var((funcVar)(func(s string) error {
		c.Consul.SSL.ServerName = config.String(s)
		return nil
	}), "consul-ssl-server-name", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		c.Consul.SSL.Verify = config.Bool(b)
		return nil
	}), "consul-ssl-verify", "")

	flags.Var((funcVar)(func(s string) error {
		c.Consul.Token = config.String(s)
		return nil
	}), "consul-token", "")

	flags.Var((funcDurationVar)(func(d time.Duration) error {
		c.Consul.Transport.DialKeepAlive = config.TimeDuration(d)
		return nil
	}), "consul-transport-dial-keep-alive", "")

	flags.Var((funcDurationVar)(func(d time.Duration) error {
		c.Consul.Transport.DialTimeout = config.TimeDuration(d)
		return nil
	}), "consul-transport-dial-timeout", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		c.Consul.Transport.DisableKeepAlives = config.Bool(b)
		return nil
	}), "consul-transport-disable-keep-alives", "")

	flags.Var((funcIntVar)(func(i int) error {
		c.Consul.Transport.MaxIdleConnsPerHost = config.Int(i)
		return nil
	}), "consul-transport-max-idle-conns-per-host", "")

	flags.Var((funcDurationVar)(func(d time.Duration) error {
		c.Consul.Transport.TLSHandshakeTimeout = config.TimeDuration(d)
		return nil
	}), "consul-transport-tls-handshake-timeout", "")

	flags.Var((funcVar)(func(s string) error {
		sig, err := signals.Parse(s)
		if err != nil {
			return err
		}
		c.KillSignal = config.Signal(sig)
		return nil
	}), "kill-signal", "")

	flags.Var((funcVar)(func(s string) error {
		c.LogLevel = config.String(s)
		return nil
	}), "log-level", "")

	flags.BoolVar(&once, "once", false, "")
	flags.BoolVar(&dry, "dry", false, "")

	flags.Var((funcVar)(func(s string) error {
		c.PidFile = config.String(s)
		return nil
	}), "pid-file", "")

	flags.Var((funcVar)(func(s string) error {
		sig, err := signals.Parse(s)
		if err != nil {
			return err
		}
		c.ReloadSignal = config.Signal(sig)
		return nil
	}), "reload-signal", "")

	flags.Var((funcBoolVar)(func(b bool) error {
		c.Syslog.Enabled = config.Bool(b)
		return nil
	}), "syslog", "")

	flags.Var((funcVar)(func(s string) error {
		c.Syslog.Facility = config.String(s)
		return nil
	}), "syslog-facility", "")

	flags.BoolVar(&isVersion, "v", false, "")
	flags.BoolVar(&isVersion, "version", false, "")

	if err := flags.Parse(args); err != nil {
		return nil, nil, false, false, false, err
	}

	args = flags.Args()
	if len(args) > 0 {
		return nil, nil, false, false, false, fmt.Errorf("cli: extra args: %q", args)
	}

	return c, configPaths, once, dry, isVersion, nil
}

func loadConfigs(paths []string, o *config.Config) (*config.Config, error) {
	finalC := config.DefaultConfig()

	for _, path := range paths {
		c, err := config.FromPath(path)
		if err != nil {
			return nil, err
		}

		finalC = finalC.Merge(c)
	}

	finalC = finalC.Merge(o)
	finalC.Finalize()
	return finalC, nil
}

func logError(err error, status int) int {
	log.Printf("[ERR] (cli) %s", err)
	return status
}

const usage = `Usage: %s [options]

  Watches a series of templates on the file system, writing new changes when
  Consul is updated. It runs until an interrupt is received unless the -once
  flag is specified.

Options:

  -config=<path>
      Sets the path to a configuration file or folder on disk. This can be
      specified multiple times to load multiple files or folders. If multiple
      values are given, they are merged left-to-right, and CLI arguments take
      the top-most precedence.

  -consul-addr=<address>
      Sets the address of the Consul instance

  -consul-auth=<username[:password]>
      Set the basic authentication username and password for communicating
      with Consul.

  -consul-retry
      Use retry logic when communication with Consul fails

  -consul-retry-attempts=<int>
      The number of attempts to use when retrying failed communications

  -consul-retry-backoff=<duration>
      The base amount to use for the backoff duration. This number will be
      increased exponentially for each retry attempt.

  -consul-retry-max-backoff=<duration>
      The maximum limit of the retry backoff duration. Default is one minute.
      0 means infinite. The backoff will increase exponentially until given value.

  -consul-ssl
      Use SSL when connecting to Consul

  -consul-ssl-ca-cert=<string>
      Validate server certificate against this CA certificate file list

  -consul-ssl-ca-path=<string>
      Sets the path to the CA to use for TLS verification

  -consul-ssl-cert=<string>
      SSL client certificate to send to server

  -consul-ssl-key=<string>
      SSL/TLS private key for use in client authentication key exchange

  -consul-ssl-server-name=<string>
      Sets the name of the server to use when validating TLS.

  -consul-ssl-verify
      Verify certificates when connecting via SSL

  -consul-token=<token>
      Sets the Consul API token

  -consul-transport-dial-keep-alive=<duration>
      Sets the amount of time to use for keep-alives

  -consul-transport-dial-timeout=<duration>
      Sets the amount of time to wait to establish a connection

  -consul-transport-disable-keep-alives
      Disables keep-alives (this will impact performance)

  -consul-transport-max-idle-conns-per-host=<int>
      Sets the maximum number of idle connections to permit per host

  -consul-transport-tls-handshake-timeout=<duration>
      Sets the handshake timeout

  -dry
      Print generated files to stdout instead of persist

  -once
      Do not run the process as a daemon

  -kill-signal=<signal>
      Signal to listen to gracefully terminate the process

  -log-level=<level>
      Set the logging level - values are "debug", "info", "warn", and "err"

  -pid-file=<path>
      Path on disk to write the PID of the process

  -from=<path>
      Consul path where files stored

  -to=<path>
      Path on disk to write generated files

  -interval=<int>
      Key update rate interval 

  -reload-signal=<signal>
      Signal to listen to reload configuration

  -syslog
      Send the output to syslog instead of standard error and standard out. The
      syslog facility defaults to LOCAL0 and can be changed using a
      configuration file

  -syslog-facility=<facility>
      Set the facility where syslog should log - if this attribute is supplied,
      the -syslog flag must also be supplied

  -v, -version
      Print the version of this daemon
`
