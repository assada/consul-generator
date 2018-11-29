package config

import (
	"log"
	"os"
	"reflect"
	"strconv"

	"github.com/mitchellh/mapstructure"
)

func StringToFileModeFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(os.FileMode(0)) {
			return data, nil
		}

		v, err := strconv.ParseUint(data.(string), 8, 12)
		if err != nil {
			return data, err
		}
		return os.FileMode(v), nil
	}
}

func ConsulStringToStructFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if t == reflect.TypeOf(ConsulConfig{}) && f.Kind() == reflect.String {
			log.Println("[WARN] consul now accepts a stanza instead of a string. " +
				"Update your configuration files and change consul = \"\" to " +
				"consul { } instead.")
			return &ConsulConfig{
				Address: String(data.(string)),
			}, nil
		}

		return data, nil
	}
}
