package workers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/channels"
	hndUtils "github.com/unspokenteam/golang-tg-dbot/pkg/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/time/rate"
)

type CommandOutputSenderQueue struct {
	limiter     *rate.Limiter
	botInstance *telego.Bot
	ctx         context.Context
}

func EnqueueMessage(updCtx context.Context, text string, message *telego.Message) {
	ctx, cancel := context.WithTimeout(updCtx, 60*time.Second)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	attempts := 0
	for {
		select {
		case <-updCtx.Done():
			slog.DebugContext(updCtx, "Queue closed, message wasn't sent.", "message", text)
			return

		case <-ctx.Done():
			slog.ErrorContext(updCtx, "Failed to enqueue message: timeout")
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

func (q *CommandOutputSenderQueue) ListenIncomingMessages() {
	for {
		select {
		case <-q.ctx.Done():
			slog.InfoContext(q.ctx, "Closing queue...")
			return
		case msg, ok := <-channels.SenderChannel:
			if !ok {
				return
			}
			if limiterErr := q.limiter.Wait(q.ctx); limiterErr != nil {
				slog.ErrorContext(q.ctx, fmt.Sprintf("Sender rate limiter error: %s", limiterErr))
				continue
			}
			if !q.tryToSendWithRetry(msg) {
				slog.ErrorContext(msg.UpdCtx, "Failed to send message after 3 retries", "send_params", msg.Msg)
			}
		}
	}
}

func (q *CommandOutputSenderQueue) tryToSendWithRetry(msg channels.Message) bool {
	var apiErr *ta.Error

	sendParams := msg.Msg.WithLinkPreviewOptions(&telego.LinkPreviewOptions{IsDisabled: true}).
		WithParseMode(telego.ModeMarkdownV2).
		WithText(hndUtils.EscapeMarkdownV2Smart(msg.Msg.Text))

	for i := 0; i < 3; i++ {
		_, err := q.botInstance.SendMessage(q.ctx, sendParams)
		if err == nil {
			slog.InfoContext(msg.UpdCtx, "Command has been handled successfully")
			return true
		} else if errors.As(err, &apiErr) && apiErr.ErrorCode == 429 {
			time.Sleep(time.Second * time.Duration(apiErr.Parameters.RetryAfter+1))
		} else {
			slog.ErrorContext(msg.UpdCtx, fmt.Sprintf("Failed to send message. %v", err))
			if errors.As(err, &apiErr) && apiErr.ErrorCode == 403 {
				break
			}
		}
	}
	return false
}

type BroadcastSenderQueue struct {
	limiter         *rate.Limiter
	botInstance     *telego.Bot
	ctx             context.Context
	successfulChats metric.Int64Counter
}

func EnqueueBroadcast(task channels.BroadcastTask) {
	ctx, cancel := context.WithTimeout(task.BatchCtx, 60*time.Second)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	attempts := 0
	for {
		select {
		case <-task.BatchCtx.Done():
			slog.DebugContext(task.BatchCtx, "Broadcast closed, broadcast message wasn't sent.", "message", task.Text)
			return

		case <-ctx.Done():
			slog.ErrorContext(task.BatchCtx, "Failed to enqueue broadcast: timeout")
			return

		case channels.BroadcastChannel <- task:
			attempts++
			return

		case <-ticker.C:
			attempts++
			if attempts%10 == 0 {
				slog.DebugContext(task.BatchCtx,
					fmt.Sprintf("Trying to send broadcast message to queue, attempt %d", attempts),
					"chat_id", task.ChatId)
			}
		}
	}
}

func (q *BroadcastSenderQueue) ListenBroadcastTasks() {
	for {
		select {
		case <-q.ctx.Done():
			slog.InfoContext(q.ctx, "Closing broadcast queue...")
			return
		case task, ok := <-channels.BroadcastChannel:
			if !ok {
				return
			}
			if limiterErr := q.limiter.Wait(q.ctx); limiterErr != nil {
				slog.ErrorContext(q.ctx, fmt.Sprintf("Broadcast rate limiter error: %s", limiterErr))
				continue
			}
			q.tryToBroadcastWithRetry(task)
		}
	}
}

func (q *BroadcastSenderQueue) tryToBroadcastWithRetry(task channels.BroadcastTask) {
	var apiErr *ta.Error

	msg := tu.Message(tu.ID(task.ChatId), task.Text)
	sendParams := msg.WithLinkPreviewOptions(&telego.LinkPreviewOptions{IsDisabled: true}).
		WithParseMode(telego.ModeMarkdownV2).
		WithText(hndUtils.EscapeMarkdownV2Smart(msg.Text))

	for i := 0; i < 3; i++ {
		_, err := q.botInstance.SendMessage(q.ctx, sendParams)
		if err == nil {
			go q.successfulChats.Add(q.ctx, 1, metric.WithAttributes(
				attribute.String("batch_id", task.BatchId),
				attribute.Int64("chat_id", task.ChatId),
			))
			task.SenderFeedbackChannel <- channels.FeedbackResult{
				Success:     true,
				ChatId:      task.ChatId,
				Message:     "",
				Err:         nil,
				AttemptedAt: time.Now(),
			}
			return
		} else if errors.As(err, &apiErr) && apiErr.ErrorCode == 429 {
			time.Sleep(time.Second * time.Duration(apiErr.Parameters.RetryAfter+1))
		} else if errors.As(err, &apiErr) &&
			strings.Contains(strings.ToLower(apiErr.Description), "not enough rights") {
			task.SenderFeedbackChannel <- channels.FeedbackResult{
				Success:     false,
				ChatId:      task.ChatId,
				Message:     "",
				Err:         apiErr,
				AttemptedAt: time.Now(),
			}
			return
		}

		slog.ErrorContext(task.BatchCtx, fmt.Sprintf("Failed to send message. %v", err))
		if errors.As(err, &apiErr) && apiErr.ErrorCode == 403 {
			break
		}
	}
	task.SenderFeedbackChannel <- channels.FeedbackResult{
		Success:     false,
		ChatId:      task.ChatId,
		Message:     "",
		Err:         nil,
		AttemptedAt: time.Now(),
	}
}

func gracefulShutdownQueues(ctx context.Context, wg *sync.WaitGroup) {
	wg.Wait()

	for {
		select {
		case <-channels.SenderChannel:
		case <-channels.BroadcastChannel:
		default:
			slog.InfoContext(ctx, "Queue has been shut down...")
			return
		}
	}
}

func InitQueues(appCtx context.Context, bot *telego.Bot, rateLimiter *rate.Limiter, meter metric.Meter) {
	var queueWg sync.WaitGroup

	successfulChatsMetric, err := meter.Int64Counter(
		fmt.Sprintf("bot.broadcast_messages.total"),
		metric.WithDescription("Total number of messages sent via broadcast"),
		metric.WithUnit("1"),
	)
	if err != nil {
		slog.ErrorContext(appCtx, fmt.Sprintf("failed to initiate broadcast worker: %s", err))
	}

	qCommands := CommandOutputSenderQueue{
		limiter:     rateLimiter,
		botInstance: bot,
		ctx:         appCtx,
	}

	qBroadcast := BroadcastSenderQueue{
		limiter:         rateLimiter,
		botInstance:     bot,
		ctx:             appCtx,
		successfulChats: successfulChatsMetric,
	}

	defer gracefulShutdownQueues(appCtx, &queueWg)
	queueWg.Go(qCommands.ListenIncomingMessages)
	queueWg.Go(qBroadcast.ListenBroadcastTasks)

	slog.InfoContext(appCtx, "Created queue successfully...")
	<-appCtx.Done()
}
