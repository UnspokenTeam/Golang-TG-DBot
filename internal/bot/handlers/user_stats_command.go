package handlers

import (
	"context"
	"fmt"
	"math"

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

	percent := 100 - ((float64(stats.Loses) / float64(stats.GamesPlayed)) * 100)
	percentRounded := math.Ceil(percent)
	winrate := fmt.Sprintf("%.2f%%\n", percentRounded)

	text := services.ConfigCache.GetString("my_stats_text_pattern")
	workers.EnqueueMessage(ctx, fmt.Sprintf(text, hndUtils.MentionUser(stats.UserName, upd.Message.From.ID),
		stats.UserRole, stats.Name, stats.DLength, stats.MActionCount, stats.FActionCount,
		stats.FActionFromStrangerCount, stats.SActionFromStrangerCount, stats.SActionCount, stats.Loses, winrate),
		upd.Message)
}
