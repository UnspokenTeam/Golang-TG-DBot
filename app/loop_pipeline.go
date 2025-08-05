package app

import (
	"context"
	"fmt"
	"github.com/mymmrac/telego"
	ch "github.com/unspokenteam/golang-tg-dbot/app/app_channels"
	"golang.org/x/time/rate"
	"logger"
	"os"
	"sync"
	"time"
)

var (
	restartMu      sync.Mutex
	restartLimiter *rate.Limiter
	env            string
	mainContext    context.Context
	appContext     context.Context
	cancel         context.CancelFunc
)

func InitAppLooper(goEnv string, mainCtx context.Context) chan struct{} {
	env = goEnv
	mainContext = mainCtx
	restartLimiter = rate.NewLimiter(rate.Every(time.Minute*10), 1)
	ch.InitChannels()
	return ch.StartChannel
}

func runApp() {
	newCtx, newCancel := context.WithCancel(mainContext)
	appContext, cancel = newCtx, newCancel
	Run(env, newCtx)
}

func restartApp() {
	restartMu.Lock()
	defer restartMu.Unlock()
	if cancel != nil {
		cancel()
		<-Done
	}
	time.Sleep(time.Second * 10)
	go runApp()
}

func stopApp() {
	restartMu.Lock()
	if cancel != nil {
		cancel()
		<-Done
	}
	os.Exit(1)
}

func LoopApp() {
	for {
		select {
		case <-mainContext.Done():
			return
		case <-ch.StartChannel:
			go runApp()
		case <-ch.RestartChannel:
			go restartApp()
		case <-ch.StopChannel:
			go stopApp()
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func HealthCheckWithRestart(bot *telego.Bot, ctx context.Context) {
	for {
		time.Sleep(time.Hour)
		if _, getBotErr := bot.GetMe(ctx); getBotErr != nil {
			ch.RestartChannel <- struct{}{}
		}
	}
}

func RestartAfterPanic() {
	if limiterErr := restartLimiter.Wait(appContext); limiterErr != nil {
		logger.LogError(fmt.Sprintf("Restart rate limiter error: %s", limiterErr), "workerRateLimitError", nil)
		ch.StopChannel <- struct{}{}
	}
	ch.RestartChannel <- struct{}{}
}
