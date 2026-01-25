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
	"github.com/unspokenteam/golang-tg-dbot/pkg/utils"
)

func main() {
	_, b, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(b), "../../")
	_ = godotenv.Load(filepath.Join(projectRoot, "example.env"))
	env := utils.GetEnv()
	if env == "PRODUCTION" {
		logger.InitLogger("", "", true)
		log.SetOutput(&logger.TelegoLogger{})
	}
	configs.InitMiddlewareConfig()
	appCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	app.Run(appCtx, cancel)
}
