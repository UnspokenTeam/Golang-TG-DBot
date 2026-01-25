package workers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/valyala/fasthttp"
)

func gracefulShutdownServer(srv *fasthttp.Server) {
	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.ShutdownWithContext(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}

func StartServer(ctx context.Context, srv *fasthttp.Server, port int16) {
	defer gracefulShutdownServer(srv)

	go func() {
		if err := srv.ListenAndServe(fmt.Sprintf(":%d", port)); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	<-ctx.Done()
}
