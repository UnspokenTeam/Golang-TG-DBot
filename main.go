package main

import (
	"configs"
	"context"
	"github.com/joho/godotenv"
	"github.com/unspokenteam/golang-tg-dbot/app"
	"log"
	"logger"
	"os"
	"os/signal"
)

func main() {
	_ = godotenv.Load()
	env := os.Getenv("GO_ENV")
	if env == "PRODUCTION" {
		logger.InitLogger("", "", true)
		log.SetOutput(&logger.TelegoLogger{})
	}
	configs.InitMiddlewareConfig()
	mainContext, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	start := app.InitAppLooper(env, mainContext)
	start <- struct{}{}
	app.LoopApp()
}
