package configs

import (
	"github.com/spf13/viper"
	"github.com/unspokenteam/golang-tg-dbot/internal/logger"
)

func LoadConfig[T any](v *viper.Viper, cfg T) T {
	if err := v.Unmarshal(&cfg); err != nil {
		logger.Fatal("Failed to unmarshal config: %v\n", err)
	}

	return cfg
}
