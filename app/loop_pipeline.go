package app

import (
	"context"
	"fmt"
	"github.com/mymmrac/telego"
	"golang.org/x/time/rate"
	"logger"
	"os"
	"sync"
	"time"
)

var (
	cancelFunc     context.CancelFunc
	restartMu      sync.Mutex
	restartLimiter *rate.Limiter
	env            string
)

func InitAppLooper(goEnv string) {
	env = goEnv
	restartLimiter = rate.NewLimiter(rate.Every(time.Minute*10), 1)
}

func LoopApp() {
	ctx, cancel := context.WithCancel(context.Background())
	cancelFunc = cancel

	Done = make(chan struct{}, 1)
	Run(env, ctx)
}

func RestartApp() {
	restartMu.Lock()
	defer restartMu.Unlock()
	if cancelFunc != nil {
		cancelFunc()
		<-Done
	}
	go LoopApp()
}

func HealthCheckWithRestart(bot *telego.Bot, ctx context.Context) {
	for {
		time.Sleep(time.Hour)
		if _, err := bot.GetMe(ctx); err != nil {
			RestartApp()
		}
	}
}

func RestartAfterPanic() {
	if err := restartLimiter.Wait(context.Background()); err != nil {
		logger.LogError(fmt.Sprintf("Restart rate limiter error: %s", err), "workerRateLimitError", nil)
		cancelFunc()
		<-Done
		os.Exit(1)
	}
	RestartApp()
}
