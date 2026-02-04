package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/configs"
	"github.com/unspokenteam/golang-tg-dbot/internal/logger"
	"github.com/unspokenteam/golang-tg-dbot/internal/middlewares"
	"github.com/unspokenteam/golang-tg-dbot/pkg/utils"
	"github.com/valyala/fasthttp"
)

var (
	services *service_wrapper.Services
)

func initBotInstance(ctx context.Context, token string) *telego.Bot {
	if bot, err := telego.NewBot(
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
	); err != nil {
		logger.Fatal(fmt.Sprintf("Error while creating bot instance: %v", err))
	} else {
		utils.InitUtils(bot)
		return bot
	}
	return nil
}

func Run(appCtx context.Context, cancelFunc context.CancelFunc) {
	var (
		bot       *telego.Bot
		updatesCh <-chan telego.Update
		srv       = &fasthttp.Server{}
	)

	servicesInstance := service_wrapper.Services{}
	services = servicesInstance.Init(appCtx)

	ctx, rootSpan := services.Tracer.Start(appCtx, "Main app span")
	defer rootSpan.End()

	switch utils.GetEnv() {
	case utils.DEVELOPMENT:
		var channelErr error
		bot = initBotInstance(ctx, services.AppViper.GetString("DEV_TOKEN"))
		if updatesCh, channelErr = bot.UpdatesViaLongPolling(ctx, nil); channelErr != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("Channel error: %v", channelErr))
		}

	case utils.PRODUCTION:
		prodConfig := configs.LoadConfig(services.AppViper, configs.ProdBotConfig{})
		bot = initBotInstance(ctx, prodConfig.ProdToken)

		webhookPath := "/" + bot.Token()
		webhookURL := fmt.Sprintf("https://api.%s%s", prodConfig.CaddyDomain, webhookPath)

		var (
			info              *telego.WebhookInfo
			getWebhookInfoErr error
		)

		if info, getWebhookInfoErr = bot.GetWebhookInfo(ctx); getWebhookInfoErr != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("Get webhook info error: %v", getWebhookInfoErr), "info", info)
		}

		if info.URL != webhookURL {
			if setWebhookErr := bot.SetWebhook(ctx, &telego.SetWebhookParams{
				URL:         webhookURL,
				SecretToken: bot.SecretToken(),
			}); setWebhookErr != nil {
				slog.ErrorContext(ctx, fmt.Sprintf("Set webhook error: %v", setWebhookErr))
			}

			if info, getWebhookInfoErr = bot.GetWebhookInfo(ctx); getWebhookInfoErr != nil {
				slog.ErrorContext(ctx, fmt.Sprintf("Get final webhook error: %v", getWebhookInfoErr), "info", info)
			}
		}
		slog.InfoContext(ctx, fmt.Sprintf("Webhook Info: %+v\n", info))

		var channelErr error
		if updatesCh, channelErr = bot.UpdatesViaWebhook(
			ctx,
			telego.WebhookFastHTTP(srv, webhookPath, bot.SecretToken()),
			telego.WithWebhookBuffer(prodConfig.BufferSize),
		); channelErr != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("Channel error: %v", channelErr))
		}

	default:
		logger.Fatal(fmt.Sprintf("Unknown env GO_ENV=%s", utils.GetEnv()))
		return
	}

	handler, handlerErr := th.NewBotHandler(bot, updatesCh)
	if handlerErr != nil {
		logger.Fatal(fmt.Sprintf("Handler creating error: %v", handlerErr))
	}
	filterWrapper := middlewares.UserFilterWrapper(services)
	handler.Use(filterWrapper)
	configureHandlers(ctx, handler)
	runComponentsWithGracefulShutdown(ctx, cancelFunc, bot, handler, srv, services.PostgresClient)
}
