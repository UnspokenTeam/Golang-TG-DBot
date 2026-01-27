package handlers

import (
	th "github.com/mymmrac/telego/telegohandler"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/roles"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
)

func CheckRoleAccess(ctx *th.Context, roles []roles.Role, services *service_wrapper.Services) {
	//todo: db check
}
