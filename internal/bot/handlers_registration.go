package app

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"runtime"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	hnd "github.com/unspokenteam/golang-tg-dbot/internal/bot/handlers"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/roles"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/logger"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func registerHandler(
	appCtx context.Context,
	handler *th.BotHandler,
	command []string,
	handleFunc func(context.Context, telego.Update, *service_wrapper.Services),
	roles []roles.Role,
) {
	funcName := runtime.FuncForPC(reflect.ValueOf(handleFunc).Pointer()).Name()
	parts := strings.Split(funcName, ".")
	funcDisplayName := parts[len(parts)-1]

	commandCounter, err := services.Meter.Int64Counter(
		fmt.Sprintf("bot.%s.total", funcDisplayName),
		metric.WithDescription(fmt.Sprintf("Total number of bot %s command processed", funcDisplayName)),
		metric.WithUnit("1"),
	)
	if err != nil {
		logger.Fatal(fmt.Sprintf("create metric err: %s", err))
		return
	}

	for _, commandBind := range command {
		handler.Handle(
			func(thCtx *th.Context, update telego.Update) error {
				defer handlePanic()

				ctx, preprocessSpan := services.Tracer.Start(thCtx.Context(), "preprocess_user")
				defer preprocessSpan.End()
				user := hnd.PreprocessUser(ctx, update, services)
				if user == nil {
					return nil
				}

				ctx, checkRoleSpan := services.Tracer.Start(ctx, "check_user_role")
				defer checkRoleSpan.End()
				if !hnd.CheckRoleAccess(ctx, user, roles) {
					return nil
				}

				ctx, handlerSpan := services.Tracer.Start(ctx, "handler_span")
				handlerSpan.SetAttributes(
					attribute.String("command", funcDisplayName),
				)
				defer handlerSpan.End()
				handleFunc(ctx, update, services)
				commandCounter.Add(ctx, 1, metric.WithAttributes(
					attribute.Int64("user_id", update.Message.From.ID),
					attribute.Int64("chat_id", update.Message.Chat.ID),
				))

				return nil
			},
			th.CommandEqual(commandBind),
		)
	}
	slog.InfoContext(appCtx, fmt.Sprintf("Registered %d commands for %s", len(command), funcName))
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

	registerHandler(
		ctx,
		handler,
		services.CommandsViper.GetStringSlice("user_stats_commands"),
		hnd.PublicStats,
		[]roles.Role{roles.USER, roles.ADMIN, roles.OWNER},
	)

	registerHandler(
		ctx,
		handler,
		[]string{"promote"},
		hnd.Promote,
		[]roles.Role{roles.OWNER},
	)

	registerHandler(
		ctx,
		handler,
		[]string{"demote"},
		hnd.Demote,
		[]roles.Role{roles.OWNER},
	)

	registerHandler(
		ctx,
		handler,
		[]string{"admin_stats"},
		hnd.PrivateStats,
		[]roles.Role{roles.OWNER, roles.ADMIN},
	)

	registerHandler(
		ctx,
		handler,
		services.CommandsViper.GetStringSlice("my_stats_commands"),
		hnd.UserStats,
		[]roles.Role{roles.OWNER, roles.ADMIN, roles.USER},
	)

	registerHandler(
		ctx,
		handler,
		services.CommandsViper.GetStringSlice("up_commands"),
		hnd.Up,
		[]roles.Role{roles.OWNER, roles.ADMIN, roles.USER},
	)
}
