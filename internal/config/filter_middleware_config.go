package configs

import (
	"fmt"

	"github.com/unspokenteam/golang-tg-dbot/pkg/env_loader"
	"github.com/unspokenteam/golang-tg-dbot/pkg/logger"

	"github.com/redis/go-redis/v9"
)

type FilterMiddlewareConfig struct {
	RedisHost    string `env:"REDIS_HOST"`
	RedisPort    int16  `env:"REDIS_PORT"`
	SpamCooldown int64  `env:"SPAM_COOLDOWN"`
	MuteCooldown int16  `env:"MUTE_COOLDOWN"`
}

var (
	MiddlewareConfig FilterMiddlewareConfig
	Rdb              *redis.Client
)

func InitMiddlewareConfig() {
	envLoader := env_loader.CreateLoaderFromEnv()
	if err := envLoader.LoadDataIntoStruct(&MiddlewareConfig); err != nil {
		logger.LogFatal(err.Error(), "configuring", nil)
	}
	Rdb = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", MiddlewareConfig.RedisHost, MiddlewareConfig.RedisPort),
		DB:   0,
	})
}
