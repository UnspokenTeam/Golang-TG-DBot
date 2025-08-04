package app

import (
	"configs"
	"context"
	"fmt"
	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/valyala/fasthttp"
	"logger"
	"middlewares"
	"os"
	"os/signal"
	"time"
)

var (
	bot       *telego.Bot
	updatesCh <-chan telego.Update
	err       error
	Done      chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
)

func initBotInstance(ctx context.Context, token string, isDev bool) *telego.Bot {
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
		telego.WithHealthCheck(ctx),
		loggerOpt,
	)
	if err != nil {
		logger.LogFatal(err.Error(), "configuring", nil)
	}
	return bot
}

func Run(env string, innerCtx context.Context) {
	ctx, cancel = signal.NotifyContext(innerCtx, os.Interrupt)
	defer cancel()
	jsonifyStack := false

	srv := &fasthttp.Server{}

	switch env {
	case "DEVELOPMENT":
		initBotInstance(ctx, os.Getenv("DEV_TOKEN"), true)
		updatesCh, _ = bot.UpdatesViaLongPolling(ctx, nil)

	case "PRODUCTION":
		prodConfig := configs.GetProdConfig()
		initBotInstance(ctx, prodConfig.ProdToken, false)

		_ = bot.SetWebhook(ctx, &telego.SetWebhookParams{
			URL:         fmt.Sprintf("https://%s/%s", prodConfig.CaddyDomain, bot.Token()),
			SecretToken: bot.SecretToken(),
		})

		info, _ := bot.GetWebhookInfo(ctx)
		logger.LogInfo(fmt.Sprintf("Webhook Info: %+v\n", info), "webhookSetup", nil)

		updatesCh, _ = bot.UpdatesViaWebhook(
			ctx,
			telego.WebhookFastHTTP(srv, "/bot", bot.SecretToken()),
			telego.WithWebhookBuffer(prodConfig.BufferSize),
		)

		go func() { _ = srv.ListenAndServe(fmt.Sprintf(":%d", prodConfig.AppPort)) }()
		jsonifyStack = true

	default:
		logger.LogFatal(fmt.Sprintf("Неизвестная среда GO_ENV=%q", env), "configuring", nil)
	}

	logger.InitLogger(bot.Token(), bot.SecretToken(), jsonifyStack)
	configs.LoadBotCommands()
	middlewares.InitQueue(ctx, bot)
	defer middlewares.ShutdownQueue()

	handler, _ := th.NewBotHandler(bot, updatesCh)
	handler.Use(middlewares.UserFilterMiddleware)

	handler.Handle(func(thCtx *th.Context, update telego.Update) error {
		go func() {
			middlewares.MessageQueue <- tu.Message(
				tu.ID(update.Message.Chat.ID),
				fmt.Sprintf("Hello %s!", update.Message.From.FirstName),
			)
		}()
		return nil
	}, th.CommandEqual("start"))

	go func() {
		<-ctx.Done()
		logger.LogInfo("Stopping...", "gracefulShutdown", nil)

		stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second*30)
		defer stopCancel()

		if env == "PRODUCTION" {
			_ = srv.ShutdownWithContext(stopCtx)
			logger.LogInfo("Server done...", "gracefulShutdown", nil)
		}

		for len(updatesCh) > 0 || len(middlewares.MessageQueue) > 0 {
			select {
			case <-stopCtx.Done():
				break
			case <-time.After(time.Microsecond * 100):
				// Continue
			}
		}
		logger.LogInfo("Webhook done...", "gracefulShutdown", nil)

		_ = handler.StopWithContext(stopCtx)
		logger.LogInfo("Bot handler done...", "gracefulShutdown", nil)

		Done <- struct{}{}
	}()

	go func() { _ = handler.Start() }()
	logger.LogInfo("Bot started", "configuring", nil)

	go HealthCheckWithRestart(bot, innerCtx)

	<-Done
	logger.LogInfo("Stopping done", "gracefulShutdown", nil)
}
