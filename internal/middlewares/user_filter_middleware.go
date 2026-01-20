package middlewares

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/unspokenteam/golang-tg-dbot/pkg/logger"

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

func muteSpammer(ctx context.Context, bot *telego.Bot, message *telego.Message) {
	mutePtr := false
	perms := &telego.ChatPermissions{
		CanSendMessages:       &mutePtr,
		CanSendOtherMessages:  &mutePtr,
		CanAddWebPagePreviews: &mutePtr,
	}
	until := message.Date + config.MiddlewareConfig.SpamCooldown

	if err := bot.RestrictChatMember(ctx, &telego.RestrictChatMemberParams{
		ChatID:                        message.Chat.ChatID(),
		UserID:                        message.From.ID,
		Permissions:                   *perms,
		UntilDate:                     until,
		UseIndependentChatPermissions: true,
	}); err != nil {
		logger.LogError(fmt.Sprintf("restrict chat member error: %s", err), "muteUser", message)
	}
}

func tryMuteSpammer(bot *telego.Bot, message *telego.Message) {
	me, err := bot.GetMe(Ctx)
	if err != nil {
		logger.LogError(fmt.Sprintf("Cannot get bot instance: %s", err), "cantGetBotInstance", message)
		return
	}
	botChatMember, botErr := bot.GetChatMember(Ctx, &telego.GetChatMemberParams{
		ChatID: message.Chat.ChatID(), UserID: me.ID,
	})
	if botErr != nil {
		logger.LogError(fmt.Sprintf("Cannot get chat bot instance: %s", botErr), "cantGetBotInstance", message)
		return
	}

	allowed := false
	switch m := botChatMember.(type) {
	case *telego.ChatMemberOwner:
		allowed = true
	case *telego.ChatMemberAdministrator:
		allowed = m.CanRestrictMembers
	}

	memberToMute, memberGetErr := bot.GetChatMember(Ctx, &telego.GetChatMemberParams{
		ChatID: message.Chat.ChatID(), UserID: message.From.ID,
	})
	if memberGetErr != nil {
		logger.LogError(fmt.Sprintf("Cannot get chat member instance: %s", memberGetErr), "cantGetMemberInstance", message)
	}
	switch memberToMute.(type) {
	case *telego.ChatMemberOwner, *telego.ChatMemberAdministrator:
		allowed = false
	}
	if message.Chat.Type != "supergroup" {
		allowed = false
	}
	if allowed {
		muteSpammer(Ctx, bot, message)
	}
}

func makeKey(chatID, userID int64) string {
	buf := make([]byte, 16)
	binary.BigEndian.PutUint64(buf[0:8], uint64(chatID))
	binary.BigEndian.PutUint64(buf[8:16], uint64(userID))
	return string(buf)
}

func incrUserState(ctx context.Context, redisId string, currentState int, cooldown time.Duration, message *telego.Message) bool {
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

func rateLimit(ctx context.Context, bot *telego.Bot, message *telego.Message) bool {
	isRequestHandled := true
	muteCooldown := time.Duration(config.MiddlewareConfig.MuteCooldown) * time.Second
	redisId := makeKey(message.Chat.ID, message.From.ID)
	userStateStr, err := config.Rdb.Get(ctx, redisId).Result()
	if errors.Is(err, redis.Nil) {
		if errIsNil := incrUserState(ctx, redisId, UNDEFINED, muteCooldown, message); errIsNil {
			isRequestHandled = false
		}
	} else if err != nil {
		logger.LogError(fmt.Sprintf("Cannot get rate limit: %s", err), "cantGetRateLimit", message)
	} else {
		userState, _ := strconv.Atoi(userStateStr)
		switch userState {
		case QUIET:
			if errIsNil := incrUserState(ctx, redisId, userState, muteCooldown, message); errIsNil {
				isRequestHandled = false
			}
		case LOUD:
			incrUserState(ctx, redisId, userState, time.Duration(config.MiddlewareConfig.SpamCooldown)*time.Second, message)
			go tryMuteSpammer(bot, message)
		}
	}
	return isRequestHandled
}

func isRequestValid(message *telego.Message) bool {
	return !(message.IsAutomaticForward || (message.From != nil && message.From.IsBot))
}

func filterRequest(ctx context.Context, bot *telego.Bot, message *telego.Message) bool {
	return isRequestValid(message) && !rateLimit(ctx, bot, message)
}

func UserFilterMiddleware(ctx *th.Context, upd telego.Update) error {
	msg := upd.Message
	if msg == nil || msg.Chat.Type == "channel" || msg.Text == "" ||
		msg.Text[0] != '/' || !filterRequest(ctx.Context(), ctx.Bot(), msg) {
		return nil
	}
	for {
		select {
		case <-ctx.Context().Done():
			return ctx.Context().Err()
		default:
			if len(MessageQueue) < cap(MessageQueue) {
				return ctx.Next(upd)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}
