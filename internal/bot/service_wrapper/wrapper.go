package service_wrapper

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/channels"
	"github.com/unspokenteam/golang-tg-dbot/internal/configs"
	"github.com/unspokenteam/golang-tg-dbot/internal/db/querier"
	"github.com/unspokenteam/golang-tg-dbot/internal/logger"
	"github.com/unspokenteam/golang-tg-dbot/pkg/utils"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/time/rate"
)

type Services struct {
	AppViper         *viper.Viper
	CommandsViper    *viper.Viper
	RateLimitCache   *redis.Client
	TelegoLogger     *logger.TelegoLogger
	PostgresClient   *querier.DbClient
	TgApiRateLimiter *rate.Limiter
	ConfigCache      *configs.ConfigCache
	Tracer           trace.Tracer
	Meter            metric.Meter
}

func (services *Services) Init(ctx context.Context) *Services {
	channels.InitChannels()
	services.Tracer = otel.Tracer("telegram-bot")
	services.Meter = otel.Meter("telegram-bot")
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
	services.ConfigCache = configs.NewConfigCache(services.CommandsViper)

	services.RateLimitCache = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf(
			"%s:%d",
			services.AppViper.GetString("REDIS_HOST"),
			services.AppViper.GetInt("REDIS_PORT")),
		DB: 0,
	})

	postgresCfg := configs.LoadConfig(services.AppViper, configs.PostgresConfig{})
	if utils.IsEnvDevelopment() && os.Getenv("IS_DOCKER") == "TRUE" {
		postgresCfg.Host = services.AppViper.GetString("POSTGRES_INTERNAL_HOST")
	}

	if client, err := querier.CreateClient(&postgresCfg, ctx, services.Tracer); err != nil {
		logger.Fatal("Failed to connect to postgres: %v", err)
	} else {
		services.PostgresClient = client
	}

	rps := services.AppViper.GetInt("RPS_LIMIT")
	services.TgApiRateLimiter = rate.NewLimiter(
		rate.Every(time.Second/time.Duration(rps)), rps)

	slog.Info("Services configured")
	return services
}
