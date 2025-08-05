package app

import (
	"fmt"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	hnd "github.com/unspokenteam/golang-tg-dbot/app/handlers"
	"logger"
)

func handlePanic() {
	panicErr := recover()
	if panicErr != nil {
		logger.LogError(fmt.Sprintf("panic: %s", panicErr), "PanicRestart", nil)
		go RestartAfterPanic()
	}
}

func panicHanlderWrapper(ctxWrapper *th.Context, updateWrapper telego.Update, wrappedFunc func(*th.Context, telego.Update)) {
	defer handlePanic()
	wrappedFunc(ctxWrapper, updateWrapper)
}

func registerHandler(handler *th.BotHandler, command string, handleFunc func(*th.Context, telego.Update)) {
	handler.Handle(
		func(thCtx *th.Context, update telego.Update) error {
			go func() { panicHanlderWrapper(thCtx, update, handleFunc) }()
			return nil
		},
		th.CommandEqual(command),
	)
}

func InjectTelegoHandlers(handler *th.BotHandler) {
	registerHandler(handler, "start", hnd.Start)
}
