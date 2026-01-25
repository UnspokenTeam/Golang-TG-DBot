package app

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
	th "github.com/mymmrac/telego/telegohandler"
	configs "github.com/unspokenteam/golang-tg-dbot/internal/config"
	"github.com/unspokenteam/golang-tg-dbot/internal/middlewares"
	"github.com/unspokenteam/golang-tg-dbot/pkg/logger"
	"github.com/unspokenteam/golang-tg-dbot/pkg/utils"
	"github.com/valyala/fasthttp"
)

var (
	bot       *telego.Bot
	updatesCh <-chan telego.Update
	err       error
)

func initBotInstance(appCtx context.Context, token string, isDev bool) {
	var loggerOpt telego.BotOption
	if isDev {
		loggerOpt = telego.WithDefaultDebugLogger()
	} else {
		//todo: переписать логгер на slog
		loggerOpt = telego.WithLogger(logger.TelegoLogger{})
	}
	bot, err = telego.NewBot(
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
}

func Run(appCtx context.Context, cancelFunc context.CancelFunc) {
	jsonifyStack := false

	srv := &fasthttp.Server{}

	switch utils.GetEnv() {
	case utils.DEVELOPMENT:
		initBotInstance(appCtx, os.Getenv("DEV_TOKEN"), true)
		updatesCh, _ = bot.UpdatesViaLongPolling(appCtx, nil)

	case utils.PRODUCTION:
		prodConfig := configs.GetProdConfig()
		initBotInstance(appCtx, prodConfig.ProdToken, false)

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

		jsonifyStack = true

	default:
		logger.LogFatal(fmt.Sprintf("Unknown env GO_ENV=%q", utils.GetEnv()), "configuring", nil)
	}

	logger.InitLogger(bot.Token(), bot.SecretToken(), jsonifyStack)
	configs.LoadBotCommands()
	handler, _ := th.NewBotHandler(bot, updatesCh)
	handler.Use(middlewares.UserFilterWrapper)
	InjectTelegoHandlers(handler)
	//todo: работа с конфигом
	runComponentsWithGracefulShutdown(appCtx, cancelFunc, bot, handler, srv, 8080)
}
