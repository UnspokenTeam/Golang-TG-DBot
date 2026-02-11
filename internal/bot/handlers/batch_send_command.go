package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
	hndUtils "github.com/unspokenteam/golang-tg-dbot/pkg/utils"
)

func Broadcast(ctx context.Context, upd telego.Update, services *service_wrapper.Services) {
	if idx := strings.Index(upd.Message.Text, " "); idx != -1 {
		batchId := workers.AddBatchTask(upd.Message.Text[idx+1:])
		text := fmt.Sprintf(
			"https://uptrace.%s/spans/1/items?&time_dur=86400&system=log:all&sort_by=_time&sort_desc=true&"+
				"query=perMin(count())+|+max(_time)+|+group+by+_group_id+|+where+batch_id::str+=+\"%s\"",
			services.AppViper.GetString("CADDY_DOMAIN"),
			batchId,
		)

		workers.EnqueueMessage(ctx, fmt.Sprintf("Запущена рассылка # %s\n\n%s", batchId,
			hndUtils.GetFormattedLink("Логи", strings.ReplaceAll(text, "\"", "%22"))), upd.Message)
	}
}
