package handlers

import (
	"context"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
	"go.opentelemetry.io/otel/trace"
)

func Help(ctx context.Context, span trace.Span, upd telego.Update, services *service_wrapper.Services) {
	text := services.CommandsViper.GetString("help_command_text")
	workers.EnqueueMessage(ctx, text, upd.Message)
}
