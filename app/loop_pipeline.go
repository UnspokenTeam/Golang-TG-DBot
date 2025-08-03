package app

import (
	"context"
	"github.com/mymmrac/telego"
	"time"
)

var cancelFunc context.CancelFunc

func LoopApp() {
	if cancelFunc != nil {
		cancelFunc()
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancelFunc = cancel

	Run(ctx)
}

func RestartApp() {
	LoopApp()
}

func HealthCheckWithRestart(bot *telego.Bot, ctx context.Context) {
	for {
		time.Sleep(time.Hour)
		if _, err := bot.GetMe(ctx); err != nil {
			RestartApp()
		}
	}
}
