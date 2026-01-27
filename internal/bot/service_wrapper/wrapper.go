package service_wrapper

import (
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"github.com/unspokenteam/golang-tg-dbot/pkg/utils"
)

type Services struct {
	// db queries
	AppViper       *viper.Viper
	CommandsViper  *viper.Viper
	RateLimitCache *redis.Client
}

func (services *Services) Init() error {
	services.AppViper = viper.New()
	services.AppViper.SetConfigType("env")
	if utils.IsEnvProduction() {
		services.AppViper.SetConfigFile(".env")
	} else {
		services.AppViper.SetConfigFile("example.env")
	}
	if err := services.AppViper.ReadInConfig(); err != nil {
		log.Fatal("Failed to read config:", err)
	}

	services.CommandsViper = viper.New()
	services.CommandsViper.SetConfigType("yaml")
	if utils.IsEnvProduction() {
		services.CommandsViper.SetConfigFile("commands.yaml")
	} else {
		services.CommandsViper.SetConfigFile("configs/bot/commands.yaml")
	}
	if err := services.CommandsViper.ReadInConfig(); err != nil {
		log.Fatal("Failed to read config:", err)
	}

	services.RateLimitCache = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf(
			"%s:%d",
			services.AppViper.GetString("REDIS_HOST"),
			services.AppViper.GetInt("REDIS_PORT")),
		DB: 0,
	})

	return nil
}
