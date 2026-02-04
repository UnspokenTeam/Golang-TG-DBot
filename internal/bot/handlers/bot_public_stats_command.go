package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
)

func PublicStats(ctx context.Context, upd telego.Update, services *service_wrapper.Services) {
	stats, err := services.PostgresClient.Queries.GetAllTimeStats(ctx)
	if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Get all time stats: %s", err))
		return
	}
	text := fmt.Sprintf("Человек, воспользовавшихся ботом: %d\nКоличество групп, воспользовавшихся ботом: %d",
		stats.TotalUsers, stats.TotalChats)
	workers.EnqueueMessage(ctx, text, upd.Message)
}
