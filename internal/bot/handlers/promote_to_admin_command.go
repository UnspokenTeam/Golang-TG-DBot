package handlers

import (
	"context"
	"fmt"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/roles"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
	"github.com/unspokenteam/golang-tg-dbot/internal/db/querier"
	hndUtils "github.com/unspokenteam/golang-tg-dbot/pkg/utils"
)

func Promote(ctx context.Context, upd telego.Update, services *service_wrapper.Services) {
	if upd.Message.ReplyToMessage == nil || upd.Message.ReplyToMessage.From.ID == upd.Message.From.ID {
		return
	}

	err := services.PostgresClient.Queries.SetUserRoleByTgId(ctx, querier.SetUserRoleByTgIdParams{
		UserRole: string(roles.ADMIN),
		TgID:     upd.Message.ReplyToMessage.From.ID,
	})
	if err != nil {
		return
	}

	text := fmt.Sprintf("Роль %s успешно повышена до %s",
		hndUtils.MentionUser(upd.Message.ReplyToMessage.From.FirstName, upd.Message.ReplyToMessage.From.ID),
		roles.ADMIN)
	workers.EnqueueMessage(ctx, text, upd.Message)
}
