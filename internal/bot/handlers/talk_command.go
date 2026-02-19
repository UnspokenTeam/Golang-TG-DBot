package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mymmrac/telego"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/workers"
	hndUtils "github.com/unspokenteam/golang-tg-dbot/pkg/utils"
)

func Talk(ctx context.Context, upd telego.Update, services *service_wrapper.Services) {
	text := ""
	if idx := strings.Index(upd.Message.Text, " "); idx != -1 {
		text = upd.Message.Text[idx+1:]
	} else {
		workers.EnqueueMessage(ctx,
			fmt.Sprintf("%s, напиши вместе с командой через пробел то, о чём хочешь поговорить!",
				hndUtils.MentionUser(upd.Message.From.FirstName, upd.Message.From.ID)),
			upd.Message)
	}

	ctx, llmSpan := services.Tracer.Start(ctx, "llm")
	defer llmSpan.End()
	reply, err := services.LlmClient.Generate(ctx, upd.Message.From.ID, text)
	if reply == nil {
		workers.EnqueueMessage(ctx,
			fmt.Sprintf("%s, не торопись! Разговаривать с ботом можно раз в минуту.",
				hndUtils.MentionUser(upd.Message.From.FirstName, upd.Message.From.ID)),
			upd.Message)
		return
	}
	if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("llm err: %v", err))
		workers.EnqueueMessage(ctx,
			fmt.Sprintf("%s, спроси меня об этом позже.",
				hndUtils.MentionUser(upd.Message.From.FirstName, upd.Message.From.ID)),
			upd.Message)
		return
	}

	workers.EnqueueMessage(ctx,
		fmt.Sprintf("%s! %s",
			hndUtils.MentionUser(upd.Message.From.FirstName, upd.Message.From.ID), reply.Response),
		upd.Message)
}
