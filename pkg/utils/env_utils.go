package utils

import (
	"os"
	"path/filepath"
	"runtime"
)

var (
	PRODUCTION  = "PRODUCTION"
	DEVELOPMENT = "DEVELOPMENT"
)

func GetEnv() string {
	return os.Getenv("GO_ENV")
}

func IsEnvDevelopment() bool {
	return os.Getenv("GO_ENV") == DEVELOPMENT
}

func IsEnvProduction() bool {
	return os.Getenv("GO_ENV") == PRODUCTION
}

func GetDevEnvFileLocation() string {
	_, b, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(b), "../../")
	return filepath.Join(projectRoot, "example.env")
}
