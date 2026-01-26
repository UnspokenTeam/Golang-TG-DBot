package handlers

import (
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/channels"
)

func Restart(_ *th.Context, _ telego.Update) {
	channels.ShutdownChannel <- struct{}{}
}
