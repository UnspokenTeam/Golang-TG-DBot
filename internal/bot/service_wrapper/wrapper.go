package service_wrapper

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"github.com/unspokenteam/golang-tg-dbot/internal/configs"
	"github.com/unspokenteam/golang-tg-dbot/internal/db"
	"github.com/unspokenteam/golang-tg-dbot/internal/logger"
	"github.com/unspokenteam/golang-tg-dbot/pkg/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type Services struct {
	// db queries
	AppViper       *viper.Viper
	CommandsViper  *viper.Viper
	RateLimitCache *redis.Client
	TelegoLogger   *logger.TelegoLogger
	PostgresClient *db.Client
	Tracer         trace.Tracer
}

func (services *Services) Init(ctx context.Context) *Services {
	services.Tracer = otel.Tracer("my-bot")
	services.TelegoLogger = logger.SetupLogger("GoLang TG D-Bot")

	services.AppViper = viper.New()
	services.AppViper.SetConfigType("env")
	if utils.IsEnvProduction() {
		services.AppViper.SetConfigFile(".env")
	} else {
		services.AppViper.SetConfigFile("example.env")
	}
	if err := services.AppViper.ReadInConfig(); err != nil {
		logger.Fatal("Failed to read .env config: %v", err)
	}

	services.TelegoLogger.WithReplacer(
		strings.NewReplacer(
			services.AppViper.GetString("PROD_TOKEN"), "PROD_TOKEN",
			services.AppViper.GetString("DEV_TOKEN"), "DEV_TOKEN",
		),
	)

	services.CommandsViper = viper.New()
	services.CommandsViper.SetConfigType("yaml")
	if utils.IsEnvProduction() {
		services.CommandsViper.SetConfigFile("commands.yaml")
	} else {
		services.CommandsViper.SetConfigFile("configs/bot/commands.yaml")
	}
	if err := services.CommandsViper.ReadInConfig(); err != nil {
		logger.Fatal("Failed to read yaml config: %v", err)
	}

	services.RateLimitCache = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf(
			"%s:%d",
			services.AppViper.GetString("REDIS_HOST"),
			services.AppViper.GetInt("REDIS_PORT")),
		DB: 0,
	})

	postgresCfg := configs.LoadConfig(services.AppViper, configs.PostgresConfig{})

	if client, err := db.CreateConnection(&postgresCfg, ctx); err != nil {
		logger.Fatal("Failed to connect to postgres: %v", err)
	} else {
		services.PostgresClient = client
	}

	slog.Info("Services configured")
	return services
}
