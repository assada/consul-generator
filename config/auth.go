package config

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrAuthStringEmpty = errors.New("auth: cannot be empty")
)

type AuthConfig struct {
	Enabled  *bool   `mapstructure:"enabled"`
	Username *string `mapstructure:"username"`
	Password *string `mapstructure:"password"`
}

func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{}
}

func ParseAuthConfig(s string) (*AuthConfig, error) {
	if s == "" {
		return nil, ErrAuthStringEmpty
	}

	var a AuthConfig

	if strings.Contains(s, ":") {
		split := strings.SplitN(s, ":", 2)
		a.Username = String(split[0])
		a.Password = String(split[1])
	} else {
		a.Username = String(s)
	}

	return &a, nil
}

func (c *AuthConfig) Copy() *AuthConfig {
	if c == nil {
		return nil
	}

	var o AuthConfig
	o.Enabled = c.Enabled
	o.Username = c.Username
	o.Password = c.Password
	return &o
}

func (c *AuthConfig) Merge(o *AuthConfig) *AuthConfig {
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

	if o.Username != nil {
		r.Username = o.Username
	}

	if o.Password != nil {
		r.Password = o.Password
	}

	return r
}

func (c *AuthConfig) Finalize() {
	if c.Enabled == nil {
		c.Enabled = Bool(false ||
			StringPresent(c.Username) ||
			StringPresent(c.Password))
	}
	if c.Username == nil {
		c.Username = String("")
	}

	if c.Password == nil {
		c.Password = String("")
	}

	if c.Enabled == nil {
		c.Enabled = Bool(*c.Username != "" || *c.Password != "")
	}
}

func (c *AuthConfig) GoString() string {
	if c == nil {
		return "(*AuthConfig)(nil)"
	}

	return fmt.Sprintf("&AuthConfig{"+
		"Enabled:%s, "+
		"Username:%s, "+
		"Password:%s"+
		"}",
		BoolGoString(c.Enabled),
		StringGoString(c.Username),
		StringGoString(c.Password),
	)
}

func (c *AuthConfig) String() string {
	if !BoolVal(c.Enabled) {
		return ""
	}

	if c.Password != nil {
		return fmt.Sprintf("%s:%s", StringVal(c.Username), StringVal(c.Password))
	}

	return StringVal(c.Username)
}
