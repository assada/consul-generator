package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"syscall"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name string
		i    string
		e    *Config
		err  bool
	}{
		{
			"consul_address",
			`consul {
				address = "1.2.3.4"
			}`,
			&Config{
				Consul: &ConsulConfig{
					Address: String("1.2.3.4"),
				},
			},
			false,
		},
		{
			"consul_auth",
			`consul {
				auth {
					username = "username"
					password = "password"
				}
			}`,
			&Config{
				Consul: &ConsulConfig{
					Auth: &AuthConfig{
						Username: String("username"),
						Password: String("password"),
					},
				},
			},
			false,
		},
		{
			"consul_retry",
			`consul {
				retry {
					backoff  = "2s"
					attempts = 10
				}
			}`,
			&Config{
				Consul: &ConsulConfig{
					Retry: &RetryConfig{
						Attempts: Int(10),
						Backoff:  TimeDuration(2 * time.Second),
					},
				},
			},
			false,
		},
		{
			"consul_ssl",
			`consul {
				ssl {}
			}`,
			&Config{
				Consul: &ConsulConfig{
					SSL: &SSLConfig{},
				},
			},
			false,
		},
		{
			"consul_ssl_enabled",
			`consul {
				ssl {
					enabled = true
				}
			}`,
			&Config{
				Consul: &ConsulConfig{
					SSL: &SSLConfig{
						Enabled: Bool(true),
					},
				},
			},
			false,
		},
		{
			"consul_ssl_verify",
			`consul {
				ssl {
					verify = true
				}
			}`,
			&Config{
				Consul: &ConsulConfig{
					SSL: &SSLConfig{
						Verify: Bool(true),
					},
				},
			},
			false,
		},
		{
			"consul_ssl_cert",
			`consul {
				ssl {
					cert = "cert"
				}
			}`,
			&Config{
				Consul: &ConsulConfig{
					SSL: &SSLConfig{
						Cert: String("cert"),
					},
				},
			},
			false,
		},
		{
			"consul_ssl_key",
			`consul {
				ssl {
					key = "key"
				}
			}`,
			&Config{
				Consul: &ConsulConfig{
					SSL: &SSLConfig{
						Key: String("key"),
					},
				},
			},
			false,
		},
		{
			"consul_ssl_ca_cert",
			`consul {
				ssl {
					ca_cert = "ca_cert"
				}
			}`,
			&Config{
				Consul: &ConsulConfig{
					SSL: &SSLConfig{
						CaCert: String("ca_cert"),
					},
				},
			},
			false,
		},
		{
			"consul_ssl_ca_path",
			`consul {
				ssl {
					ca_path = "ca_path"
				}
			}`,
			&Config{
				Consul: &ConsulConfig{
					SSL: &SSLConfig{
						CaPath: String("ca_path"),
					},
				},
			},
			false,
		},
		{
			"consul_ssl_server_name",
			`consul {
				ssl {
					server_name = "server_name"
				}
			}`,
			&Config{
				Consul: &ConsulConfig{
					SSL: &SSLConfig{
						ServerName: String("server_name"),
					},
				},
			},
			false,
		},
		{
			"consul_token",
			`consul {
				token = "token"
			}`,
			&Config{
				Consul: &ConsulConfig{
					Token: String("token"),
				},
			},
			false,
		},
		{
			"consul_transport_dial_keep_alive",
			`consul {
				transport {
					dial_keep_alive = "10s"
				}
			}`,
			&Config{
				Consul: &ConsulConfig{
					Transport: &TransportConfig{
						DialKeepAlive: TimeDuration(10 * time.Second),
					},
				},
			},
			false,
		},
		{
			"consul_transport_dial_timeout",
			`consul {
				transport {
					dial_timeout = "10s"
				}
			}`,
			&Config{
				Consul: &ConsulConfig{
					Transport: &TransportConfig{
						DialTimeout: TimeDuration(10 * time.Second),
					},
				},
			},
			false,
		},
		{
			"consul_transport_disable_keep_alives",
			`consul {
				transport {
					disable_keep_alives = true
				}
			}`,
			&Config{
				Consul: &ConsulConfig{
					Transport: &TransportConfig{
						DisableKeepAlives: Bool(true),
					},
				},
			},
			false,
		},
		{
			"consul_transport_max_idle_conns_per_host",
			`consul {
				transport {
					max_idle_conns_per_host = 100
				}
			}`,
			&Config{
				Consul: &ConsulConfig{
					Transport: &TransportConfig{
						MaxIdleConnsPerHost: Int(100),
					},
				},
			},
			false,
		},
		{
			"consul_transport_tls_handshake_timeout",
			`consul {
				transport {
					tls_handshake_timeout = "30s"
				}
			}`,
			&Config{
				Consul: &ConsulConfig{
					Transport: &TransportConfig{
						TLSHandshakeTimeout: TimeDuration(30 * time.Second),
					},
				},
			},
			false,
		},
		{
			"kill_signal",
			`kill_signal = "SIGUSR1"`,
			&Config{
				KillSignal: Signal(syscall.SIGUSR1),
			},
			false,
		},
		{
			"log_level",
			`log_level = "WARN"`,
			&Config{
				LogLevel: String("WARN"),
			},
			false,
		},
		{
			"pid_file",
			`pid_file = "/var/pid"`,
			&Config{
				PidFile: String("/var/pid"),
			},
			false,
		},
		{
			"reload_signal",
			`reload_signal = "SIGUSR1"`,
			&Config{
				ReloadSignal: Signal(syscall.SIGUSR1),
			},
			false,
		},
		{
			"syslog",
			`syslog {}`,
			&Config{
				Syslog: &SyslogConfig{},
			},
			false,
		},
		{
			"syslog_enabled",
			`syslog {
				enabled = true
			}`,
			&Config{
				Syslog: &SyslogConfig{
					Enabled: Bool(true),
				},
			},
			false,
		},
		{
			"syslog_facility",
			`syslog {
				facility = "facility"
			}`,
			&Config{
				Syslog: &SyslogConfig{
					Facility: String("facility"),
				},
			},
			false,
		},
		{
			"invalid_key",
			`not_a_valid_key = "hello"`,
			nil,
			true,
		},
		{
			"invalid_stanza",
			`not_a_valid_stanza {
				a = "b"
			}`,
			nil,
			true,
		},
		{
			"mapstructure_error",
			`consul = true`,
			nil,
			true,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			c, err := Parse(tc.i)
			if (err != nil) != tc.err {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tc.e, c) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.e, c)
			}
		})
	}
}

func TestConfig_Merge(t *testing.T) {
	cases := []struct {
		name string
		a    *Config
		b    *Config
		r    *Config
	}{
		{
			"nil_a",
			nil,
			&Config{},
			&Config{},
		},
		{
			"nil_b",
			&Config{},
			nil,
			&Config{},
		},
		{
			"nil_both",
			nil,
			nil,
			nil,
		},
		{
			"empty",
			&Config{},
			&Config{},
			&Config{},
		},
		{
			"consul",
			&Config{
				Consul: &ConsulConfig{
					Address: String("consul"),
				},
			},
			&Config{
				Consul: &ConsulConfig{
					Address: String("consul-diff"),
				},
			},
			&Config{
				Consul: &ConsulConfig{
					Address: String("consul-diff"),
				},
			},
		},
		{
			"kill_signal",
			&Config{
				KillSignal: Signal(syscall.SIGUSR1),
			},
			&Config{
				KillSignal: Signal(syscall.SIGUSR2),
			},
			&Config{
				KillSignal: Signal(syscall.SIGUSR2),
			},
		},
		{
			"log_level",
			&Config{
				LogLevel: String("log_level"),
			},
			&Config{
				LogLevel: String("log_level-diff"),
			},
			&Config{
				LogLevel: String("log_level-diff"),
			},
		},
		{
			"pid_file",
			&Config{
				PidFile: String("pid_file"),
			},
			&Config{
				PidFile: String("pid_file-diff"),
			},
			&Config{
				PidFile: String("pid_file-diff"),
			},
		},
		{
			"reload_signal",
			&Config{
				ReloadSignal: Signal(syscall.SIGUSR1),
			},
			&Config{
				ReloadSignal: Signal(syscall.SIGUSR2),
			},
			&Config{
				ReloadSignal: Signal(syscall.SIGUSR2),
			},
		},
		{
			"syslog",
			&Config{
				Syslog: &SyslogConfig{
					Enabled: Bool(true),
				},
			},
			&Config{
				Syslog: &SyslogConfig{
					Enabled: Bool(false),
				},
			},
			&Config{
				Syslog: &SyslogConfig{
					Enabled: Bool(false),
				},
			},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			r := tc.a.Merge(tc.b)
			if !reflect.DeepEqual(tc.r, r) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.r, r)
			}
		})
	}
}

func TestFromPath(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	emptyDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(emptyDir)

	configDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(configDir)
	cf1, err := ioutil.TempFile(configDir, "")
	if err != nil {
		t.Fatal(err)
	}
	d := []byte(`
		consul {
			address = "1.2.3.4"
		}
	`)
	if err = ioutil.WriteFile(cf1.Name(), d, 0644); err != nil {
		t.Fatal(err)
	}
	cf2, err := ioutil.TempFile(configDir, "")
	if err != nil {
		t.Fatal(err)
	}
	d = []byte(`
		consul {
			token = "token"
		}
	`)
	if err := ioutil.WriteFile(cf2.Name(), d, 0644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		path string
		e    *Config
		err  bool
	}{
		{
			"missing_dir",
			"/not/a/real/dir",
			nil,
			true,
		},
		{
			"file",
			f.Name(),
			&Config{},
			false,
		},
		{
			"empty_dir",
			emptyDir,
			nil,
			false,
		},
		{
			"config_dir",
			configDir,
			&Config{
				Consul: &ConsulConfig{
					Address: String("1.2.3.4"),
					Token:   String("token"),
				},
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			c, err := FromPath(tc.path)
			if (err != nil) != tc.err {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tc.e, c) {
				t.Errorf("\nexp: %#v\nact: %#v", tc.e, c)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cases := []struct {
		env string
		val string
		e   *Config
		err bool
	}{
		{
			"CONSUL_HTTP_ADDR",
			"1.2.3.4",
			&Config{
				Consul: &ConsulConfig{
					Address: String("1.2.3.4"),
				},
			},
			false,
		},
		{
			"CONSUL_TEMPLATE_LOG",
			"DEBUG",
			&Config{
				LogLevel: String("DEBUG"),
			},
			false,
		},
		{
			"CONSUL_TOKEN",
			"token",
			&Config{
				Consul: &ConsulConfig{
					Token: String("token"),
				},
			},
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.env), func(t *testing.T) {
			if err := os.Setenv(tc.env, tc.val); err != nil {
				t.Fatal(err)
			}
			defer os.Unsetenv(tc.env)

			r := DefaultConfig()
			r.Merge(tc.e)

			c := DefaultConfig()
			if !reflect.DeepEqual(r, c) {
				t.Errorf("\nexp: %#v\nact: %#v", r, c)
			}
		})
	}
}
