package middlewares

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/unspokenteam/golang-tg-dbot/internal/bot/channels"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/configs"
	"github.com/unspokenteam/golang-tg-dbot/internal/logger"
	"github.com/unspokenteam/golang-tg-dbot/pkg/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/redis/go-redis/v9"
)

const (
	UNDEFINED = iota
	QUIET
	LOUD
	MUTED
)

var (
	services    *service_wrapper.Services
	muteCounter metric.Int64Counter
	cfg         configs.RateLimiterConfig
)

func makeKey(chatID, userID int64) string {
	buf := make([]byte, 16)
	binary.BigEndian.PutUint64(buf[0:8], uint64(chatID))
	binary.BigEndian.PutUint64(buf[8:16], uint64(userID))
	return string(buf)
}

func tryIncrUserState(ctx context.Context, redisId string, currentState int, cooldown time.Duration, message *telego.Message) bool {
	if setErr := services.RateLimitCache.Set(
		ctx,
		redisId,
		currentState+1,
		cooldown,
	).Err(); setErr != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Cannot set rate limit: %s", setErr), "payload", message)
		return false
	}
	return true
}

func isUserDeniedByRateLimit(ctx *th.Context, message *telego.Message) bool {
	userDenied := true
	muteCooldown := time.Duration(cfg.MuteCooldown) * time.Second

	redisId := makeKey(message.Chat.ID, message.From.ID)
	userState, err := services.RateLimitCache.Get(ctx.Context(), redisId).Int()

	if errors.Is(err, redis.Nil) {
		userDenied = !tryIncrUserState(ctx.Context(), redisId, UNDEFINED, muteCooldown, message)
	} else if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Cannot get rate limit: %s", err), "payload", message)
	} else {
		switch userState {
		case QUIET:
			userDenied = !tryIncrUserState(ctx.Context(), redisId, userState, muteCooldown, message)
		case LOUD:
			tryIncrUserState(ctx.Context(), redisId, userState, time.Duration(cfg.SpamCooldown)*time.Second, message)
			slog.DebugContext(ctx, fmt.Sprintf("Muting user %d...", message.From.ID), "payload", message)

			go muteCounter.Add(ctx, 1, metric.WithAttributes(
				attribute.Int64("user_id", message.From.ID),
				attribute.Int64("chat_id", message.Chat.ID),
				attribute.String("msg", fmt.Sprintf("%+v", message)),
			))

			go utils.TryMuteSpammer(ctx, message, cfg.SpamCooldown, services.TgApiRateLimiter)
		default:
		}
	}
	return userDenied
}

func UserFilterWrapper(wrapper *service_wrapper.Services) func(*th.Context, telego.Update) error {
	services = wrapper
	cfg = configs.LoadConfig(services.AppViper, configs.RateLimiterConfig{})

	var err error
	muteCounter, err = services.Meter.Int64Counter(
		"bot.mute.total",
		metric.WithDescription("Total number of mutes in rate limiter"),
		metric.WithUnit("1"),
	)
	if err != nil {
		logger.Fatal(fmt.Sprintf("create metric err: %s", err))
		return nil
	}

	return func(ctx *th.Context, upd telego.Update) error {
		msg := upd.Message
		if !utils.IsMessageChatCommand(msg) || isUserDeniedByRateLimit(ctx, msg) {
			return nil
		}

		timeoutCtx, cancel := context.WithTimeout(ctx.Context(), 30*time.Second)
		defer cancel()

		for {
			select {
			case <-timeoutCtx.Done():
				if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
					slog.WarnContext(
						timeoutCtx, "Update dropped: 30s timeout waiting for channel",
						"update", upd.UpdateID,
						"payload", upd,
					)
					return nil
				}
				return timeoutCtx.Err()
			default:
				if len(channels.SenderChannel) < cap(channels.SenderChannel) {
					return ctx.Next(upd)
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}
