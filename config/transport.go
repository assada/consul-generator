package config

import (
	"fmt"
	"runtime"
	"time"
)

const (
	DefaultDialKeepAlive       = 30 * time.Second
	DefaultDialTimeout         = 30 * time.Second
	DefaultIdleConnTimeout     = 90 * time.Second
	DefaultMaxIdleConns        = 100
	DefaultTLSHandshakeTimeout = 10 * time.Second
)

var (
	DefaultMaxIdleConnsPerHost = runtime.GOMAXPROCS(0) + 1
)

type TransportConfig struct {
	DialKeepAlive       *time.Duration `mapstructure:"dial_keep_alive"`
	DialTimeout         *time.Duration `mapstructure:"dial_timeout"`
	DisableKeepAlives   *bool          `mapstructure:"disable_keep_alives"`
	IdleConnTimeout     *time.Duration `mapstructure:"idle_conn_timeout"`
	MaxIdleConns        *int           `mapstructure:"max_idle_conns"`
	MaxIdleConnsPerHost *int           `mapstructure:"max_idle_conns_per_host"`
	TLSHandshakeTimeout *time.Duration `mapstructure:"tls_handshake_timeout"`
}

func DefaultTransportConfig() *TransportConfig {
	return &TransportConfig{}
}

func (c *TransportConfig) Copy() *TransportConfig {
	if c == nil {
		return nil
	}

	var o TransportConfig

	o.DialKeepAlive = c.DialKeepAlive
	o.DialTimeout = c.DialTimeout
	o.DisableKeepAlives = c.DisableKeepAlives
	o.IdleConnTimeout = c.IdleConnTimeout
	o.MaxIdleConns = c.MaxIdleConns
	o.MaxIdleConnsPerHost = c.MaxIdleConnsPerHost
	o.TLSHandshakeTimeout = c.TLSHandshakeTimeout

	return &o
}

func (c *TransportConfig) Merge(o *TransportConfig) *TransportConfig {
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

	if o.DialKeepAlive != nil {
		r.DialKeepAlive = o.DialKeepAlive
	}

	if o.DialTimeout != nil {
		r.DialTimeout = o.DialTimeout
	}

	if o.DisableKeepAlives != nil {
		r.DisableKeepAlives = o.DisableKeepAlives
	}

	if o.IdleConnTimeout != nil {
		r.IdleConnTimeout = o.IdleConnTimeout
	}

	if o.MaxIdleConns != nil {
		r.MaxIdleConns = o.MaxIdleConns
	}

	if o.MaxIdleConnsPerHost != nil {
		r.MaxIdleConnsPerHost = o.MaxIdleConnsPerHost
	}

	if o.TLSHandshakeTimeout != nil {
		r.TLSHandshakeTimeout = o.TLSHandshakeTimeout
	}

	return r
}

func (c *TransportConfig) Finalize() {
	if c.DialKeepAlive == nil {
		c.DialKeepAlive = TimeDuration(DefaultDialKeepAlive)
	}

	if c.DialTimeout == nil {
		c.DialTimeout = TimeDuration(DefaultDialTimeout)
	}

	if c.DisableKeepAlives == nil {
		c.DisableKeepAlives = Bool(false)
	}

	if c.IdleConnTimeout == nil {
		c.IdleConnTimeout = TimeDuration(DefaultIdleConnTimeout)
	}

	if c.MaxIdleConns == nil {
		c.MaxIdleConns = Int(DefaultMaxIdleConns)
	}

	if c.MaxIdleConnsPerHost == nil {
		c.MaxIdleConnsPerHost = Int(DefaultMaxIdleConnsPerHost)
	}

	if c.TLSHandshakeTimeout == nil {
		c.TLSHandshakeTimeout = TimeDuration(DefaultTLSHandshakeTimeout)
	}
}

func (c *TransportConfig) GoString() string {
	if c == nil {
		return "(*TransportConfig)(nil)"
	}

	return fmt.Sprintf("&TransportConfig{"+
		"DialKeepAlive:%s, "+
		"DialTimeout:%s, "+
		"DisableKeepAlives:%t, "+
		"MaxIdleConnsPerHost:%d, "+
		"TLSHandshakeTimeout:%s"+
		"}",
		TimeDurationVal(c.DialKeepAlive),
		TimeDurationVal(c.DialTimeout),
		BoolVal(c.DisableKeepAlives),
		IntVal(c.MaxIdleConnsPerHost),
		TimeDurationVal(c.TLSHandshakeTimeout),
	)
}
