package app

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/channels"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
	"github.com/unspokenteam/golang-tg-dbot/internal/logger"
	"github.com/unspokenteam/golang-tg-dbot/pkg/utils"
	"github.com/uptrace/uptrace-go/uptrace"
	"github.com/valyala/fasthttp"
	"go.opentelemetry.io/otel/attribute"
)

var componentsWG sync.WaitGroup

func panicListener(cancel context.CancelFunc) {
	select {
	case <-channels.ShutdownChannel:
		defer cancel()

		ctx, handlerSpan := services.Tracer.Start(rootCtx, "shutdown_span")
		handlerSpan.SetAttributes(attribute.Bool("shutdown", true))
		slog.DebugContext(ctx, "Shutting down gracefully")
		handlerSpan.End()

		timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 5*time.Second)
		defer timeoutCancel()
		if err := uptrace.ForceFlush(timeoutCtx); err != nil {
			slog.ErrorContext(timeoutCtx, fmt.Sprintf("flush error: %v", err))
		}
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
	services *service_wrapper.Services,
	bot *telego.Bot,
	handler *th.BotHandler,
	srv *fasthttp.Server,
) {
	go panicListener(cancel)

	addComponent(func() { workers.GracefulShutdownLoggerBridge(rootCtx) })
	addComponent(func() {
		defer services.PostgresClient.Close(rootCtx)
		<-rootCtx.Done()
	})
	if utils.IsEnvProduction() {
		port := services.AppViper.GetInt("WEBHOOK_PORT")
		addComponent(func() { workers.StartServer(rootCtx, srv, port) })
	}
	addComponent(func() { workers.InitQueues(rootCtx, bot, services.TgApiRateLimiter, services.Meter) })
	addComponent(func() { workers.RunCommandConsumer(rootCtx, handler) })
	addComponent(func() { workers.InitBroadcastWorker(rootCtx, services) })

	slog.InfoContext(rootCtx, "Started app components")
	componentsWG.Wait()

	channels.CloseChannels()
	slog.InfoContext(rootCtx, "Stopped app components")
}
