package env_loader

import (
	"fmt"
	"git
	"os"
/joho/godo
	"reflect"
	"strconv"
)

func init() {
	envPath := filepath.Join("..", "..", ".env")
	if _, err := os.Stat(envPath); err == nil {
		err = godotenv.Load(envPath)
		if err != nil {
			panic(fmt.Sprintf("Ошибка при загрузке .env файла: %v", err))
		}
	}
}

func getEnvOrPanic(typeName string, fieldName string) string {
	envVarName := fmt.Sprintf("%s__%s", typeName, fieldName)
	envVal, ok := os.LookupEnv(envVarName)
	if !ok {
		panic(fmt.Sprintf("Can't map env to struct %s, because %s key for field %s does not exist", typeName, envVarName, fieldName))
	}
	return envVal
}

func setValueToStruct(val *reflect.Value, stringVal string) {
	switch val.Type().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.Atoi(stringVal)
		if err != nil {
			panic(fmt.Sprintf("Can't convert string to int value"))
		}
		val.SetInt(int64(intVal))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		intVal, err := strconv.ParseUint(stringVal, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("Can't convert string to uint value"))
		}
		val.SetUint(intVal)
	case reflect.String:
		val.SetString(stringVal)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(stringVal)
		if err != nil {
			panic(fmt.Sprintf("Can't convert string to bool value"))
		}
		val.SetBool(boolVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(stringVal, 64)
		if err != nil {
			panic(fmt.Sprintf("Can't convert string to float value"))
		}
		val.SetFloat(floatVal)
	default:
		panic(fmt.Sprintf("Unsupported type %s", val.Type().Name()))
	}
}

func GetFromEnv[T any]() *T {
	obj := new(T)
	objValue := reflect.ValueOf(obj).Elem()
	objType := objValue.Type()
	typeName := objType.Name()
	for idx := 0; idx < objValue.NumField(); idx++ {
		typeField := objType.Field(idx)
		fieldName, ok := typeField.Tag.Lookup("env")
		if !ok {
			panic(fmt.Sprintf("Can't map env to struct, because field %s in %s struct does not have env tag attached to it", typeField.Name, typeName))
		}

		stringVal := getEnvOrPanic(typeName, fieldName)
		field := objValue.Field(idx)
		setValueToStruct(&field, stringVal)
	}

	return obj
}
