package handlers

import (
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
)

func PreprocessUser(ctx *th.Context, upd telego.Update, services *service_wrapper.Services) {
	// todo: db user init
}
