package app

import (
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/channels"
	hnd "github.com/unspokenteam/golang-tg-dbot/internal/bot/handlers"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/roles"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/pkg/logger"
)

func handlePanic() {
	panicErr := recover()
	if panicErr != nil {
		logger.LogError(fmt.Sprintf("panic: %s", panicErr), "PanicRestart", nil)
		channels.ShutdownChannel <- struct{}{}
	}
}

func registerHandler(handler *th.BotHandler, command []string, handleFunc func(*th.Context, telego.Update, *service_wrapper.Services), roles []roles.Role) {
	for _, commandBind := range command {
		handler.Handle(
			func(thCtx *th.Context, update telego.Update) error {
				defer handlePanic()
				hnd.PreprocessUser(thCtx, update, services)
				hnd.CheckRoleAccess(thCtx, roles, services)
				handleFunc(thCtx, update, services)
				return nil
			},
			th.CommandEqual(commandBind),
		)
	}
}

func configureHandlers(handler *th.BotHandler) {
	registerHandler(
		handler,
		services.CommandsViper.GetStringSlice("start_commands"),
		hnd.Start,
		[]roles.Role{roles.USER, roles.ADMIN, roles.OWNER},
	)

	registerHandler(
		handler,
		services.CommandsViper.GetStringSlice("restart_commands"),
		hnd.Restart,
		[]roles.Role{roles.OWNER},
	)

	registerHandler(
		handler,
		services.CommandsViper.GetStringSlice("help_commands"),
		hnd.Help,
		[]roles.Role{roles.USER, roles.ADMIN, roles.OWNER},
	)
}
