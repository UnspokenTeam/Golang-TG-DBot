package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/db/querier"
	"github.com/unspokenteam/golang-tg-dbot/pkg/utils"
)

func PreprocessUser(ctx context.Context, upd telego.Update, services *service_wrapper.Services) *querier.User {
	memberCount := utils.GetChatMemberCount(ctx, upd.Message.Chat.ID)

	if upd.Message.ReplyToMessage != nil {
		if err := services.PostgresClient.Queries.InitChatUserData(
			ctx,
			querier.InitChatUserDataParams{
				PUserTgID:     upd.Message.ReplyToMessage.From.ID,
				PUserTag:      upd.Message.ReplyToMessage.From.Username,
				PUserName:     upd.Message.ReplyToMessage.From.FirstName,
				PUserLastname: upd.Message.ReplyToMessage.From.LastName,
				PChatTgID:     upd.Message.ReplyToMessage.Chat.ID,
				PChatType:     upd.Message.ReplyToMessage.Chat.Type,
				PChatName:     upd.Message.ReplyToMessage.Chat.FirstName,
				PMemberCount:  int32(memberCount),
			}); err != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("reply to user preprocess error: %v", err))
			return nil
		}
	}
	if err := services.PostgresClient.Queries.InitChatUserData(
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
		}); err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("user preprocess error: %v", err))
		return nil
	}
	if user, findErr := services.PostgresClient.Queries.GetUserByTgId(ctx, upd.Message.From.ID); findErr != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("user find error: %v", findErr))
		return nil
	} else {
		slog.InfoContext(ctx, "User preprocessed...")
		return &user
	}
}
