package main

import (
	"configs"
	"github.com/joho/godotenv"
	"github.com/unspokenteam/golang-tg-dbot/app"
)

func main() {
	_ = godotenv.Load()
	configs.InitMiddlewareConfig()
	app.LoopApp()
}
