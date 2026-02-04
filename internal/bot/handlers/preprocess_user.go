package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/db/querier"
	"github.com/unspokenteam/golang-tg-dbot/pkg/utils"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func PreprocessUser(ctx context.Context, span trace.Span, upd telego.Update, services *service_wrapper.Services) *querier.User {
	memberCount := utils.GetChatMemberCount(ctx, upd.Message.Chat.ID)
	err := services.PostgresClient.Queries.InitChatUserData(
		ctx,
		querier.InitChatUserDataParams{
			PUserTgID:     upd.Message.From.ID,
			PUserTag:      upd.Message.From.Username,
			PUserName:     upd.Message.From.FirstName,
			PUserLastname: upd.Message.From.LastName,
			PChatTgID:     upd.Message.Chat.ID,
			PChatType:     upd.Message.Chat.Type,
			PChatName:     upd.Message.Chat.FirstName,
			PMemberCount:  int32(memberCount),
		})
	if err != nil {
		span.RecordError(fmt.Errorf("user preprocess error: %v", err))
		span.SetStatus(codes.Error, err.Error())
		return nil
	}
	if user, findErr := services.PostgresClient.Queries.GetUserByTgId(ctx, upd.Message.From.ID); findErr != nil {
		span.RecordError(fmt.Errorf("user find error: %v", findErr))
		span.SetStatus(codes.Error, findErr.Error())
		return nil
	} else {
		slog.InfoContext(ctx, "User preprocessed...")
		return &user
	}
}
