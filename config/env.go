package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type EnvConfig struct {
	Blacklist []string `mapstructure:"blacklist"`
	Custom    []string `mapstructure:"custom"`
	Pristine  *bool    `mapstructure:"pristine"`
	Whitelist []string `mapstructure:"whitelist"`
}

func DefaultEnvConfig() *EnvConfig {
	return &EnvConfig{}
}

func (c *EnvConfig) Copy() *EnvConfig {
	if c == nil {
		return nil
	}

	var o EnvConfig

	if c.Blacklist != nil {
		o.Blacklist = append([]string{}, c.Blacklist...)
	}

	if c.Custom != nil {
		o.Custom = append([]string{}, c.Custom...)
	}

	o.Pristine = c.Pristine

	if c.Whitelist != nil {
		o.Whitelist = append([]string{}, c.Whitelist...)
	}

	return &o
}

func (c *EnvConfig) Merge(o *EnvConfig) *EnvConfig {
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

	if o.Blacklist != nil {
		r.Blacklist = append(r.Blacklist, o.Blacklist...)
	}

	if o.Custom != nil {
		r.Custom = append(r.Custom, o.Custom...)
	}

	if o.Pristine != nil {
		r.Pristine = o.Pristine
	}

	if o.Whitelist != nil {
		r.Whitelist = append(r.Whitelist, o.Whitelist...)
	}

	return r
}

func (c *EnvConfig) Env() []string {
	if BoolVal(c.Pristine) {
		if len(c.Custom) > 0 {
			return c.Custom
		}
		return []string{}
	}

	environ := os.Environ()
	keys := make([]string, len(environ))
	env := make(map[string]string, len(environ))
	for i, v := range environ {
		list := strings.SplitN(v, "=", 2)
		keys[i] = list[0]
		env[list[0]] = list[1]
	}

	anyGlobMatch := func(s string, patterns []string) bool {
		for _, pattern := range patterns {
			if matched, _ := filepath.Match(pattern, s); matched {
				return true
			}
		}
		return false
	}

	if len(c.Whitelist) > 0 {
		newKeys := make([]string, 0, len(keys))
		for _, k := range keys {
			if anyGlobMatch(k, c.Whitelist) {
				newKeys = append(newKeys, k)
			}
		}
		keys = newKeys
	}

	if len(c.Blacklist) > 0 {
		newKeys := make([]string, 0, len(keys))
		for _, k := range keys {
			if !anyGlobMatch(k, c.Blacklist) {
				newKeys = append(newKeys, k)
			}
		}
		keys = newKeys
	}

	finalEnv := make([]string, 0, len(keys)+len(c.Custom))
	for _, k := range keys {
		finalEnv = append(finalEnv, k+"="+env[k])
	}

	finalEnv = append(finalEnv, c.Custom...)

	return finalEnv
}

func (c *EnvConfig) Finalize() {
	if c.Blacklist == nil {
		c.Blacklist = []string{}
	}

	if c.Custom == nil {
		c.Custom = []string{}
	}

	if c.Pristine == nil {
		c.Pristine = Bool(false)
	}

	if c.Whitelist == nil {
		c.Whitelist = []string{}
	}
}

func (c *EnvConfig) GoString() string {
	if c == nil {
		return "(*EnvConfig)(nil)"
	}

	return fmt.Sprintf("&EnvConfig{"+
		"Blacklist:%v, "+
		"Custom:%v, "+
		"Pristine:%s, "+
		"Whitelist:%v"+
		"}",
		c.Blacklist,
		c.Custom,
		BoolGoString(c.Pristine),
		c.Whitelist,
	)
}
