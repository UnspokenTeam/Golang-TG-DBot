package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/channels"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	configs "github.com/unspokenteam/golang-tg-dbot/internal/config"
	"github.com/unspokenteam/golang-tg-dbot/internal/middlewares"
	"github.com/unspokenteam/golang-tg-dbot/pkg/utils"
	"github.com/valyala/fasthttp"
	"go.opentelemetry.io/otel"
)

var (
	services *service_wrapper.Services
)

func initBotInstance(ctx context.Context, token string) *telego.Bot {
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
		telego.WithHealthCheck(ctx),
		telego.WithLogger(services.TelegoLogger),
	)
	if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Error while creating bot instance: %v", err))
		channels.ShutdownChannel <- struct{}{}
	}
	utils.InitUtils(bot)

	return bot
}

func Run(appCtx context.Context, cancelFunc context.CancelFunc) {
	var (
		bot       *telego.Bot
		updatesCh <-chan telego.Update
		srv       = &fasthttp.Server{}
	)

	servicesInstance := service_wrapper.Services{}
	services = servicesInstance.Init()

	tracer := otel.Tracer("my-bot")
	ctx, rootSpan := tracer.Start(appCtx, "Main app span")
	defer rootSpan.End()

	switch utils.GetEnv() {
	case utils.DEVELOPMENT:
		var channelErr error
		bot = initBotInstance(ctx, services.AppViper.GetString("DEV_TOKEN"))
		updatesCh, channelErr = bot.UpdatesViaLongPolling(ctx, nil)
		if channelErr != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("Channel error: %v", channelErr))
		}

	case utils.PRODUCTION:
		prodConfig := configs.LoadConfig(services.AppViper, configs.ProdBotConfig{})
		bot = initBotInstance(ctx, prodConfig.ProdToken)

		webhookPath := "/" + bot.Token()
		webhookURL := fmt.Sprintf("https://api.%s%s", prodConfig.CaddyDomain, webhookPath)

		info, getWebhookInfoErr := bot.GetWebhookInfo(ctx)
		if getWebhookInfoErr != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("Get webhook info error: %v", getWebhookInfoErr), "info", info)
		}

		if info.URL != webhookURL {
			var err error
			setWebhookErr := bot.SetWebhook(ctx, &telego.SetWebhookParams{
				URL:         webhookURL,
				SecretToken: bot.SecretToken(),
			})
			if setWebhookErr != nil {
				slog.ErrorContext(ctx, fmt.Sprintf("Set webhook error: %v", setWebhookErr))
			}

			info, err = bot.GetWebhookInfo(ctx)
			if err != nil {
				slog.ErrorContext(ctx, fmt.Sprintf("Get final webhook error: %v", err), "info", info)
			}
		}
		slog.InfoContext(ctx, fmt.Sprintf("Webhook Info: %+v\n", info))

		var channelErr error
		updatesCh, channelErr = bot.UpdatesViaWebhook(
			ctx,
			telego.WebhookFastHTTP(srv, webhookPath, bot.SecretToken()),
			telego.WithWebhookBuffer(prodConfig.BufferSize),
		)
		if channelErr != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("Channel error: %v", channelErr))
		}

	default:
		slog.ErrorContext(ctx, fmt.Sprintf("Unknown env GO_ENV=%s", utils.GetEnv()))
		channels.ShutdownChannel <- struct{}{}
	}

	handler, handlerErr := th.NewBotHandler(bot, updatesCh)
	if handlerErr != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Handler creating error: %v", handlerErr))
	}
	filterWrapper := middlewares.UserFilterWrapper(services)
	handler.Use(filterWrapper)
	configureHandlers(ctx, handler)
	runComponentsWithGracefulShutdown(ctx, cancelFunc, bot, handler, srv)
}
