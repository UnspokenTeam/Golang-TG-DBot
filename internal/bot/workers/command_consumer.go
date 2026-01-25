package workers

import (
	"context"
	"log"
	"time"

	"github.com/mymmrac/telego/telegohandler"
)

func gracefulShutdownConsumer(handler *telegohandler.BotHandler) {
	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := handler.StopWithContext(shutdownCtx); err != nil {
		log.Printf("Error stopping handler: %v", err)
	}
}

func RunCommandConsumer(ctx context.Context, handler *telegohandler.BotHandler) {
	defer gracefulShutdownConsumer(handler)

	go func() {
		if err := handler.Start(); err != nil {
			log.Printf("Error starting handler: %v", err)
		}
	}()

	<-ctx.Done()
}
