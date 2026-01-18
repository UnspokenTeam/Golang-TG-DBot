package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot"
	configs "github.com/unspokenteam/golang-tg-dbot/internal/config"
	"github.com/unspokenteam/golang-tg-dbot/pkg/logger"
)

func main() {
	_, b, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(b), "../../")
	_ = godotenv.Load(filepath.Join(projectRoot, "example.env"))
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
