package handlers

import (
	"context"
	"fmt"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
	hndUtils "github.com/unspokenteam/golang-tg-dbot/pkg/utils"
)

func Start(ctx context.Context, upd telego.Update, _ *service_wrapper.Services) {
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
	workers.EnqueueMessage(ctx, text, upd.Message)
}
