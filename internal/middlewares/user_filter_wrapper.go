package middlewares

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/unspokenteam/golang-tg-dbot/internal/bot/channels"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	configs "github.com/unspokenteam/golang-tg-dbot/internal/config"
	"github.com/unspokenteam/golang-tg-dbot/pkg/logger"
	"github.com/unspokenteam/golang-tg-dbot/pkg/utils"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/redis/go-redis/v9"
)

const (
	UNDEFINED = -1
	QUIET     = 0
	LOUD      = 1
	MUTED     = 2
)

var (
	services *service_wrapper.Services
	cfg      configs.RateLimiterConfig
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
		logger.LogError(fmt.Sprintf("Cannot set rate limit: %s", setErr), "cantSetRateLimit", message)
		return false
	}
	return true
}

func isUserDeniedByRateLimit(ctx *th.Context, message *telego.Message) bool {
	userDenied := true
	muteCooldown := time.Duration(cfg.MuteCooldown) * time.Second

	redisId := makeKey(message.Chat.ID, message.From.ID)
	userStateStr, err := services.RateLimitCache.Get(ctx.Context(), redisId).Result()

	if errors.Is(err, redis.Nil) {
		userDenied = !tryIncrUserState(ctx.Context(), redisId, UNDEFINED, muteCooldown, message)
	} else if err != nil {
		logger.LogError(fmt.Sprintf("Cannot get rate limit: %s", err), "cantGetRateLimit", message)
	} else {
		userState, _ := strconv.Atoi(userStateStr)
		switch userState {
		case QUIET:
			userDenied = !tryIncrUserState(ctx.Context(), redisId, userState, muteCooldown, message)
		case LOUD:
			tryIncrUserState(ctx.Context(), redisId, userState, time.Duration(cfg.SpamCooldown)*time.Second, message)
			go utils.TryMuteSpammer(ctx, message, cfg.SpamCooldown)
		}
	}
	return userDenied
}

func UserFilterWrapper(wrapper *service_wrapper.Services) func(*th.Context, telego.Update) error {
	services = wrapper
	cfg = configs.LoadConfig(services.AppViper, configs.RateLimiterConfig{})

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
					log.Printf("Update %d dropped: 30s timeout waiting for channel", upd.UpdateID)
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
