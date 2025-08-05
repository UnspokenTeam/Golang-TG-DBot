package app

import (
	"configs"
	"context"
	"fmt"
	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/valyala/fasthttp"
	"logger"
	"middlewares"
	"os"
	"time"
)

var (
	bot       *telego.Bot
	updatesCh <-chan telego.Update
	err       error
	Done      chan struct{}
)

func initBotInstance(appCtx context.Context, token string, isDev bool) *telego.Bot {
	var loggerOpt telego.BotOption
	if isDev {
		loggerOpt = telego.WithDefaultDebugLogger()
	} else {
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
	return bot
}

func waitForGracefulShutdown(funcCtx context.Context, server *fasthttp.Server, ch <-chan telego.Update, hnd *th.BotHandler) {
	defer middlewares.ShutdownQueue()
	<-funcCtx.Done()
	logger.LogInfo("Stopping...", "gracefulShutdown", nil)

	stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second*30)
	defer stopCancel()

	if env == "PRODUCTION" {
		_ = server.ShutdownWithContext(stopCtx)
		logger.LogInfo("Server done...", "gracefulShutdown", nil)
	}

	for len(ch) > 0 || len(middlewares.MessageQueue) > 0 {
		select {
		case <-stopCtx.Done():
			break
		case <-time.After(time.Microsecond * 100):
			// Continue
		}
	}
	logger.LogInfo("Webhook done...", "gracefulShutdown", nil)

	_ = hnd.StopWithContext(stopCtx)
	logger.LogInfo("Bot handler done...", "gracefulShutdown", nil)

	close(Done)
}

func Run(env string, appCtx context.Context) {
	Done = make(chan struct{})
	jsonifyStack := false

	srv := &fasthttp.Server{}

	switch env {
	case "DEVELOPMENT":
		initBotInstance(appCtx, os.Getenv("DEV_TOKEN"), true)
		updatesCh, _ = bot.UpdatesViaLongPolling(appCtx, nil)

	case "PRODUCTION":
		prodConfig := configs.GetProdConfig()
		initBotInstance(appCtx, prodConfig.ProdToken, false)

		_ = bot.SetWebhook(appCtx, &telego.SetWebhookParams{
			URL:         fmt.Sprintf("https://%s/%s", prodConfig.CaddyDomain, bot.Token()),
			SecretToken: bot.SecretToken(),
		})

		info, _ := bot.GetWebhookInfo(appCtx)
		logger.LogInfo(fmt.Sprintf("Webhook Info: %+v\n", info), "webhookSetup", nil)

		updatesCh, _ = bot.UpdatesViaWebhook(
			appCtx,
			telego.WebhookFastHTTP(srv, "/bot", bot.SecretToken()),
			telego.WithWebhookBuffer(prodConfig.BufferSize),
		)

		go func() { _ = srv.ListenAndServe(fmt.Sprintf(":%d", prodConfig.AppPort)) }()
		jsonifyStack = true

	default:
		logger.LogFatal(fmt.Sprintf("Unknown env GO_ENV=%q", env), "configuring", nil)
	}

	logger.InitLogger(bot.Token(), bot.SecretToken(), jsonifyStack)
	configs.LoadBotCommands()
	middlewares.InitQueue(appCtx, bot)
	handler, _ := th.NewBotHandler(bot, updatesCh)
	handler.Use(middlewares.UserFilterMiddleware)
	InjectTelegoHandlers(handler)
	go waitForGracefulShutdown(appCtx, srv, updatesCh, handler)
	go func() { _ = handler.Start() }()
	logger.LogInfo("Bot started", "configuring", nil)
	go HealthCheckWithRestart(bot, appCtx)

	<-Done
	logger.LogInfo("Stopping done", "gracefulShutdown", nil)
}
