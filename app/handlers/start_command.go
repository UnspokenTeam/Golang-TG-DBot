package handlers

import (
	"fmt"
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"middlewares"
)

func Start(ctx *th.Context, upd telego.Update) {
	middlewares.MessageQueue <- tu.Message(
		tu.ID(upd.Message.Chat.ID),
		fmt.Sprintf("Hello %s!", upd.Message.From.FirstName),
	)
}
