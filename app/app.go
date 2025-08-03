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
	"log"
	"middlewares"
	"os"
	"os/signal"
	"time"
)

var (
	bot       *telego.Bot
	updatesCh <-chan telego.Update
	err       error
)

func initBotInstance(ctx context.Context, token string) *telego.Bot {
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
		telego.WithDefaultDebugLogger(),
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return bot
}

func Run(innerCtx context.Context) {
	env := os.Getenv("GO_ENV")

	ctx, cancel := signal.NotifyContext(innerCtx, os.Interrupt)
	defer cancel()

	done := make(chan struct{}, 1)
	srv := &fasthttp.Server{}

	switch env {
	case "DEVELOPMENT":
		initBotInstance(ctx, os.Getenv("DEV_TOKEN"))
		updatesCh, _ = bot.UpdatesViaLongPolling(ctx, nil)

	case "PRODUCTION":
		prodConfig := configs.GetProdConfig()
		initBotInstance(ctx, prodConfig.ProdToken)

		_ = bot.SetWebhook(ctx, &telego.SetWebhookParams{
			URL:         fmt.Sprintf("https://%s/%s", prodConfig.CaddyDomain, bot.Token()),
			SecretToken: bot.SecretToken(),
		})

		info, _ := bot.GetWebhookInfo(ctx)
		fmt.Printf("Webhook Info: %+v\n", info)

		updatesCh, _ = bot.UpdatesViaWebhook(
			ctx,
			telego.WebhookFastHTTP(srv, "/bot", bot.SecretToken()),
			telego.WithWebhookBuffer(prodConfig.BufferSize),
		)

		go func() { _ = srv.ListenAndServe(fmt.Sprintf(":%d", prodConfig.AppPort)) }()

	default:
		log.Fatalf("Неизвестная среда GO_ENV=%q", env)
	}

	configs.LoadBotCommands()
	middlewares.InitQueue(ctx, bot)
	log.Println("Bot started, waiting for updates...")

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
		fmt.Println("Stopping...")

		stopCtx, stopCancel := context.WithTimeout(context.Background(), time.Second*30)
		defer stopCancel()

		if env == "PRODUCTION" {
			_ = srv.ShutdownWithContext(stopCtx)
			fmt.Println("Server done")
		}

		for len(updatesCh) > 0 || len(middlewares.MessageQueue) > 0 {
			select {
			case <-stopCtx.Done():
				break
			case <-time.After(time.Microsecond * 100):
				// Continue
			}
		}
		fmt.Println("Webhook done")

		_ = handler.StopWithContext(stopCtx)
		fmt.Println("Bot handler done")

		done <- struct{}{}
	}()

	go func() { _ = handler.Start() }()

	go HealthCheckWithRestart(bot, innerCtx)

	<-done
	fmt.Println("Done")
}
