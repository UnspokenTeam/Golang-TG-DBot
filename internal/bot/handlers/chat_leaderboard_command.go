package handlers

import (
	"context"
	"fmt"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
	hndUtils "github.com/unspokenteam/golang-tg-dbot/pkg/utils"
)

func ChatLeaderboard(ctx context.Context, upd telego.Update, services *service_wrapper.Services) {
	if !hndUtils.IsGroup(upd) {
		workers.EnqueueMessage(ctx, "Команда доступна только в группах.", upd.Message)
		return
	}
	text := services.ConfigCache.GetString("top_placement_text_pattern") + "\n"
	header := fmt.Sprintf("Топ чата %s:\n", upd.Message.Chat.Title)

	top, err := services.PostgresClient.Queries.GetChatLeaderBoards(ctx, upd.Message.Chat.ID)
	if err != nil {
		return
	}
	for placement, row := range top {
		userMention := hndUtils.MentionUser(row.UserName, row.UserTgID)
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
