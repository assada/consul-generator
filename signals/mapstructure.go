package signals

import (
	"reflect"

	"github.com/mitchellh/mapstructure"
)

func StringToSignalFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}

		if t.String() != "os.Signal" {
			return data, nil
		}

		if data == nil || data.(string) == "" {
			return SIGNIL, nil
		}

		return Parse(data.(string))
	}
}
