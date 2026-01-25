package workers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoapi"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/channels"
	"github.com/unspokenteam/golang-tg-dbot/pkg/logger"
	hndUtils "github.com/unspokenteam/golang-tg-dbot/pkg/utils"
	"golang.org/x/time/rate"
)

var (
	limiter     *rate.Limiter
	botInstance *telego.Bot
	ctx         context.Context
	wg          sync.WaitGroup
)

func consumeMessages() {
	for {
		select {
		case <-ctx.Done():
			logger.LogInfo("Shutting down worker...", "gracefulShutdown", nil)
			return
		case msg, ok := <-channels.SenderChannel:
			if !ok {
				return
			}
			if limiterErr := limiter.Wait(ctx); limiterErr != nil {
				logger.LogError(fmt.Sprintf("rate limiter error: %s", limiterErr), "workerRateLimitError", nil)
				continue
			}
			tryToSendWithRetry(msg)
		}
	}
}

func EnqueueMessage(text string, message *telego.Message) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	attempts := 0
	for {
		select {
		case <-ctx.Done():
			log.Printf("Context canceled after %d attempts", attempts)
			return

		case channels.SenderChannel <- hndUtils.GetMsgSendParams(text, message):
			if attempts > 0 {
				log.Printf("Message sent after %d retries", attempts)
			}
			return

		case <-ticker.C:
			attempts++
			if attempts%10 == 0 {
				log.Printf("Still retrying, attempt %d", attempts)
			}
		}
	}
}

func tryToSendWithRetry(msg *telego.SendMessageParams) bool {
	var (
		err    error
		apiErr *telegoapi.Error
	)

	msg = msg.WithLinkPreviewOptions(&telego.LinkPreviewOptions{IsDisabled: true}).
		WithParseMode(telego.ModeMarkdownV2).
		WithText(hndUtils.EscapeMarkdownV2Smart(msg.Text))

	for i := 0; i < 3; i++ {
		_, err = botInstance.SendMessage(ctx, msg)
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

func gracefulShutdownQueue() {
	wg.Wait()

	for {
		select {
		case <-channels.SenderChannel:
		default:
			return
		}
	}
}

func OpenQueue(appCtx context.Context, bot *telego.Bot) {
	ctx = appCtx
	botInstance = bot

	rps, err := strconv.Atoi(os.Getenv("RPS_LIMIT"))
	if err != nil {
		logger.LogFatal(fmt.Sprintf("Failed to init queue. Cannot find RPS_LIMIT. %s", err.Error()), "configuring", nil)
	}
	limiter = rate.NewLimiter(rate.Every(time.Second/time.Duration(rps)), 1)

	defer gracefulShutdownQueue()
	wg.Go(consumeMessages)
	<-ctx.Done()
}
