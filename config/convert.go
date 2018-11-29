package config

import (
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/consul-template/signals"
)

func Bool(b bool) *bool {
	return &b
}

func BoolVal(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func BoolGoString(b *bool) string {
	if b == nil {
		return "(*bool)(nil)"
	}
	return fmt.Sprintf("%t", *b)
}

func BoolPresent(b *bool) bool {
	if b == nil {
		return false
	}
	return true
}

func FileMode(o os.FileMode) *os.FileMode {
	return &o
}

func FileModeVal(o *os.FileMode) os.FileMode {
	if o == nil {
		return 0
	}
	return *o
}

func FileModeGoString(o *os.FileMode) string {
	if o == nil {
		return "(*os.FileMode)(nil)"
	}
	return fmt.Sprintf("%q", *o)
}

func FileModePresent(o *os.FileMode) bool {
	if o == nil {
		return false
	}
	return *o != 0
}

func Int(i int) *int {
	return &i
}

func IntVal(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func IntGoString(i *int) string {
	if i == nil {
		return "(*int)(nil)"
	}
	return fmt.Sprintf("%d", *i)
}

func IntPresent(i *int) bool {
	if i == nil {
		return false
	}
	return *i != 0
}

func Signal(s os.Signal) *os.Signal {
	return &s
}

func SignalVal(s *os.Signal) os.Signal {
	if s == nil {
		return (os.Signal)(nil)
	}
	return *s
}

func SignalGoString(s *os.Signal) string {
	if s == nil {
		return "(*os.Signal)(nil)"
	}
	if *s == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%q", *s)
}

func SignalPresent(s *os.Signal) bool {
	if s == nil {
		return false
	}
	return *s != signals.SIGNIL
}

func String(s string) *string {
	return &s
}

func StringVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func StringGoString(s *string) string {
	if s == nil {
		return "(*string)(nil)"
	}
	return fmt.Sprintf("%q", *s)
}

func StringPresent(s *string) bool {
	if s == nil {
		return false
	}
	return *s != ""
}

func TimeDuration(t time.Duration) *time.Duration {
	return &t
}

func TimeDurationVal(t *time.Duration) time.Duration {
	if t == nil {
		return time.Duration(0)
	}
	return *t
}

func TimeDurationGoString(t *time.Duration) string {
	if t == nil {
		return "(*time.Duration)(nil)"
	}
	return fmt.Sprintf("%s", t)
}

func TimeDurationPresent(t *time.Duration) bool {
	if t == nil {
		return false
	}
	return *t != 0
}
