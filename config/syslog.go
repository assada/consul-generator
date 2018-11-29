package config

import "fmt"

const (
	DefaultSyslogFacility = "LOCAL0"
)

type SyslogConfig struct {
	Enabled  *bool   `mapstructure:"enabled"`
	Facility *string `mapstructure:"facility"`
}

func DefaultSyslogConfig() *SyslogConfig {
	return &SyslogConfig{}
}

func (c *SyslogConfig) Copy() *SyslogConfig {
	if c == nil {
		return nil
	}

	var o SyslogConfig
	o.Enabled = c.Enabled
	o.Facility = c.Facility
	return &o
}

func (c *SyslogConfig) Merge(o *SyslogConfig) *SyslogConfig {
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

	if o.Enabled != nil {
		r.Enabled = o.Enabled
	}

	if o.Facility != nil {
		r.Facility = o.Facility
	}

	return r
}

func (c *SyslogConfig) Finalize() {
	if c.Enabled == nil {
		c.Enabled = Bool(StringPresent(c.Facility))
	}

	if c.Facility == nil {
		c.Facility = String(DefaultSyslogFacility)
	}
}

func (c *SyslogConfig) GoString() string {
	if c == nil {
		return "(*SyslogConfig)(nil)"
	}

	return fmt.Sprintf("&SyslogConfig{"+
		"Enabled:%s, "+
		"Facility:%s"+
		"}",
		BoolGoString(c.Enabled),
		StringGoString(c.Facility),
	)
}
