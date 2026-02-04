package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/roles"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
	"github.com/unspokenteam/golang-tg-dbot/internal/db/querier"
	hndUtils "github.com/unspokenteam/golang-tg-dbot/pkg/utils"
)

func Demote(ctx context.Context, upd telego.Update, services *service_wrapper.Services) {
	if upd.Message.ReplyToMessage == nil || upd.Message.ReplyToMessage.From.ID == upd.Message.From.ID {
		return
	}
	err := services.PostgresClient.Queries.SetUserRoleByTgId(ctx, querier.SetUserRoleByTgIdParams{
		UserRole: string(roles.USER),
		TgID:     upd.Message.ReplyToMessage.From.ID,
	})
	if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Demote user err: %s", err))
		return
	}
	text := fmt.Sprintf("Роль %s успешно понижена до %s",
		hndUtils.MentionUser(upd.Message.ReplyToMessage.From.FirstName, upd.Message.ReplyToMessage.From.ID),
		roles.USER)
	workers.EnqueueMessage(ctx, text, upd.Message)
}
