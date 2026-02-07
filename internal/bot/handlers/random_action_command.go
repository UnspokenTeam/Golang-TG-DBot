package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
	"github.com/unspokenteam/golang-tg-dbot/internal/db/querier"
	hndUtils "github.com/unspokenteam/golang-tg-dbot/pkg/utils"
)

func randomChoice(choices []string) string {
	return choices[rand.Intn(len(choices))]
}

func Random(ctx context.Context, upd telego.Update, services *service_wrapper.Services) {
	var (
		action   string
		yourself = upd.Message.ReplyToMessage == nil
	)
	if yourself {
		newest, actionErr := services.PostgresClient.Queries.GetYourselfRandomActionFromNewest(ctx)
		if actionErr != nil {
			return
		}

		action = newest.Action
		slog.DebugContext(ctx, "Action performed", "action_id", newest.ID)
	} else {
		newest, actionErr := services.PostgresClient.Queries.GetRandomActionForStrangerFromNewest(ctx)
		if actionErr != nil {
			return
		}

		action = newest.Action
		slog.DebugContext(ctx, "Action performed", "action_id", newest.ID)
	}

	err := services.PostgresClient.Queries.UpdateLastMessageAt(ctx, querier.UpdateLastMessageAtParams{
		ChatTgID: upd.Message.Chat.ID,
		UserTgID: upd.Message.From.ID,
	})
	if err != nil {
		return
	}

	text := services.ConfigCache.GetString("perform_random_text_pattern")

	actionTo := ""
	if upd.Message.ReplyToMessage != nil {
		actionTo = hndUtils.MentionUser(upd.Message.ReplyToMessage.From.FirstName, upd.Message.ReplyToMessage.From.ID)
	}

	workers.EnqueueMessage(ctx,
		fmt.Sprintf(text,
			hndUtils.MentionUser(upd.Message.From.FirstName, upd.Message.From.ID),
			action,
			actionTo,
		), upd.Message)

}
