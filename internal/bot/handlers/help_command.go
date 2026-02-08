package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/roles"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
)

func Help(ctx context.Context, upd telego.Update, services *service_wrapper.Services) {
	text := services.ConfigCache.GetString("help_command_text")

	if user, findErr := services.PostgresClient.Queries.GetUserByTgId(ctx, upd.Message.From.ID); findErr != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("user find error: %v", findErr))
		return
	} else {
		if roles.Role(user.UserRole) == roles.OWNER {
			text += `
Для владельца (недоступно обычным пользователям) :
  /restart - Перезагрузка бота
  /admin_stats - Приватная статистика за последние сутки
  /promote - Повысить пользователя до администратора __(применяется в ответ на сообщение)__
  /demote - Понизить администратора до пользователя __(применяется в ответ на сообщение)__
  /analyze - Получить ссылку на логи по конкретному пользователю и чаты в Uptrace __(применяется в ответ на сообщение)__
  /echo %s - Проверить как бот видит шаблон сообщения
  /send %s - Сделать глобальную рассылку по шаблону`
		}
	}

	workers.EnqueueMessage(ctx, text, upd.Message)
}
