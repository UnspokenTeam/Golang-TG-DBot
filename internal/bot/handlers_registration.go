package app

import (
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/channels"
	hnd "github.com/unspokenteam/golang-tg-dbot/internal/bot/handlers"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/roles"
	"github.com/unspokenteam/golang-tg-dbot/pkg/logger"
)

func handlePanic() {
	panicErr := recover()
	if panicErr != nil {
		logger.LogError(fmt.Sprintf("panic: %s", panicErr), "PanicRestart", nil)
		channels.ShutdownChannel <- struct{}{}
	}
}

func registerHandler(handler *th.BotHandler, command []string, handleFunc func(*th.Context, telego.Update), roles []roles.Role) {
	for _, commandBind := range command {
		handler.Handle(
			func(thCtx *th.Context, update telego.Update) error {
				defer handlePanic()
				hnd.PreprocessUser(thCtx, update)
				hnd.CheckRoleAccess(thCtx, roles)
				handleFunc(thCtx, update)
				return nil
			},
			th.CommandEqual(commandBind),
		)
	}
}

func configureHandlers(handler *th.BotHandler) {
	registerHandler(handler, []string{"start"}, hnd.Start, []roles.Role{roles.USER, roles.ADMIN, roles.OWNER})
	registerHandler(handler, []string{"restart"}, hnd.Restart, []roles.Role{roles.OWNER})
}
