package handlers

import (
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/unspokenteam/golang-tg-dbot/internal/middlewares"
	hndUtils "github.com/unspokenteam/golang-tg-dbot/pkg/utils"
)

func Start(ctx *th.Context, upd telego.Update) {
	text := fmt.Sprintf(
		"Привет, %s!\n\nДля продолжения взаимодействия с ботом используй команду /help.",
		hndUtils.MentionUser(upd.Message.From.FirstName, upd.Message.From.ID),
	)
	if !hndUtils.IsGroup(upd) {
		text += fmt.Sprintf(
			"\n\nДобавь бота в группу для расширения функционала:\n\n%s\nИЛИ\n%s",
			hndUtils.GetAddToGroupLink("НАЖМИ, ЧТОБЫ ДОБАВИТЬ БОТА В ГРУППУ"),
			hndUtils.GetSendInviteLink("НАЖМИ, ЧТОБЫ ОТПРАВИТЬ БОТА АДМИНУ ГРУППЫ", "НАЖМИ, ЧТОБЫ ДОБАВИТЬ БОТА В ГРУППУ"),
		)
	}
	middlewares.MessageQueue <- hndUtils.GetMsgSendParams(text, upd.Message)
}
