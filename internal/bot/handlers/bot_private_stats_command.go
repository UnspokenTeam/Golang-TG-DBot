package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
)

func PrivateStats(ctx context.Context, upd telego.Update, services *service_wrapper.Services) {
	stats, err := services.PostgresClient.Queries.GetAllAdminTimeStats(ctx)
	if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Get all time stats: %s", err))
		return
	}
	text := fmt.Sprintf(
		"Новых пользователей сегодня: %d\nНовых чатов сегодня: %d\nПотерянных чатов сегодня: %d\nПотерянных людей сегодня: %d",
		stats.TodayNewUsers, stats.TodayNewChats, stats.TodayBurnedChats, stats.TodayBurnedUsers)
	workers.EnqueueMessage(ctx, text, upd.Message)
}
