package middlewares

import (
	"context"
	"errors"
	"fmt"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoapi"
	hndUtils "github.com/unspokenteam/golang-tg-dbot/app/handler_utils"
	"golang.org/x/time/rate"
	"logger"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	MessageQueue chan *telego.SendMessageParams
	limiter      *rate.Limiter
	BotInstance  *telego.Bot
	Ctx          context.Context
	wg           sync.WaitGroup
)

func InitQueue(appCtx context.Context, bot *telego.Bot) {
	Ctx = appCtx
	BotInstance = bot
	MessageQueue = make(chan *telego.SendMessageParams, 1000)
	rps, err := strconv.Atoi(os.Getenv("RPS_LIMIT"))
	if err != nil {
		logger.LogFatal(fmt.Sprintf("Failed to init queue. Cannot find RPS_LIMIT. %s", err.Error()), "configuring", nil)
	}
	limiter = rate.NewLimiter(rate.Every(time.Second/time.Duration(rps)), 1)
	wg.Add(1)
	go startWorker()
}

func tryToSendWithRetry(msg *telego.SendMessageParams) bool {
	var (
		err    error
		apiErr *telegoapi.Error
	)

	msg = msg.WithLinkPreviewOptions(
		&telego.LinkPreviewOptions{IsDisabled: true},
	).WithParseMode(
		telego.ModeMarkdownV2,
	).WithText(
		hndUtils.EscapeMarkdownV2Smart(msg.Text))

	fmt.Println(msg.Text)
	for i := 0; i < 3; i++ {
		_, err = BotInstance.SendMessage(Ctx, msg)
		if err == nil {
			return true
		} else if errors.As(err, &apiErr) && apiErr.ErrorCode == 429 {
			time.Sleep(time.Second * time.Duration(apiErr.Parameters.RetryAfter+1))
		} else {
			logger.LogError(fmt.Sprintf("Failed to send message. %s", err.Error()), "sendMessageError", msg)
		}
	}
	return false
}

func startWorker() {
	defer wg.Done()
	for {
		select {
		case <-Ctx.Done():
			logger.LogInfo("Shutting down worker...", "gracefulShutdown", nil)
			return
		case msg, ok := <-MessageQueue:
			if !ok {
				return
			}
			if err := limiter.Wait(Ctx); err != nil {
				logger.LogError(fmt.Sprintf("rate limiter error: %s", err), "workerRateLimitError", nil)
				continue
			}
			tryToSendWithRetry(msg)
		}
	}
}

func ShutdownQueue() {
	close(MessageQueue)
	wg.Wait()
}
