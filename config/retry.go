package config

import (
	"fmt"
	"math"
	"time"
)

const (
	DefaultRetryAttempts   = 12
	DefaultRetryBackoff    = 250 * time.Millisecond
	DefaultRetryMaxBackoff = 1 * time.Minute
)

type RetryFunc func(int) (bool, time.Duration)

type RetryConfig struct {
	Attempts   *int
	Backoff    *time.Duration
	MaxBackoff *time.Duration `mapstructure:"max_backoff"`
	Enabled    *bool
}

func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{}
}

func (c *RetryConfig) Copy() *RetryConfig {
	if c == nil {
		return nil
	}

	var o RetryConfig

	o.Attempts = c.Attempts

	o.Backoff = c.Backoff

	o.MaxBackoff = c.MaxBackoff

	o.Enabled = c.Enabled

	return &o
}

func (c *RetryConfig) Merge(o *RetryConfig) *RetryConfig {
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

	if o.Attempts != nil {
		r.Attempts = o.Attempts
	}

	if o.Backoff != nil {
		r.Backoff = o.Backoff
	}

	if o.MaxBackoff != nil {
		r.MaxBackoff = o.MaxBackoff
	}

	if o.Enabled != nil {
		r.Enabled = o.Enabled
	}

	return r
}

func (c *RetryConfig) RetryFunc() RetryFunc {
	return func(retry int) (bool, time.Duration) {
		if !BoolVal(c.Enabled) {
			return false, 0
		}

		if IntVal(c.Attempts) > 0 && retry > IntVal(c.Attempts)-1 {
			return false, 0
		}

		baseSleep := TimeDurationVal(c.Backoff)
		maxSleep := TimeDurationVal(c.MaxBackoff)

		if maxSleep > 0 {
			attemptsTillMaxBackoff := int(math.Log2(maxSleep.Seconds() / baseSleep.Seconds()))
			if retry > attemptsTillMaxBackoff {
				return true, maxSleep
			}
		}

		base := math.Pow(2, float64(retry))
		sleep := time.Duration(base) * baseSleep

		return true, sleep
	}
}

func (c *RetryConfig) Finalize() {
	if c.Attempts == nil {
		c.Attempts = Int(DefaultRetryAttempts)
	}

	if c.Backoff == nil {
		c.Backoff = TimeDuration(DefaultRetryBackoff)
	}

	if c.MaxBackoff == nil {
		c.MaxBackoff = TimeDuration(DefaultRetryMaxBackoff)
	}

	if c.Enabled == nil {
		c.Enabled = Bool(true)
	}
}

func (c *RetryConfig) GoString() string {
	if c == nil {
		return "(*RetryConfig)(nil)"
	}

	return fmt.Sprintf("&RetryConfig{"+
		"Attempts:%s, "+
		"Backoff:%s, "+
		"MaxBackoff:%s, "+
		"Enabled:%s"+
		"}",
		IntGoString(c.Attempts),
		TimeDurationGoString(c.Backoff),
		TimeDurationGoString(c.MaxBackoff),
		BoolGoString(c.Enabled),
	)
}
