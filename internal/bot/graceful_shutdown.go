package app

import (
	"context"
	"sync"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/channels"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
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

func runComponentsWithGracefulShutdown(
	ctx context.Context,
	cancel context.CancelFunc,
	bot *telego.Bot,
	handler *th.BotHandler,
	srv *fasthttp.Server,
	port int16,
) {
	go panicListener(cancel)
	if utils.IsEnvProduction() {
		componentsWG.Go(func() { workers.StartServer(ctx, srv, port) })
	}
	componentsWG.Go(func() { workers.OpenQueue(ctx, bot) })
	componentsWG.Go(func() { workers.RunCommandConsumer(ctx, handler) })
	componentsWG.Wait()
	channels.CloseChannels()
}
