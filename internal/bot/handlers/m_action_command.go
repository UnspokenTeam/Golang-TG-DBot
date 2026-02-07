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

func MAction(ctx context.Context, upd telego.Update, services *service_wrapper.Services) {
	if !hndUtils.IsGroup(upd) {
		workers.EnqueueMessage(ctx, "Команда доступна только в группах.", upd.Message)
		return
	}

	cooldown := time.Duration(services.ConfigCache.GetInt("m_action_command_cooldown")) * time.Minute

	count, err := services.PostgresClient.Queries.TryPerformMAction(ctx, querier.TryPerformMActionParams{
		ChatTgID: upd.Message.Chat.ID,
		UserTgID: upd.Message.From.ID,
		Cooldown: pgtype.Interval{
			Microseconds: cooldown.Microseconds(),
			Valid:        true,
		},
	})

	if errors.Is(err, pgx.ErrNoRows) {
		lastTime, _ := services.PostgresClient.Queries.GetLastTimeMAction(ctx, querier.GetLastTimeMActionParams{
			ChatTgID: upd.Message.Chat.ID,
			UserTgID: upd.Message.From.ID,
		})
		sendFail(ctx, cooldown, services.ConfigCache, *lastTime, upd, "m_action_text_fail_pattern")
	}
	if err != nil {
		return
	}

	text := services.ConfigCache.GetString("m_action_text_pattern")
	phrase := randomChoice(services.ConfigCache.GetStringSlice("m_action_phrases"))
	if upd.Message.ReplyToMessage != nil && upd.Message.From.ID == upd.Message.ReplyToMessage.From.ID {
		newest, actionErr := services.PostgresClient.Queries.GetYourselfRandomActionFromNewest(ctx)
		if actionErr != nil {
			newest = querier.GetYourselfRandomActionFromNewestRow{
				ID:     -1,
				Action: ", потому что очень себя любит",
			}
		} else {
			newest.Action = fmt.Sprintf(" и %s", newest.Action)
		}
		slog.DebugContext(ctx, "Action performed", "action_id", newest.ID)
		phrase += newest.Action
	}

	actionTo := ""
	if upd.Message.ReplyToMessage != nil {
		actionTo = fmt.Sprintf(" на %s",
			hndUtils.MentionUser(upd.Message.ReplyToMessage.From.FirstName, upd.Message.ReplyToMessage.From.ID))
	}

	workers.EnqueueMessage(ctx,
		fmt.Sprintf(text,
			hndUtils.MentionUser(upd.Message.From.FirstName, upd.Message.From.ID),
			actionTo,
			phrase,
			count,
			hndUtils.GetChatLink("Заходи в наш чат!", services.ConfigCache.GetString("bot_public_chat_url")),
			hndUtils.GetFormattedLink("💝 ПОДДЕРЖАТЬ БОТА 💝", services.ConfigCache.GetString("support_bot_url")),
		), upd.Message)

}
