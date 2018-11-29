package signals

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

var SIGNIL os.Signal = new(NilSignal)

var ValidSignals []string

func init() {
	valid := make([]string, 0, len(SignalLookup))
	for k := range SignalLookup {
		valid = append(valid, k)
	}
	sort.Strings(valid)
	ValidSignals = valid
}

func Parse(s string) (os.Signal, error) {
	sig, ok := SignalLookup[strings.ToUpper(s)]
	if !ok {
		return nil, fmt.Errorf("invalid signal %q - valid signals are %q",
			s, ValidSignals)
	}
	return sig, nil
}
