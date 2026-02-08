package handlers

import (
	"context"
	"fmt"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
	hndUtils "github.com/unspokenteam/golang-tg-dbot/pkg/utils"
)

func Analyze(ctx context.Context, upd telego.Update, services *service_wrapper.Services) {
	if upd.Message.ReplyToMessage == nil || upd.Message.ReplyToMessage.From == nil {
		return
	}
	text := fmt.Sprintf(
		"https://uptrace.%s/spans/1/items?time_dur=86400&system=spans:all&sort_by=_time&sort_desc=true&"+
			"query=perMin(count())+|+quantiles(_dur_ms)+|+_error_rate+|+group+by+_group_id+|+where+chat_id+=+%d+|+"+
			"where+user_id::int+=+%d",
		services.AppViper.GetString("CADDY_DOMAIN"),
		upd.Message.Chat.ID,
		upd.Message.ReplyToMessage.From.ID,
	)

	workers.EnqueueMessage(ctx, hndUtils.GetFormattedLink("Логи", text), upd.Message)
}
