package handlers

import (
	"context"
	"log/slog"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"go.opentelemetry.io/otel/trace"
)

func PreprocessUser(ctx context.Context, span trace.Span, upd telego.Update, services *service_wrapper.Services) {
	// todo: db user init
	slog.InfoContext(ctx, "User preprocessed...")
}
