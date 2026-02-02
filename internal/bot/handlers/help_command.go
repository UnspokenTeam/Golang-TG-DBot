package handlers

import (
	"context"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
)

func Help(ctx context.Context, upd telego.Update, services *service_wrapper.Services) {
	text := services.CommandsViper.GetString("help_command_text")
	workers.EnqueueMessage(ctx, text, upd.Message)
}
