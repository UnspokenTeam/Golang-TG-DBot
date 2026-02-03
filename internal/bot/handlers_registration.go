package app

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"runtime"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	hnd "github.com/unspokenteam/golang-tg-dbot/internal/bot/handlers"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/roles"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"go.opentelemetry.io/otel/trace"
)

func registerHandler(
	appCtx context.Context,
	handler *th.BotHandler,
	command []string,
	handleFunc func(context.Context, trace.Span, telego.Update, *service_wrapper.Services),
	roles []roles.Role,
) {
	for _, commandBind := range command {
		handler.Handle(
			func(thCtx *th.Context, update telego.Update) error {
				defer handlePanic()

				ctx, preprocessSpan := services.Tracer.Start(thCtx.Context(), "preprocess_user")
				defer preprocessSpan.End()
				hnd.PreprocessUser(ctx, preprocessSpan, update, services)

				ctx, checkRoleSpan := services.Tracer.Start(ctx, "check_user_role")
				defer checkRoleSpan.End()
				hnd.CheckRoleAccess(ctx, checkRoleSpan, roles, services)

				ctx, handlerSpan := services.Tracer.Start(ctx, "handler_span")
				defer handlerSpan.End()
				handleFunc(ctx, handlerSpan, update, services)

				return nil
			},
			th.CommandEqual(commandBind),
		)
	}
	slog.InfoContext(appCtx, fmt.Sprintf("Registered %d commands for %s", len(command), runtime.FuncForPC(reflect.ValueOf(handleFunc).Pointer()).Name()))
}

func configureHandlers(ctx context.Context, handler *th.BotHandler) {
	registerHandler(
		ctx,
		handler,
		services.CommandsViper.GetStringSlice("start_commands"),
		hnd.Start,
		[]roles.Role{roles.USER, roles.ADMIN, roles.OWNER},
	)

	registerHandler(
		ctx,
		handler,
		services.CommandsViper.GetStringSlice("restart_commands"),
		hnd.Restart,
		[]roles.Role{roles.OWNER},
	)

	registerHandler(
		ctx,
		handler,
		services.CommandsViper.GetStringSlice("help_commands"),
		hnd.Help,
		[]roles.Role{roles.USER, roles.ADMIN, roles.OWNER},
	)
}
