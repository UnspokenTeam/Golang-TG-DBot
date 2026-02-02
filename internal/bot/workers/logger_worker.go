package workers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/uptrace/uptrace-go/uptrace"
)

func GracefulShutdownLogger(ctx context.Context) {
	defer func(ctx context.Context) {
		if err := uptrace.Shutdown(ctx); err != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("Error in uptrace shutdown: %v", err))
		}
	}(ctx)

	<-ctx.Done()
}
