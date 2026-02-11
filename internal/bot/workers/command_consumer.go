package workers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mymmrac/telego/telegohandler"
)

func gracefulShutdownConsumer(ctx context.Context, handler *telegohandler.BotHandler) {
	slog.InfoContext(ctx, "Shutting down bot...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := handler.StopWithContext(shutdownCtx); err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Error stopping handler: %v", err))
	}
	slog.InfoContext(ctx, "Bot has been shut down successfully.")
}

func RunCommandConsumer(ctx context.Context, handler *telegohandler.BotHandler) {
	defer gracefulShutdownConsumer(ctx, handler)

	go func() {
		slog.InfoContext(ctx, "Starting bot...")
		if err := handler.Start(); err != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("Error starting main handler: %v", err))
		}
	}()

	<-ctx.Done()
}
