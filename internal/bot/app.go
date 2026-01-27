package app

import (
	"context"
	"fmt"
	"time"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	configs "github.com/unspokenteam/golang-tg-dbot/internal/config"
	"github.com/unspokenteam/golang-tg-dbot/internal/middlewares"
	"github.com/unspokenteam/golang-tg-dbot/pkg/logger"
	"github.com/unspokenteam/golang-tg-dbot/pkg/utils"
	"github.com/valyala/fasthttp"
)

var (
	services *service_wrapper.Services
)

func initBotInstance(appCtx context.Context, token string) *telego.Bot {
	var (
		loggerOpt telego.BotOption
		err       error
	)
	if utils.IsEnvDevelopment() {
		loggerOpt = telego.WithDefaultDebugLogger()
	} else {
		//todo: переписать логгер на slog
		loggerOpt = telego.WithLogger(logger.TelegoLogger{})
	}
	bot, err := telego.NewBot(
		token,
		telego.WithAPICaller(
			&ta.RetryCaller{
				Caller:       ta.DefaultFastHTTPCaller,
				MaxAttempts:  4,
				ExponentBase: 2,
				StartDelay:   time.Millisecond * 10,
				MaxDelay:     time.Second,
			}),
		telego.WithHealthCheck(appCtx),
		loggerOpt,
	)
	if err != nil {
		logger.LogFatal(err.Error(), "configuring", nil)
	}
	utils.InitUtils(bot)

	return bot
}

func Run(appCtx context.Context, cancelFunc context.CancelFunc) {
	servicesInstance := service_wrapper.Services{}
	serviceInitErr := servicesInstance.Init()
	if serviceInitErr != nil {
		return
	}
	services = &servicesInstance

	var (
		bot       *telego.Bot
		updatesCh <-chan telego.Update
	)
	srv := &fasthttp.Server{}

	switch utils.GetEnv() {
	case utils.DEVELOPMENT:
		bot = initBotInstance(appCtx, services.AppViper.GetString("DEV_TOKEN"))
		updatesCh, _ = bot.UpdatesViaLongPolling(appCtx, nil)

	case utils.PRODUCTION:
		prodConfig := configs.LoadConfig(services.AppViper, configs.ProdBotConfig{})
		bot = initBotInstance(appCtx, prodConfig.ProdToken)

		webhookPath := "/" + bot.Token()
		webhookURL := fmt.Sprintf("https://api.%s%s", prodConfig.CaddyDomain, webhookPath)

		info, _ := bot.GetWebhookInfo(appCtx)
		if info.URL != webhookURL {
			_ = bot.SetWebhook(appCtx, &telego.SetWebhookParams{
				URL:         webhookURL,
				SecretToken: bot.SecretToken(),
			})
			info, _ = bot.GetWebhookInfo(appCtx)
		}
		logger.LogInfo(fmt.Sprintf("Webhook Info: %+v\n", info), "webhookSetup", nil)

		updatesCh, _ = bot.UpdatesViaWebhook(
			appCtx,
			telego.WebhookFastHTTP(srv, webhookPath, bot.SecretToken()),
			telego.WithWebhookBuffer(prodConfig.BufferSize),
		)

	default:
		logger.LogFatal(fmt.Sprintf("Unknown env GO_ENV=%q", utils.GetEnv()), "configuring", nil)
	}

	//logger.InitLogger(bot.Token(), bot.SecretToken(), true)
	handler, _ := th.NewBotHandler(bot, updatesCh)
	filterWrapper := middlewares.UserFilterWrapper(services)
	handler.Use(filterWrapper)
	configureHandlers(handler)
	runComponentsWithGracefulShutdown(appCtx, cancelFunc, bot, handler, srv)
}
