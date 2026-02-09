package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
	"github.com/unspokenteam/golang-tg-dbot/internal/db/querier"
	hndUtils "github.com/unspokenteam/golang-tg-dbot/pkg/utils"
)

func SAction(ctx context.Context, upd telego.Update, services *service_wrapper.Services) {
	if !hndUtils.IsGroup(upd) {
		workers.EnqueueMessage(ctx, "Команда доступна только в группах.", upd.Message)
		return
	}

	var isStrangerValid = hndUtils.IsValidUser(upd.Message.ReplyToMessage)
	if !isStrangerValid && upd.Message.ReplyToMessage == nil {
		workers.EnqueueMessage(ctx, fmt.Sprintf(services.ConfigCache.GetString("s_action_tutorial"),
			hndUtils.MentionUser(upd.Message.From.FirstName, upd.Message.From.ID)), upd.Message)
		return
	}

	cooldown := time.Duration(services.ConfigCache.GetInt("s_action_command_cooldown")) * time.Minute

	txCtx, tx := services.PostgresClient.NewTx(ctx, &pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	defer services.PostgresClient.RollbackTx(txCtx, tx)

	count, err := tx.TryPerformSAction(txCtx, querier.TryPerformSActionParams{
		ChatTgID: upd.Message.Chat.ID,
		UserTgID: upd.Message.From.ID,
		Cooldown: pgtype.Interval{
			Microseconds: cooldown.Microseconds(),
			Valid:        true,
		},
	})

	if errors.Is(err, pgx.ErrNoRows) {
		lastTime, _ := tx.GetLastTimeSAction(txCtx, querier.GetLastTimeSActionParams{
			ChatTgID: upd.Message.Chat.ID,
			UserTgID: upd.Message.From.ID,
		})
		sendFail(ctx, cooldown, services.ConfigCache, *lastTime, upd, "s_action_text_fail_pattern")
	}
	if err != nil {
		return
	}

	var actionTo = ""
	if isStrangerValid {
		if err = tx.ConfirmSAction(txCtx, querier.ConfirmSActionParams{
			ChatTgID: upd.Message.Chat.ID,
			UserTgID: upd.Message.ReplyToMessage.From.ID,
		}); err != nil {
			return
		}
		actionTo = hndUtils.MentionUser(upd.Message.ReplyToMessage.From.FirstName, upd.Message.ReplyToMessage.From.ID)
	} else {
		actionTo = hndUtils.GetStrangerName(upd.Message.ReplyToMessage)
	}

	services.PostgresClient.CommitTx(txCtx, tx)

	text := services.ConfigCache.GetString("s_action_text_pattern")
	phrase := randomChoice(services.ConfigCache.GetStringSlice("s_action_phrases"))
	if upd.Message.From.ID == upd.Message.ReplyToMessage.From.ID {
		newest, actionErr := services.PostgresClient.Queries.GetRandomActionFromNewest(ctx, true)
		if actionErr != nil {
			newest = querier.GetRandomActionFromNewestRow{
				ID:     -1,
				Action: ", потому что очень себя любит",
			}
		} else {
			newest.Action = fmt.Sprintf(" и %s", newest.Action)
		}
		slog.DebugContext(ctx, "Action performed", "action_id", newest.ID)
		phrase += newest.Action
	}

	workers.EnqueueMessage(ctx,
		fmt.Sprintf(text,
			actionTo,
			hndUtils.MentionUser(upd.Message.From.FirstName, upd.Message.From.ID),
			phrase,
			count,
			hndUtils.GetChatLink("Заходи в наш чат!", services.ConfigCache.GetString("bot_public_chat_url")),
			hndUtils.GetFormattedLink("💝 ПОДДЕРЖАТЬ БОТА 💝", services.ConfigCache.GetString("support_bot_url")),
		), upd.Message)

}
