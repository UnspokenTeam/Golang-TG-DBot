package main

import (
	"context"
	"github.com/unspokenteam/golang-tg-dbot/internal/db"
)

func main() {
	ctx := context.Background()
	db.RunAutoDumpJob(ctx)
}
