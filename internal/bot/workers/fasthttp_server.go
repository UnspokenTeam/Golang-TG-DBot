package workers

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/valyala/fasthttp"
)

func gracefulShutdownServer(ctx context.Context, srv *fasthttp.Server) {
	slog.InfoContext(ctx, "Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.ShutdownWithContext(shutdownCtx); err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Server shutdown error: %v", err))
	}
	slog.InfoContext(ctx, "Server has been shut down successfully.")
}

func StartServer(ctx context.Context, srv *fasthttp.Server, port int) {
	defer gracefulShutdownServer(ctx, srv)

	go func() {
		slog.InfoContext(ctx, "Starting server...")
		next := srv.Handler
		srv.Handler = func(ctx *fasthttp.RequestCtx) {
			if bytes.Equal(ctx.Path(), []byte("/healthz")) {
				ctx.SetStatusCode(fasthttp.StatusOK)
				ctx.SetContentType("text/plain; charset=utf-8")
				ctx.SetBodyString("ok")
				return
			}
			next(ctx)
		}
		if err := srv.ListenAndServe(fmt.Sprintf(":%d", port)); err != nil {
			slog.ErrorContext(ctx, fmt.Sprintf("Server error: %v", err))
		}
	}()

	<-ctx.Done()
}
