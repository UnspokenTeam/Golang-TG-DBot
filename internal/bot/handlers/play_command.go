package handlers

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
	"github.com/unspokenteam/golang-tg-dbot/internal/db/querier"
	hndUtils "github.com/unspokenteam/golang-tg-dbot/pkg/utils"
)

func Play(ctx context.Context, upd telego.Update, services *service_wrapper.Services) {
	if !hndUtils.IsGroup(upd) {
		workers.EnqueueMessage(ctx, "Команда доступна только в группах.", upd.Message)
		return
	}

	header := fmt.Sprintf(services.ConfigCache.GetString("game_header_text_pattern")+"\n",
		hndUtils.MentionUser(upd.Message.From.FirstName, upd.Message.From.ID))
	cooldown := time.Duration(services.ConfigCache.GetInt("play_command_cooldown")) * time.Minute

	txCtx, tx := services.PostgresClient.NewTx(ctx, &pgx.TxOptions{IsoLevel: pgx.Serializable})
	defer services.PostgresClient.RollbackTx(txCtx, tx)

	_, err := tx.StartGame(txCtx, querier.StartGameParams{
		TgID: upd.Message.Chat.ID,
		Cooldown: pgtype.Interval{
			Microseconds: cooldown.Microseconds(),
			Valid:        true,
		},
	})

	if errors.Is(err, pgx.ErrNoRows) {
		lastTime, _ := tx.GetGameLastTime(txCtx, upd.Message.Chat.ID)
		sendFail(ctx, cooldown, services.ConfigCache, *lastTime, upd, "play_command_text_fail_pattern")
	}
	if err != nil {
		return
	}

	chatMemberCount, err := tx.GetChatMemberCount(txCtx, upd.Message.Chat.ID)
	if err != nil {
		return
	}
	if chatMemberCount < 3 {
		workers.EnqueueMessage(ctx, "Недостаточно людей для игры.", upd.Message)
		return
	}

	memberCount := int(min(10, chatMemberCount))
	inGameUsers := make([]querier.GetUsersForGameCursorBasedRow, 0)
	lostUsers := make([]int64, 0)
	cursor := time.Now().Add(time.Hour)
	var atts, tiebreaker int64 = 0, 0
	for len(inGameUsers) < memberCount && atts < chatMemberCount {
		users, _ := tx.GetUsersForGameCursorBased(txCtx, querier.GetUsersForGameCursorBasedParams{
			ChatTgID:       upd.Message.Chat.ID,
			Limit:          int32(memberCount - len(inGameUsers)),
			Cursor:         cursor,
			TiebreakerTgID: tiebreaker,
		})

		if len(users) == 0 {
			break
		}

		for _, user := range users {
			inGameUsers = append(inGameUsers, user)
			cursor = user.LastMessageAt
			tiebreaker = user.UserTgID
			//if hndUtils.IsUserInChat(ctx, upd.Message.Chat.ID, user.UserTgID, services.TgApiRateLimiter) {
			//	inGameUsers = append(inGameUsers, user)
			//} else {
			//	lostUsers = append(lostUsers, user.UserTgID)
			//}
			atts++
		}
	}
	if len(inGameUsers) < 3 {
		workers.EnqueueMessage(ctx, "Недостаточно людей для игры.", upd.Message)
		return
	}
	_ = cursor.Hour()

	rand.Shuffle(len(inGameUsers), func(i, j int) {
		inGameUsers[i], inGameUsers[j] = inGameUsers[j], inGameUsers[i]
	})

	for _, user := range inGameUsers {
		header += fmt.Sprintf(
			services.ConfigCache.GetString("game_placement_text_pattern"),
			hndUtils.MentionUser(user.UserName, user.UserTgID),
		)
	}

	userStats, err := tx.GetUserStatsByTgId(txCtx, querier.GetUserStatsByTgIdParams{
		ChatTgID: upd.Message.Chat.ID,
		UserTgID: inGameUsers[len(inGameUsers)-1].UserTgID,
	})
	if err != nil {
		return
	}
	header += fmt.Sprintf(
		services.ConfigCache.GetString("game_lose_text_pattern"),
		hndUtils.MentionUser(inGameUsers[len(inGameUsers)-1].UserName, inGameUsers[len(inGameUsers)-1].UserTgID),
		userStats.Loses+1,
	)

	if err = tx.RecordGameLose(txCtx, querier.RecordGameLoseParams{
		ChatTgID: upd.Message.Chat.ID,
		UserTgID: inGameUsers[len(inGameUsers)-1].UserTgID,
	}); err != nil {
		return
	}

	inGameIds := make([]int64, len(inGameUsers))
	for i, u := range inGameUsers {
		inGameIds[i] = u.UserTgID
	}
	if err = tx.RecordGame(txCtx, querier.RecordGameParams{
		ChatTgID: upd.Message.Chat.ID,
		Ids:      inGameIds,
	}); err != nil {
		return
	}

	if err = tx.RemoveLostUsers(txCtx, querier.RemoveLostUsersParams{
		ChatTgID: upd.Message.Chat.ID,
		Ids:      lostUsers,
	}); err != nil {
		return
	}

	services.PostgresClient.CommitTx(txCtx, tx)

	workers.EnqueueMessage(ctx,
		fmt.Sprintf("%s\n\n%s\n%s",
			header,
			hndUtils.GetChatLink("Заходи в наш чат!",
				services.ConfigCache.GetString("bot_public_chat_url")),
			hndUtils.GetFormattedLink("💝 ПОДДЕРЖАТЬ БОТА 💝",
				services.ConfigCache.GetString("support_bot_url"))),
		upd.Message)

}
