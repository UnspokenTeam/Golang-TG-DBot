package main

import (
	"configs"
	"github.com/joho/godotenv"
	"github.com/unspokenteam/golang-tg-dbot/app"
	"log"
	"logger"
	"os"
)

func main() {
	_ = godotenv.Load()
	env := os.Getenv("GO_ENV")
	if env == "PRODUCTION" {
		logger.InitLogger("", "", true)
		log.SetOutput(&logger.TelegoLogger{})
	}
	configs.InitMiddlewareConfig()
	app.InitAppLooper(env)
	app.LoopApp()
}
