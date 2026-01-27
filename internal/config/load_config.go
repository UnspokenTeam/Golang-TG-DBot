package configs

import (
	"log"

	"github.com/spf13/viper"
)

func LoadConfig[T any](v *viper.Viper, cfg T) T {
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("Failed to unmarshal config: %v\n", err)
	}

	return cfg
}
