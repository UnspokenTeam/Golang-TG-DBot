package middlewares

import (
	"configs"
	"context"
	"errors"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/redis/go-redis/v9"
	"log"
	"runtime/debug"
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

	err := bot.RestrictChatMember(ctx, &telego.RestrictChatMemberParams{
		ChatID:                        message.Chat.ChatID(),
		UserID:                        message.From.ID,
		Permissions:                   *perms,
		UntilDate:                     until,
		UseIndependentChatPermissions: true,
	})
	if err != nil {
		log.Println(err)
	}
}

func tryMuteSpammer(bot *telego.Bot, message *telego.Message) {
	ctx := context.Background()
	me, err := bot.GetMe(ctx)
	if err != nil {
		log.Printf("ERROR: %s\nSTACK:\n%s", err.Error(), debug.Stack())
		return
	}
	botChatMember, err := bot.GetChatMember(ctx, &telego.GetChatMemberParams{
		ChatID: message.Chat.ChatID(), UserID: me.ID,
	})
	if err != nil {
		log.Printf("ERROR: %s\nSTACK:\n%s", err.Error(), debug.Stack())
		return
	}

	allowed := false
	switch m := botChatMember.(type) {
	case *telego.ChatMemberOwner:
		allowed = true
	case *telego.ChatMemberAdministrator:
		allowed = allowed || m.CanRestrictMembers
	}

	memberToMute, err := bot.GetChatMember(ctx, &telego.GetChatMemberParams{
		ChatID: message.Chat.ChatID(), UserID: message.From.ID,
	})
	if err != nil {
		log.Println(err)
	}
	switch memberToMute.(type) {
	case *telego.ChatMemberOwner, *telego.ChatMemberAdministrator:
		allowed = false
	}
	if message.Chat.Type != "supergroup" {
		allowed = false
	}
	if allowed {
		muteSpammer(ctx, bot, message)
	}
}

func rateLimit(ctx context.Context, bot *telego.Bot, message *telego.Message) bool {
	isRequestHandled := true
	userStateStr, err := configs.Rdb.Get(ctx, strconv.FormatInt(message.From.ID, 10)).Result()
	if errors.Is(err, redis.Nil) {
		err := configs.Rdb.Set(
			ctx,
			strconv.FormatInt(message.From.ID, 10),
			QUIET,
			time.Duration(configs.MiddlewareConfig.MuteCooldown)*time.Second,
		).Err()
		if err != nil {
			log.Println(err)
		} else {
			isRequestHandled = false
		}
	} else if err != nil {
		log.Println(err)
	} else {
		userState, _ := strconv.Atoi(userStateStr)
		switch userState {
		case QUIET:
			err := configs.Rdb.Set(
				ctx,
				strconv.FormatInt(message.From.ID, 10),
				LOUD,
				time.Duration(configs.MiddlewareConfig.MuteCooldown)*time.Second,
			).Err()
			if err != nil {
				log.Println(err)
			} else {
				isRequestHandled = false
			}
		case LOUD:
			err = configs.Rdb.Set(
				ctx,
				strconv.FormatInt(message.From.ID, 10),
				MUTED,
				time.Duration(configs.MiddlewareConfig.SpamCooldown)*time.Second,
			).Err()
			if err != nil {
				log.Println(err)
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
