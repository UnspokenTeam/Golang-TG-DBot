package env_loader

import (
	"errors"
	"fmt"
	"logger"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type envType int

const (
	envFile envType = iota
	environment
)

var bitSizes = map[reflect.Kind]int{
	reflect.Int8:    8,
	reflect.Int16:   16,
	reflect.Int32:   32,
	reflect.Int64:   64,
	reflect.Uint8:   8,
	reflect.Uint16:  16,
	reflect.Uint32:  32,
	reflect.Uint64:  64,
	reflect.Float64: 64,
	reflect.Float32: 32,
}

var arrayRegexp = regexp.MustCompile("\\[(?P<inside>(?:[^\\[\\]\\,]+\\,)*(?:[^\\[\\]\\,]+){1})\\]")

type Env struct {
	data map[string]string
	t    envType
}

func (env *Env) GetIntoReflectValue(value reflect.Value, key string) error {
	if value.Kind() != reflect.Pointer {
		logger.LogFatal("value should be of pointer type", "envLoading", nil)
	}

	value = value.Elem()

	if !value.CanSet() {
		return errors.New(fmt.Sprintf("can't set value of kind %s as this field should be exported", value.Type().Kind()))
	}

	switch env.t {
	case envFile:
		envInterface, err := env.getInterfaceValueFromEnvFile(value.Type(), key)
		if err != nil {
			return err
		}
		envValue := reflect.ValueOf(envInterface)
		if envValue.Type() != value.Type() {
			value.Set(envValue.Convert(value.Type()))
			break
		}
		value.Set(reflect.ValueOf(envInterface))
		break
	case environment:
		envInterface, err := env.getInterfaceValueFromEnvironment(value.Type(), key)
		if err != nil {
			return err
		}
		envValue := reflect.ValueOf(envInterface)
		if envValue.Type() != value.Type() {
			value.Set(envValue.Convert(value.Type()))
			break
		}
		value.Set(envValue)
		break
	}

	return nil
}

func (env *Env) getInterfaceValueFromEnvironment(t reflect.Type, key string) (interface{}, error) {
	envValue, exists := os.LookupEnv(key)
	if !exists {
		return nil, errors.New(fmt.Sprintf("can't find value with key %s in the environment", key))
	}

	envInterface, conversionError := convertStringToReflectType(envValue, t)
	if conversionError != nil {
		return nil, conversionError
	}

	return envInterface, nil
}

func (env *Env) getInterfaceValueFromEnvFile(t reflect.Type, key string) (interface{}, error) {
	envValue, exists := env.data[key]
	if !exists {
		return nil, errors.New(fmt.Sprintf("can't find value with key %s in the environment", key))
	}

	envInterface, conversionError := convertStringToReflectType(envValue, t)
	if conversionError != nil {
		return nil, conversionError
	}

	return envInterface, nil
}

func CreateEnvFromFile(path string) (*Env, error) {
	data, err := DeserializeEnvFile(path)
	if err != nil {
		return nil, err
	}

	return &Env{
		data: data,
		t:    envFile,
	}, nil
}

func CreateEnvFromEnvironment() *Env {
	return &Env{
		t: environment,
	}
}

func convertStringToReflectType(v string, t reflect.Type) (interface{}, error) {
	switch t.Kind() {
	case reflect.Int:
		return strconv.Atoi(v)
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.ParseInt(v, 10, bitSizes[t.Kind()])
	case reflect.Float64, reflect.Float32:
		return strconv.ParseFloat(v, bitSizes[t.Kind()])
	case reflect.Uint:
		ui64, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return nil, err
		}
		return uint(ui64), nil
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.ParseUint(v, 10, bitSizes[t.Kind()])
	case reflect.Bool:
		return strconv.ParseBool(v)
	case reflect.String:
		return v, nil
	case reflect.Slice:
		return deserializeSliceFromString(v, t)
	default:
		return nil, errors.New(fmt.Sprintf("can't convert string to %s", t.Kind()))
	}

}

func deserializeSliceFromString(v string, t reflect.Type) (interface{}, error) {
	elemType := t.Elem()

	switch elemType.Kind() {
	case reflect.Array, reflect.Struct, reflect.Pointer, reflect.Uintptr, reflect.UnsafePointer, reflect.Map:
		return nil, errors.New(fmt.Sprintf("value inside array is of unsupported type %s", elemType.Kind()))
	default:
		break
	}

	sliceValue := reflect.MakeSlice(t, 0, 0)

	index := arrayRegexp.SubexpIndex("inside")
	matches := arrayRegexp.FindStringSubmatch(v)

	if index >= len(matches) {
		return nil, errors.New("array value does not match the style or is empty")
	}

	inside := matches[index]
	for _, item := range strings.Split(inside, ",") {
		trimmed := strings.TrimSpace(item)
		itemV, conversionError := convertStringToReflectType(trimmed, elemType)
		if conversionError != nil {
			return nil, conversionError
		}
		sliceValue = reflect.Append(sliceValue, reflect.ValueOf(itemV))
	}

	return sliceValue.Interface(), nil
}
