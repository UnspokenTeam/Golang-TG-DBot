package handlers

import (
	"context"
	"strings"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
)

func Echo(ctx context.Context, upd telego.Update, _ *service_wrapper.Services) {
	if idx := strings.Index(upd.Message.Text, " "); idx != -1 {
		workers.EnqueueMessage(ctx, upd.Message.Text[idx+1:], upd.Message)
	}
}
