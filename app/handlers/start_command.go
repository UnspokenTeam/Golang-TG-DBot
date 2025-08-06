package handlers

import (
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	hndUtils "github.com/unspokenteam/golang-tg-dbot/app/handler_utils"
	"middlewares"
)

func Start(ctx *th.Context, upd telego.Update) {
	middlewares.MessageQueue <- &telego.SendMessageParams{
		ChatID:    tu.ID(upd.Message.Chat.ID),
		Text:      hndUtils.EscapeMarkdownV2Smart("") + hndUtils.MentionUser(upd.Message.From.Username, upd.Message.From.ID),
		ParseMode: telego.ModeMarkdownV2,
	}
}
