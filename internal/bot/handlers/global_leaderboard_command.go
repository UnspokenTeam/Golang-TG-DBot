package handlers

import (
	"context"
	"fmt"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
	hndUtils "github.com/unspokenteam/golang-tg-dbot/pkg/utils"
)

func GlobalLeaderboard(ctx context.Context, upd telego.Update, services *service_wrapper.Services) {
	text := services.ConfigCache.GetString("top_placement_text_pattern") + "\n"
	header := "Топ 10 пользователей бота:\n"

	top, err := services.PostgresClient.Queries.GetGlobalLeaderBoards(ctx)
	if err != nil {
		return
	}
	for placement, row := range top {
		userMention := hndUtils.MentionUser(row.UserName, row.TgID)
		if row.UserTag != nil && *row.UserTag != "" {
			userMention = fmt.Sprintf("%s (@%s)", userMention, *row.UserTag)
		}
		header += fmt.Sprintf(text,
			placement+1,
			userMention,
			row.DLength,
			row.MActionCount,
			row.FActionCount,
			row.FActionFromStrangerCount,
			row.SActionFromStrangerCount,
			row.SActionCount,
			row.Loses)
	}

	workers.EnqueueMessage(ctx, header, upd.Message)
}
