package handlers

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mymmrac/telego"
	"github.com/shopspring/decimal"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
	"github.com/unspokenteam/golang-tg-dbot/internal/configs"
	"github.com/unspokenteam/golang-tg-dbot/internal/db/querier"
	hndUtils "github.com/unspokenteam/golang-tg-dbot/pkg/utils"
)

func sendFail(
	ctx context.Context,
	cooldown time.Duration,
	configCache *configs.ConfigCache,
	lastTime time.Time,
	upd telego.Update) {
	toWait := cooldown - time.Since(lastTime)
	text := configCache.GetString("up_text_fail_pattern")
	workers.EnqueueMessage(ctx,
		fmt.Sprintf(text,
			hndUtils.MentionUser(upd.Message.From.FirstName, upd.Message.From.ID),
			int(toWait.Minutes()),
			int(toWait.Seconds())%60,
		), upd.Message)
}

func Up(ctx context.Context, upd telego.Update, services *service_wrapper.Services) {
	if !hndUtils.IsGroup(upd) {
		workers.EnqueueMessage(ctx, "Команда доступна только в группах.", upd.Message)
	}
	lastTime, _ := services.PostgresClient.Queries.GetLastTimeDAction(ctx, querier.GetLastTimeDActionParams{
		ChatTgID: upd.Message.Chat.ID,
		UserTgID: upd.Message.From.ID,
	})
	cooldown := time.Duration(services.ConfigCache.GetInt("up_command_cooldown")) * time.Minute
	if lastTime != nil && time.Since(*lastTime) < cooldown {
		sendFail(ctx, cooldown, services.ConfigCache, *lastTime, upd)
		return
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	x := rng.Intn(101) // 0..100

	sign := -1
	if rng.Float64() >= 0.30 {
		sign = 1
	}

	incr := decimal.NewFromInt(int64(x * sign)).Div(decimal.NewFromInt(10))
	size, err := services.PostgresClient.Queries.GrowD(ctx, querier.GrowDParams{
		DLength:  incr,
		ChatTgID: upd.Message.Chat.ID,
		UserTgID: upd.Message.From.ID,
	})

	if errors.Is(err, pgx.ErrNoRows) {
		lastTime, _ = services.PostgresClient.Queries.GetLastTimeDAction(ctx, querier.GetLastTimeDActionParams{
			ChatTgID: upd.Message.Chat.ID,
			UserTgID: upd.Message.From.ID,
		})
		sendFail(ctx, cooldown, services.ConfigCache, *lastTime, upd)
	}
	if err != nil {
		return
	}

	text := services.ConfigCache.GetString("up_text_pattern")
	action := "уменьшил"
	if sign == 1 {
		action = "увеличил"
	}

	workers.EnqueueMessage(ctx,
		fmt.Sprintf(text,
			hndUtils.MentionUser(upd.Message.From.FirstName, upd.Message.From.ID),
			action,
			incr.Abs(),
			size,
			hndUtils.GetChatLink("Заходи в наш чат!", services.ConfigCache.GetString("bot_public_chat_url")),
			hndUtils.GetFormattedLink("💝 ПОДДЕРЖАТЬ БОТА 💝", services.ConfigCache.GetString("support_bot_url")),
		), upd.Message)

}
