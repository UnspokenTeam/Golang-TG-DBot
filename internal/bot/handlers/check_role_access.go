package handlers

import (
	"context"
	"log/slog"

	"github.com/unspokenteam/golang-tg-dbot/internal/bot/roles"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"go.opentelemetry.io/otel/trace"
)

func CheckRoleAccess(ctx context.Context, span trace.Span, roles []roles.Role, services *service_wrapper.Services) {
	//todo: db check
	slog.InfoContext(ctx, "Checked role access...")
}
