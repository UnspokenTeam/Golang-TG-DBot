package workers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoapi"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/channels"
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
			slog.InfoContext(ctx, "Closing queue...")
			return
		case msg, ok := <-channels.SenderChannel:
			if !ok {
				return
			}
			if limiterErr := limiter.Wait(ctx); limiterErr != nil {
				slog.ErrorContext(ctx, fmt.Sprintf("Sender rate limiter error: %s", limiterErr))
				continue
			}
			if !tryToSendWithRetry(msg) {
				slog.ErrorContext(msg.UpdCtx, "Failed to send message after 3 retries", "send_params", msg.Msg)
			}
		}
	}
}

func EnqueueBroadcast(updCtx context.Context, text string, chatId int64) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	attempts := 0
	for {
		select {
		case <-ctx.Done():
			slog.DebugContext(updCtx, "Queue closed, broadcast message wasn't sent.", "message", text)
			return

		case channels.SenderChannel <- channels.Message{UpdCtx: updCtx, Msg: tu.Message(tu.ID(chatId), text)}:
			attempts++
			slog.InfoContext(updCtx, fmt.Sprintf("Broadcast message has been sent to queue after %d retries", attempts))
			return

		case <-ticker.C:
			attempts++
			if attempts%10 == 0 {
				slog.DebugContext(updCtx, fmt.Sprintf("Trying to send broadcast message to queue, attempt %d", attempts))
			}
		}
	}
}

func EnqueueMessage(updCtx context.Context, text string, message *telego.Message) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	attempts := 0
	for {
		select {
		case <-ctx.Done():
			slog.DebugContext(updCtx, "Queue closed, message wasn't sent.", "message", text)
			return

		case channels.SenderChannel <- channels.Message{UpdCtx: updCtx, Msg: hndUtils.GetMsgSendParams(text, message)}:
			attempts++
			slog.InfoContext(updCtx, fmt.Sprintf("Message has been sent to queue after %d retries", attempts))
			return

		case <-ticker.C:
			attempts++
			if attempts%10 == 0 {
				slog.DebugContext(updCtx, fmt.Sprintf("Trying to send message to queue, attempt %d", attempts))
			}
		}
	}
}

func tryToSendWithRetry(msg channels.Message) bool {
	var (
		err    error
		apiErr *telegoapi.Error
	)

	sendParams := msg.Msg.WithLinkPreviewOptions(&telego.LinkPreviewOptions{IsDisabled: true}).
		WithParseMode(telego.ModeMarkdownV2).
		WithText(hndUtils.EscapeMarkdownV2Smart(msg.Msg.Text))

	for i := 0; i < 3; i++ {
		_, err = botInstance.SendMessage(ctx, sendParams)
		if err == nil {
			slog.InfoContext(msg.UpdCtx, "Command has been handled successfully")
			return true
		} else if errors.As(err, &apiErr) && apiErr.ErrorCode == 429 {
			time.Sleep(time.Second * time.Duration(apiErr.Parameters.RetryAfter+1))
		} else {
			slog.ErrorContext(msg.UpdCtx, fmt.Sprintf("Failed to send message. %v", err))
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
			slog.InfoContext(ctx, "Queue has been shut down...")
			return
		}
	}
}

func OpenQueue(appCtx context.Context, bot *telego.Bot, rateLimiter *rate.Limiter) {
	ctx = appCtx
	botInstance = bot
	limiter = rateLimiter

	defer gracefulShutdownQueue()
	wg.Go(consumeMessages)
	slog.InfoContext(ctx, "Created queue successfully...")
	<-ctx.Done()
}
