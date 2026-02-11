package handlers

import (
	"context"
	"fmt"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
)

func PrivateStats(ctx context.Context, upd telego.Update, services *service_wrapper.Services) {
	stats, err := services.PostgresClient.Queries.GetAllAdminTimeStats(ctx)
	if err != nil {
		return
	}
	text := fmt.Sprintf(
		`Активных пользователей за последние 24 часа: %d
Активных чатов за последние 24 часа: %d
Новых чатов за последние 24 часа: %d
Новых пользователей за последние 24 часа: %d`,
		stats.TodayActiveUsers, stats.TodayActiveChats, stats.TodayNewChats, stats.TodayNewUsers)
	workers.EnqueueMessage(ctx, text, upd.Message)
}
