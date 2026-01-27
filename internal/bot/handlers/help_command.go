package handlers

import (
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
)

func Help(_ *th.Context, upd telego.Update, services *service_wrapper.Services) {
	text := services.CommandsViper.GetString("help_command_text")
	workers.EnqueueMessage(text, upd.Message)
}
