package middlewares

import (
	"context"
	"errors"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoapi"
	"golang.org/x/time/rate"
	"log"
	"os"
	"strconv"
	"time"
)

var (
	MessageQueue = make(chan *telego.SendMessageParams, 1000)
	limiter      *rate.Limiter
	botInstance  *telego.Bot
	ctx          context.Context
)

func InitQueue(appCtx context.Context, bot *telego.Bot) {
	ctx = appCtx
	botInstance = bot
	rps, err := strconv.Atoi(os.Getenv("RPS_LIMIT"))
	if err != nil {
		panic("Failed to init queue. Cannot find RPS_LIMIT.")
	}
	limiter = rate.NewLimiter(rate.Every(time.Second/time.Duration(rps)), 1)
	go startWorker()
}

func tryToSendWithRetry(msg *telego.SendMessageParams) bool {
	var (
		err    error
		apiErr *telegoapi.Error
	)

	for i := 0; i < 3; i++ {
		_, err = botInstance.SendMessage(ctx, msg)
		if err == nil {
			return true
		}
		if errors.As(err, &apiErr) && apiErr.ErrorCode == 429 {
			time.Sleep(time.Second * time.Duration(apiErr.Parameters.RetryAfter+1))
		} else {
			log.Println(err)
		}
	}
	return false
}

func startWorker() {
	for msg := range MessageQueue {
		if err := limiter.Wait(context.Background()); err != nil {
			log.Println("rate limiter error:", err)
			continue
		}
		tryToSendWithRetry(msg)
	}
}
