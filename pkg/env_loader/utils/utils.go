package env_loader

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

const envTagName = "env"
const envSeparator = "__"

func ParseTag(t reflect.Type, tField reflect.StructField) (*string, error) {
	tag, tagExists := tField.Tag.Lookup(envTagName)
	if !tagExists {
		return nil, nil
	}

	if tag == "" {
		return nil, errors.New(fmt.Sprintf("env tag on field %s is empty", tField.Name))
	}

	envVariableName := fmt.Sprintf("%s%s%s", t.Name(), envSeparator, tag)

	return &envVariableName, nil
}

func DeserializeEnvFile(path string) (map[string]string, error) {
	file := CreateFile(path)
	if fileReadingError := file.Read(); fileReadingError != nil {
		return nil, fileReadingError
	}
	stringData := file.GetData()

	return deserializeFromStringContents(stringData)
}

func deserializeFromStringContents(content *string) (map[string]string, error) {
	result := make(map[string]string)

	splitByNewLine := strings.Split(*content, "\n")
	for rowIdx, row := range splitByNewLine {
		if len(row) == 0 {
			continue
		}

		if !strings.Contains(row, "=") {
			return nil, errors.New(fmt.Sprintf("env file is malformed as it does not have = sign on the line %d", rowIdx+1))
		}

		splitByEqSign := strings.Split(row, "=")

		key, value := strings.TrimSpace(splitByEqSign[0]), strings.TrimSpace(splitByEqSign[1])
		if len(key) == 0 || len(value) == 0 {
			return nil, errors.New(fmt.Sprintf("env file is malformed as there is no key or value on the line %d", rowIdx+1))
		}

		if _, exists := result[key]; exists {
			return nil, errors.New(fmt.Sprintf("detected env key override as its a duplicate key %s on the line %d", key, rowIdx+1))
		}

		result[key] = value
	}

	return result, nil
}
