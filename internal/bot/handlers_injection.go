package app

import (
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/channels"
	hnd "github.com/unspokenteam/golang-tg-dbot/internal/bot/handlers"
	"github.com/unspokenteam/golang-tg-dbot/pkg/logger"
)

func handlePanic() {
	panicErr := recover()
	if panicErr != nil {
		logger.LogError(fmt.Sprintf("panic: %s", panicErr), "PanicRestart", nil)
		channels.ShutdownChannel <- struct{}{}
	}
}

func handlerWrapper(ctxWrapper *th.Context, updateWrapper telego.Update, wrappedFunc func(*th.Context, telego.Update)) {
	defer handlePanic()
	//hnd.HandleUser(ctxWrapper, updateWrapper)
	wrappedFunc(ctxWrapper, updateWrapper)
}

func registerHandler(handler *th.BotHandler, command []string, handleFunc func(*th.Context, telego.Update)) {
	for _, commandBind := range command {
		handler.Handle(
			func(thCtx *th.Context, update telego.Update) error {
				handlerWrapper(thCtx, update, handleFunc)
				return nil
			},
			th.CommandEqual(commandBind),
		)
	}
}

func InjectTelegoHandlers(handler *th.BotHandler) {
	registerHandler(handler, []string{"start"}, hnd.Start)
}
