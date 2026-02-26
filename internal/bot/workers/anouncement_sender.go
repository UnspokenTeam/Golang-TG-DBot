package workers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/channels"
	"github.com/unspokenteam/golang-tg-dbot/internal/bot/service_wrapper"
	"github.com/unspokenteam/golang-tg-dbot/internal/db/querier"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/time/rate"
)

type BatchWorker struct {
	ctx                 context.Context
	services            *service_wrapper.Services
	internalRateLimiter *rate.Limiter
	wg                  sync.WaitGroup
}

func (w *BatchWorker) start(ctx context.Context) {
	go w.listen(ctx)

	<-ctx.Done()
	w.wg.Wait()
	slog.InfoContext(ctx, "Gracefully stopped broadcasts")
}

func (w *BatchWorker) listen(ctx context.Context) {
	slog.InfoContext(ctx, "Started broadcast worker")
	for {
		select {
		case <-ctx.Done():
			slog.DebugContext(ctx, "Broadcaster listener stopped")
			return
		case msg, ok := <-channels.NotifyBroadCasterChannel:
			if !ok {
				slog.DebugContext(ctx, "Broadcaster channel closed")
				return
			}
			w.runBatchTask(msg)
		}
	}
}

func (w *BatchWorker) runBatchTask(base channels.BroadcastBase) {
	w.wg.Go(func() { w.sendNewBatch(base) })
}

func (w *BatchWorker) sendNewBatch(base channels.BroadcastBase) {
	totalSuccessfulChats := 0
	var cursorId int64 = 0
	var batchIterationSize int32 = 100
	batchChan := make(chan channels.FeedbackResult)
	deadChats := make([]int64, 0)
	tracer := otel.Tracer("telegram-bot")

	batchCtx, batchSpan := tracer.Start(w.ctx, fmt.Sprintf("batch_send_%s", base.BatchId))
	defer batchSpan.End()
	batchSpan.SetAttributes(attribute.String("batch_id", base.BatchId))

	slog.InfoContext(batchCtx, fmt.Sprintf("Preparing batch # %s\n\n%s", base.BatchId, base.Text),
		"batch_id", base.BatchId)

	totalChats, _ := w.services.PostgresClient.Queries.GetChatCountForBroadcast(batchCtx)
	slog.InfoContext(batchCtx, fmt.Sprintf("Starting batch send # %s to %d chats...", base.BatchId, totalChats),
		"batch_id", base.BatchId)
	start := time.Now()

	for {
		aliveChats := make([]int64, 0)
		chats, err := w.services.PostgresClient.Queries.GetChatsForBroadcastCursorBased(batchCtx,
			querier.GetChatsForBroadcastCursorBasedParams{
				CursorID: cursorId,
				PageSize: batchIterationSize,
			})
		if errors.Is(err, pgx.ErrNoRows) || len(chats) == 0 {
			break
		}
		if err != nil {
			slog.ErrorContext(batchCtx,
				fmt.Sprintf("batch send failed after %d/%d messages", totalSuccessfulChats, totalChats),
				"batch_id", base.BatchId)
			return
		}
		cursorId = chats[len(chats)-1].ID

		for _, chat := range chats {
			select {
			case <-w.ctx.Done():
				slog.DebugContext(batchCtx,
					fmt.Sprintf("batch send cancelled after %d/%d messages", totalSuccessfulChats, totalChats),
					"batch_id", base.BatchId)
				return
			default:
				// Continue
			}

			if limiterErr := w.internalRateLimiter.Wait(w.ctx); limiterErr != nil {
				slog.ErrorContext(batchCtx, fmt.Sprintf("rate limiter interrupted: %s", limiterErr),
					"batch_id", base.BatchId)
				continue
			}

			EnqueueBroadcast(channels.BroadcastTask{
				BatchCtx:              batchCtx,
				ChatId:                chat.TgID,
				Text:                  base.Text,
				BatchId:               base.BatchId,
				SenderFeedbackChannel: batchChan,
			})
		}

		for range chats {
			result := <-batchChan

			if result.Success {
				totalSuccessfulChats++
				aliveChats = append(aliveChats, result.ChatId)
			} else if result.Err == nil {
				deadChats = append(deadChats, result.ChatId)
			} else {
				slog.ErrorContext(batchCtx, fmt.Sprintf("Send to chat %d failed. Forbidden 403.\n%s",
					result.ChatId, result.Err),
					"batch_id", base.BatchId, "chat_id", result.ChatId)
			}
			if len(result.Message) != 0 {
				slog.InfoContext(batchCtx, fmt.Sprintf("%d: %s", result.ChatId, result.Message),
					"batch_id", base.BatchId)
			}
		}

		err = w.services.PostgresClient.Queries.UpdateLastSysTimestamp(batchCtx, aliveChats)
		if err != nil {
			slog.DebugContext(batchCtx,
				fmt.Sprintf("batch send cancelled after %d/%d messages", totalSuccessfulChats, totalChats),
				"batch_id", base.BatchId)
			return
		}
		slog.InfoContext(batchCtx,
			fmt.Sprintf("Batch iteration send %d/%d chat(s) successfully", len(aliveChats),
				min(int(batchIterationSize), int(totalChats))),
			"batch_id", base.BatchId)

		if len(deadChats) >= 1000 {
			updateChatsErr := w.services.PostgresClient.Queries.UpdateChatStatusToDead(batchCtx, deadChats)
			if updateChatsErr != nil {
				slog.DebugContext(batchCtx,
					fmt.Sprintf("batch send cancelled after %d/%d messages", totalSuccessfulChats, totalChats),
					"batch_id", base.BatchId)
				return
			}
			deadChats = make([]int64, 0)
		}
	}

	if len(deadChats) > 0 {
		updateChatsErr := w.services.PostgresClient.Queries.UpdateChatStatusToDead(batchCtx, deadChats)
		if updateChatsErr != nil {
			slog.DebugContext(batchCtx,
				fmt.Sprintf("batch send cancelled after %d/%d messages", totalSuccessfulChats, totalChats),
				"batch_id", base.BatchId)
			return
		}
	}

	finalTime := time.Since(start)

	slog.InfoContext(batchCtx,
		fmt.Sprintf("Successfully complete batch send # %s.\nSended to %d/%d chats.\n\nDone in %d h %d min %d secs",
			base.BatchId, totalSuccessfulChats, totalChats, int(finalTime.Hours()),
			int(finalTime.Minutes())%60, int(finalTime.Seconds())%60), "done", true, "batch_id", base.BatchId)
}

func AddBatchTask(msg string) string {
	batchId := uuid.NewString()
	channels.NotifyBroadCasterChannel <- channels.BroadcastBase{Text: msg, BatchId: batchId}
	return batchId
}

func InitBroadcastWorker(ctx context.Context, services *service_wrapper.Services) {

	worker := BatchWorker{
		ctx:                 ctx,
		services:            services,
		internalRateLimiter: rate.NewLimiter(rate.Every(time.Second/time.Duration(5)), 5),
		wg:                  sync.WaitGroup{},
	}
	worker.start(ctx)
}
