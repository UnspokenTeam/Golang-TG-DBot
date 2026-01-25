package utils

import "os"

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
