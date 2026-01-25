package middlewares

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/unspokenteam/golang-tg-dbot/internal/bot/channels"
	"github.com/unspokenteam/golang-tg-dbot/pkg/logger"
	"github.com/unspokenteam/golang-tg-dbot/pkg/utils"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/redis/go-redis/v9"
	config "github.com/unspokenteam/golang-tg-dbot/internal/config"
)

const (
	UNDEFINED = -1
	QUIET     = 0
	LOUD      = 1
	MUTED     = 2
)

func makeKey(chatID, userID int64) string {
	buf := make([]byte, 16)
	binary.BigEndian.PutUint64(buf[0:8], uint64(chatID))
	binary.BigEndian.PutUint64(buf[8:16], uint64(userID))
	return string(buf)
}

func tryIncrUserState(ctx context.Context, redisId string, currentState int, cooldown time.Duration, message *telego.Message) bool {
	if setErr := config.Rdb.Set(
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
	muteCooldown := time.Duration(config.MiddlewareConfig.MuteCooldown) * time.Second

	redisId := makeKey(message.Chat.ID, message.From.ID)
	userStateStr, err := config.Rdb.Get(ctx.Context(), redisId).Result()

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
			tryIncrUserState(ctx.Context(), redisId, userState, time.Duration(config.MiddlewareConfig.SpamCooldown)*time.Second, message)
			go utils.TryMuteSpammer(ctx, message, config.MiddlewareConfig.SpamCooldown)
		}
	}
	return userDenied
}

func UserFilterWrapper(ctx *th.Context, upd telego.Update) error {
	msg := upd.Message
	if !utils.IsMessageChatCommand(msg) || isUserDeniedByRateLimit(ctx, msg) {
		return nil
	}
	for {
		select {
		case <-ctx.Context().Done():
			return ctx.Context().Err()
		default:
			if len(channels.SenderChannel) < cap(channels.SenderChannel) {
				return ctx.Next(upd)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}
