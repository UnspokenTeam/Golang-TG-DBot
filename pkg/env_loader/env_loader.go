package env_loader

import (
	"errors"
	"fmt"
	"reflect"
)

type Environment string

const (
	PRODUCTION  Environment = "production"
	DEVELOPMENT             = "development"
)

type Env struct {
	data map[string]string
}

func (e *Env) LoadData(environment Environment) error {
	filename := fmt.Sprintf(".env.%s", environment)

	handler := FileHandler{}

	if err := Open(filename, &handler); err != nil {
		return err
	}

	defer handler.Close()

	var err error
	err, e.data = handler.Read()

	return err
}

func (e *Env) GetValue(data interface{}) error {

	dataType := reflect.TypeOf(data)

	if dataType.Kind() != reflect.Pointer {
		return errors.New("data should be a pointer type")
	}

	dataType = dataType.Elem()

	switch dataType.Kind() {
	case reflect.Struct:
		_ = e.unmarshalStruct(dataType, data)
		break
	default:
		break
	}

	return nil
}

func (e *Env) unmarshalStruct(t reflect.Type, data interface{}) error {
	val := reflect.ValueOf(data)
	fieldCount := t.NumField()
	for i := range fieldCount {
		field := t.Field(i)
		fmt.Println(field.Name)
		val.Elem().FieldByName(field.Name).SetString("qwe")
	}
	return nil
}

func (e *Env) getFieldValue(data interface{}) {

}
