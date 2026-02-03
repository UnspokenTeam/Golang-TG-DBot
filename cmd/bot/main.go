package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/unspokenteam/golang-tg-dbot/internal/bot"
)

func main() {
	appCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	app.Run(appCtx, cancel)
}
