package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/joho/godotenv"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot"
	"github.com/unspokenteam/golang-tg-dbot/pkg/logger"
	"github.com/unspokenteam/golang-tg-dbot/pkg/utils"
)

func main() {
	_ = godotenv.Load(utils.GetDevEnvFileLocation())
	if utils.IsEnvProduction() {
		// todo: replace logger to slog
		logger.InitLogger("", "", true)
		log.SetOutput(&logger.TelegoLogger{})
	}
	appCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	app.Run(appCtx, cancel)
}
