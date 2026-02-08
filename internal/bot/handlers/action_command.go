package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
	"github.com/unspokenteam/golang-tg-dbot/internal/db/querier"
	hndUtils "github.com/unspokenteam/golang-tg-dbot/pkg/utils"
)

func Perform(ctx context.Context, upd telego.Update, services *service_wrapper.Services) {
	var (
		action   string
		yourself = upd.Message.ReplyToMessage == nil
	)

	if idx := strings.Index(upd.Message.Text, " "); idx != -1 {
		action = upd.Message.Text[idx+1:]
	} else {
		workers.EnqueueMessage(ctx,
			fmt.Sprintf("%s, напиши вместе с командой через пробел действие, которое хочешь совершить!",
				hndUtils.MentionUser(upd.Message.From.FirstName, upd.Message.From.ID)),
			upd.Message)
	}

	if err := services.PostgresClient.Queries.InsertNewAction(ctx, querier.InsertNewActionParams{
		IsYourself: yourself,
		ChatTgID:   upd.Message.Chat.ID,
		UserTgID:   upd.Message.From.ID,
		Action:     action,
	}); err != nil {
		return
	}

	if err := services.PostgresClient.Queries.UpdateLastMessageAt(ctx, querier.UpdateLastMessageAtParams{
		ChatTgID: upd.Message.Chat.ID,
		UserTgID: upd.Message.From.ID,
	}); err != nil {
		return
	}

	actionTo := ""
	if upd.Message.ReplyToMessage != nil {
		if hndUtils.IsValidUser(upd.Message.ReplyToMessage) {
			actionTo = hndUtils.MentionUser(upd.Message.ReplyToMessage.From.FirstName, upd.Message.ReplyToMessage.From.ID)
		} else {
			actionTo = hndUtils.GetStrangerName(upd.Message.ReplyToMessage)
		}
	}

	workers.EnqueueMessage(ctx,
		fmt.Sprintf("%s %s %s",
			hndUtils.MentionUser(upd.Message.From.FirstName, upd.Message.From.ID),
			action,
			actionTo,
		), upd.Message)

}
