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
		`Активных пользователей сегодня: %d
Активных чатов сегодня: %d
Неактивных чатов сегодня: %d
Неактивных людей сегодня: %d
Новых чатов сегодня: %d
Новых пользователей сегодня: %d`,
		stats.TodayActiveUsers, stats.TodayActiveChats, stats.TodayLazyChats,
		stats.TodayLazyUsers, stats.TodayNewChats, stats.TodayNewUsers)
	workers.EnqueueMessage(ctx, text, upd.Message)
}
