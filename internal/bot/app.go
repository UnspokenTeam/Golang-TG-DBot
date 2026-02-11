package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/configs"
	"github.com/unspokenteam/golang-tg-dbot/internal/logger"
	"github.com/unspokenteam/golang-tg-dbot/internal/middlewares"
	"github.com/unspokenteam/golang-tg-dbot/pkg/utils"
	"github.com/uptrace/uptrace-go/uptrace"
	"github.com/valyala/fasthttp"
	"go.opentelemetry.io/otel/trace"
)

var (
	services *service_wrapper.Services
	rootCtx  context.Context
)

func initBotInstance(token string) *telego.Bot {
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
		telego.WithHealthCheck(rootCtx),
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

	var rootSpan trace.Span
	rootCtx, rootSpan = services.Tracer.Start(appCtx, "Main app span")
	defer rootSpan.End()

	switch utils.GetEnv() {
	case utils.DEVELOPMENT:
		var channelErr error
		bot = initBotInstance(services.AppViper.GetString("DEV_TOKEN"))
		if updatesCh, channelErr = bot.UpdatesViaLongPolling(rootCtx, nil); channelErr != nil {
			slog.ErrorContext(rootCtx, fmt.Sprintf("Channel error: %v", channelErr))
		}

	case utils.PRODUCTION:
		prodConfig := configs.LoadConfig(services.AppViper, configs.ProdBotConfig{})
		bot = initBotInstance(prodConfig.ProdToken)

		webhookPath := "/" + bot.Token()
		webhookURL := fmt.Sprintf("https://api.%s%s", prodConfig.CaddyDomain, webhookPath)

		var (
			info              *telego.WebhookInfo
			getWebhookInfoErr error
			replacer          = strings.NewReplacer(
				services.AppViper.GetString("PROD_TOKEN"), "PROD_TOKEN",
				services.AppViper.GetString("DEV_TOKEN"), "DEV_TOKEN",
			)
		)

		if info, getWebhookInfoErr = bot.GetWebhookInfo(rootCtx); getWebhookInfoErr != nil {

			slog.ErrorContext(rootCtx, replacer.Replace(fmt.Sprintf("Get webhook info error: %v\nInfo:%s",
				getWebhookInfoErr, utils.MarshalJsonIgnoreError(rootCtx, info))))
		}

		if info.URL != webhookURL {
			if setWebhookErr := bot.SetWebhook(rootCtx, &telego.SetWebhookParams{
				URL:         webhookURL,
				SecretToken: bot.SecretToken(),
			}); setWebhookErr != nil {
				slog.ErrorContext(rootCtx, fmt.Sprintf("Set webhook error: %v", setWebhookErr))
			}

			if info, getWebhookInfoErr = bot.GetWebhookInfo(rootCtx); getWebhookInfoErr != nil {
				slog.ErrorContext(rootCtx, replacer.Replace(fmt.Sprintf("Get final webhook error: %v\nInfo:%s",
					getWebhookInfoErr, utils.MarshalJsonIgnoreError(rootCtx, info))))
			}
		}

		slog.InfoContext(rootCtx, replacer.Replace(fmt.Sprintf("Webhook Info: %s\n",
			utils.MarshalJsonIgnoreError(rootCtx, info))))

		var channelErr error
		if updatesCh, channelErr = bot.UpdatesViaWebhook(
			rootCtx,
			telego.WebhookFastHTTP(srv, webhookPath, bot.SecretToken()),
			telego.WithWebhookBuffer(prodConfig.BufferSize),
		); channelErr != nil {
			slog.ErrorContext(rootCtx, fmt.Sprintf("Channel error: %v", channelErr))
		}

	default:
		logger.Fatal(fmt.Sprintf("Unknown env GO_ENV=%s", utils.GetEnv()))
		services.PostgresClient.Close(rootCtx)
		if err := uptrace.Shutdown(rootCtx); err != nil {
			slog.ErrorContext(rootCtx, fmt.Sprintf("Error in uptrace shutdown: %v", err))
		}
		return
	}

	handler, handlerErr := th.NewBotHandler(bot, updatesCh)
	if handlerErr != nil {
		logger.Fatal(fmt.Sprintf("Handler creating error: %v", handlerErr))
	}
	filterWrapper := middlewares.UserFilterWrapper(services)
	handler.Use(filterWrapper)
	configureHandlers(handler)
	runComponentsWithGracefulShutdown(cancelFunc, services, bot, handler, srv)
}
