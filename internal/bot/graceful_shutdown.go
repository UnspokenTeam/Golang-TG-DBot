package app

import (
	"context"
	"log/slog"
	"sync"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/channels"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
	"github.com/unspokenteam/golang-tg-dbot/internal/db"
	"github.com/unspokenteam/golang-tg-dbot/internal/logger"
	"github.com/unspokenteam/golang-tg-dbot/pkg/utils"
	"github.com/valyala/fasthttp"
)

var componentsWG sync.WaitGroup

func panicListener(cancel context.CancelFunc) {
	select {
	case <-channels.ShutdownChannel:
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
	ctx context.Context,
	cancel context.CancelFunc,
	bot *telego.Bot,
	handler *th.BotHandler,
	srv *fasthttp.Server,
	postgresClient *db.Client,
) {
	channels.InitChannels()

	go panicListener(cancel)

	addComponent(func() { workers.GracefulShutdownLoggerBridge(ctx) })
	addComponent(func() {
		defer postgresClient.Close(ctx)
		<-ctx.Done()
	})
	if utils.IsEnvProduction() {
		port := services.AppViper.GetInt("WEBHOOK_PORT")
		addComponent(func() { workers.StartServer(ctx, srv, port) })
	}
	addComponent(func() { workers.OpenQueue(ctx, bot, services.AppViper.GetInt("RPS_LIMIT")) })
	addComponent(func() { workers.RunCommandConsumer(ctx, handler) })

	slog.InfoContext(ctx, "Started app components")
	componentsWG.Wait()

	channels.CloseChannels()
	slog.InfoContext(ctx, "Stopped app components")
}
