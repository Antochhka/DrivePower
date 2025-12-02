package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultConfigPathEnv = "CONFIG_FILE"

// LoadConfig hydrates the provided struct pointer with values from YAML config file (optional)
// and overrides them with environment variables. Nested structs are supported via automatic
// ENV key generation (PARENT_CHILD) or explicit `env:"CUSTOM_KEY"` struct tags.
func LoadConfig(target interface{}) error {
	if target == nil {
		return errors.New("config: target is nil")
	}

	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return errors.New("config: target must be pointer to struct")
	}

	if path := os.Getenv(defaultConfigPathEnv); path != "" {
		if err := loadFromFile(path, target); err != nil {
			return err
		}
	}

	return populateFromEnv(val.Elem(), "")
}

func loadFromFile(path string, target interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("config: read file: %w", err)
	}

	if err := yaml.Unmarshal(data, target); err != nil {
		return fmt.Errorf("config: decode yaml: %w", err)
	}

	return nil
}

func populateFromEnv(v reflect.Value, prefix string) error {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		fieldVal := v.Field(i)
		fieldType := t.Field(i)

		if !fieldVal.CanSet() {
			continue
		}

		if fieldType.Anonymous {
			if err := populateFromEnv(fieldVal, prefix); err != nil {
				return err
			}
			continue
		}

		rawKey := fieldType.Tag.Get("env")
		if rawKey == "-" {
			continue
		}

		var envKey string
		if rawKey != "" {
			envKey = normalizeKey("", rawKey)
		} else {
			envKey = normalizeKey(prefix, fieldType.Name)
		}

		if fieldVal.Kind() == reflect.Struct {
			if err := populateFromEnv(fieldVal, envKey); err != nil {
				return err
			}
			continue
		}

		if val, ok := os.LookupEnv(envKey); ok {
			if err := assign(fieldVal, val); err != nil {
				return fmt.Errorf("config: parse %s: %w", envKey, err)
			}
		}
	}
	return nil
}

func normalizeKey(prefix, key string) string {
	key = strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
	if prefix == "" {
		return key
	}
	return fmt.Sprintf("%s_%s", prefix, key)
}

func assign(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Bool:
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(parsed)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(value, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsed, err := strconv.ParseUint(value, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(parsed)
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(value, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetFloat(parsed)
	default:
		return fmt.Errorf("unsupported field type %s", field.Type().String())
	}
	return nil
}
