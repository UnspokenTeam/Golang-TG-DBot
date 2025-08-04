package middlewares

import (
	"configs"
	"context"
	"errors"
	"fmt"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/redis/go-redis/v9"
	"logger"
	"strconv"
	"time"
)

const (
	QUIET = 0
	LOUD  = 1
	MUTED = 2
)

func muteSpammer(ctx context.Context, bot *telego.Bot, message *telego.Message) {
	mutePtr := false
	perms := &telego.ChatPermissions{
		CanSendMessages:       &mutePtr,
		CanSendOtherMessages:  &mutePtr,
		CanAddWebPagePreviews: &mutePtr,
	}
	until := message.Date + configs.MiddlewareConfig.SpamCooldown

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

func rateLimit(ctx context.Context, bot *telego.Bot, message *telego.Message) bool {
	isRequestHandled := true
	userStateStr, err := configs.Rdb.Get(ctx, strconv.FormatInt(message.From.ID, 10)).Result()
	if errors.Is(err, redis.Nil) {
		if setErr := configs.Rdb.Set(
			ctx,
			strconv.FormatInt(message.From.ID, 10),
			QUIET,
			time.Duration(configs.MiddlewareConfig.MuteCooldown)*time.Second,
		).Err(); setErr != nil {
			logger.LogError(fmt.Sprintf("Cannot set rate limit: %s", setErr), "cantSetRateLimit", message)
		} else {
			isRequestHandled = false
		}
	} else if err != nil {
		logger.LogError(fmt.Sprintf("Cannot get rate limit: %s", err), "cantGetRateLimit", message)
	} else {
		userState, _ := strconv.Atoi(userStateStr)
		switch userState {
		case QUIET:
			if incrErr := configs.Rdb.Set(
				ctx,
				strconv.FormatInt(message.From.ID, 10),
				LOUD,
				time.Duration(configs.MiddlewareConfig.MuteCooldown)*time.Second,
			).Err(); incrErr != nil {
				logger.LogError(fmt.Sprintf("Cannot set rate limit: %s", incrErr), "cantSetRateLimit", message)
			} else {
				isRequestHandled = false
			}
		case LOUD:
			if muteErr := configs.Rdb.Set(
				ctx,
				strconv.FormatInt(message.From.ID, 10),
				MUTED,
				time.Duration(configs.MiddlewareConfig.SpamCooldown)*time.Second,
			).Err(); muteErr != nil {
				logger.LogError(fmt.Sprintf("Cannot set rate limit: %s", muteErr), "cantSetRateLimit", message)
			}
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
	for len(MessageQueue) >= cap(MessageQueue) {
		time.Sleep(time.Second)
	}
	return ctx.Next(upd)
}
