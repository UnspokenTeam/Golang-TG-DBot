package handlers

import (
	"context"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/channels"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"go.opentelemetry.io/otel/trace"
)

func Restart(_ context.Context, _ trace.Span, _ telego.Update, _ *service_wrapper.Services) {
	channels.ShutdownChannel <- struct{}{}
}
