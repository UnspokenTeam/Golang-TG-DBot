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
	"github.com/unspokenteam/golang-tg-dbot/pkg/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var reqCounter metric.Int64Counter

func registerHandler(
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
					attribute.Int64("user_id", update.Message.From.ID),
					attribute.Int64("chat_id", update.Message.Chat.ID),
					attribute.String("msg", utils.MarshalJsonIgnoreError(ctx, update.Message)),
				)
				defer handlerSpan.End()
				handleFunc(ctx, update, services)
				go reqCounter.Add(ctx, 1, metric.WithAttributes(
					attribute.Int64("user_id", update.Message.From.ID),
					attribute.Int64("chat_id", update.Message.Chat.ID),
				))
				go commandCounter.Add(ctx, 1, metric.WithAttributes(
					attribute.Int64("user_id", update.Message.From.ID),
					attribute.Int64("chat_id", update.Message.Chat.ID),
				))

				return nil
			},
			th.CommandEqual(commandBind),
		)
	}
	slog.InfoContext(rootCtx, fmt.Sprintf("Registered %d commands for %s", len(command), funcName))
}

func configureHandlers(handler *th.BotHandler) {
	var err error
	reqCounter, err = services.Meter.Int64Counter(
		"bot.requests.total",
		metric.WithDescription("Total number of bot commands processed"),
		metric.WithUnit("1"),
	)
	if err != nil {
		logger.Fatal(fmt.Sprintf("create metric err: %s", err))
		return
	}

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

	registerHandler(
		handler,
		services.CommandsViper.GetStringSlice("user_stats_commands"),
		hnd.PublicStats,
		[]roles.Role{roles.USER, roles.ADMIN, roles.OWNER},
	)

	registerHandler(
		handler,
		[]string{"promote"},
		hnd.Promote,
		[]roles.Role{roles.OWNER},
	)

	registerHandler(
		handler,
		[]string{"demote"},
		hnd.Demote,
		[]roles.Role{roles.OWNER},
	)

	registerHandler(
		handler,
		[]string{"admin_stats"},
		hnd.PrivateStats,
		[]roles.Role{roles.OWNER, roles.ADMIN},
	)

	registerHandler(
		handler,
		services.CommandsViper.GetStringSlice("my_stats_commands"),
		hnd.UserStats,
		[]roles.Role{roles.OWNER, roles.ADMIN, roles.USER},
	)

	registerHandler(
		handler,
		services.CommandsViper.GetStringSlice("up_commands"),
		hnd.Up,
		[]roles.Role{roles.OWNER, roles.ADMIN, roles.USER},
	)

	registerHandler(
		handler,
		[]string{"analyze"},
		hnd.Analyze,
		[]roles.Role{roles.OWNER, roles.ADMIN},
	)

	registerHandler(
		handler,
		services.CommandsViper.GetStringSlice("s_commands"),
		hnd.SAction,
		[]roles.Role{roles.OWNER, roles.ADMIN, roles.USER},
	)

	registerHandler(
		handler,
		services.CommandsViper.GetStringSlice("random_action_commands"),
		hnd.Random,
		[]roles.Role{roles.OWNER, roles.ADMIN, roles.USER},
	)

	registerHandler(
		handler,
		services.CommandsViper.GetStringSlice("m_commands"),
		hnd.MAction,
		[]roles.Role{roles.OWNER, roles.ADMIN, roles.USER},
	)

	registerHandler(
		handler,
		services.CommandsViper.GetStringSlice("f_commands"),
		hnd.FAction,
		[]roles.Role{roles.OWNER, roles.ADMIN, roles.USER},
	)

	registerHandler(
		handler,
		[]string{"echo"},
		hnd.Echo,
		[]roles.Role{roles.OWNER},
	)

	registerHandler(
		handler,
		services.CommandsViper.GetStringSlice("action_commands"),
		hnd.Perform,
		[]roles.Role{roles.OWNER, roles.ADMIN, roles.USER},
	)

	registerHandler(
		handler,
		services.CommandsViper.GetStringSlice("global_stats_commands"),
		hnd.GlobalLeaderboard,
		[]roles.Role{roles.OWNER, roles.ADMIN, roles.USER},
	)

	registerHandler(
		handler,
		services.CommandsViper.GetStringSlice("chat_stats_commands"),
		hnd.ChatLeaderboard,
		[]roles.Role{roles.OWNER, roles.ADMIN, roles.USER},
	)

	registerHandler(
		handler,
		services.CommandsViper.GetStringSlice("play_commands"),
		hnd.Play,
		[]roles.Role{roles.OWNER, roles.ADMIN, roles.USER},
	)

	registerHandler(
		handler,
		[]string{"send"},
		hnd.Broadcast,
		[]roles.Role{roles.OWNER},
	)

	registerHandler(
		handler,
		[]string{"talk"},
		hnd.Talk,
		[]roles.Role{roles.OWNER, roles.ADMIN, roles.USER},
	)
}
