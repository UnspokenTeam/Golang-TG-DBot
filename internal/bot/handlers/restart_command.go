package handlers

import (
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/channels"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
)

func Restart(_ *th.Context, _ telego.Update, _ *service_wrapper.Services) {
	channels.ShutdownChannel <- struct{}{}
}
