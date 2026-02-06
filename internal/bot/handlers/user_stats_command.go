package handlers

import (
	"context"
	"fmt"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
	"github.com/unspokenteam/golang-tg-dbot/internal/db/querier"
	hndUtils "github.com/unspokenteam/golang-tg-dbot/pkg/utils"
)

func UserStats(ctx context.Context, upd telego.Update, services *service_wrapper.Services) {
	if !hndUtils.IsGroup(upd) {
		workers.EnqueueMessage(ctx, "Команда доступна только в группах.", upd.Message)
	}
	stats, err := services.PostgresClient.Queries.GetUserStatsByTgId(ctx, querier.GetUserStatsByTgIdParams{
		ChatTgID: upd.Message.Chat.ID,
		UserTgID: upd.Message.From.ID,
	})
	if err != nil {
		return
	}
	text := services.ConfigCache.GetString("my_stats_text_pattern")
	workers.EnqueueMessage(ctx, fmt.Sprintf(text, hndUtils.MentionUser(stats.UserName, upd.Message.From.ID), stats.Name, stats.DLength, stats.MActionCount, stats.FActionCount, stats.SActionCount, stats.Loses), upd.Message)
}
