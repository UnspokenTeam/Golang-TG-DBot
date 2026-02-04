package handlers

import (
	"context"
	"log/slog"

	"github.com/unspokenteam/golang-tg-dbot/internal/bot/roles"
	"github.com/unspokenteam/golang-tg-dbot/internal/db/querier"
)

func CheckRoleAccess(ctx context.Context, user *querier.User, allowedRoles []roles.Role) bool {
	userRole := roles.Role(user.UserRole)

	for _, role := range allowedRoles {
		if userRole == role {
			slog.InfoContext(ctx, "Checked role access...")
			return true
		}
	}

	slog.WarnContext(ctx, "User role has been rejected", "user", user, "roles", allowedRoles)
	return false
}
