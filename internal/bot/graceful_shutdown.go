package app

import (
	"context"
	"log/slog"
	"sync"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/channels"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
	"github.com/unspokenteam/golang-tg-dbot/internal/db/querier"
	"github.com/unspokenteam/golang-tg-dbot/internal/logger"
	"github.com/unspokenteam/golang-tg-dbot/pkg/utils"
	"github.com/valyala/fasthttp"
	"go.opentelemetry.io/otel/attribute"
)

var componentsWG sync.WaitGroup

func panicListener(cancel context.CancelFunc) {
	select {
	case <-channels.ShutdownChannel:
		ctx, handlerSpan := services.Tracer.Start(rootCtx, "shutdown_span")
		handlerSpan.SetAttributes(attribute.Bool("shutdown", true))
		slog.DebugContext(ctx, "Shutting down gracefully")
		handlerSpan.End()

		cancel()
	}
}

func handlePanic() {
	panicErr := recover()
	if panicErr != nil {
		logger.Fatal("panic: %s", panicErr)
	}
}

func addComponent(function func()) {
	componentsWG.Go(func() {
		defer handlePanic()
		function()
	})
}

func runComponentsWithGracefulShutdown(
	cancel context.CancelFunc,
	bot *telego.Bot,
	handler *th.BotHandler,
	srv *fasthttp.Server,
	postgresClient *querier.DbClient,
) {
	go panicListener(cancel)

	addComponent(func() { workers.GracefulShutdownLoggerBridge(rootCtx) })
	addComponent(func() {
		defer postgresClient.Close(rootCtx)
		<-rootCtx.Done()
	})
	if utils.IsEnvProduction() {
		port := services.AppViper.GetInt("WEBHOOK_PORT")
		addComponent(func() { workers.StartServer(rootCtx, srv, port) })
	}
	addComponent(func() { workers.OpenQueue(rootCtx, bot, services.TgApiRateLimiter) })
	addComponent(func() { workers.RunCommandConsumer(rootCtx, handler) })

	slog.InfoContext(rootCtx, "Started app components")
	componentsWG.Wait()

	channels.CloseChannels()
	slog.InfoContext(rootCtx, "Stopped app components")
}
