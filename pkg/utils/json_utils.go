package utils

import (
	"context"
	"encoding/json"
	"log/slog"
)

func MarshalJsonIgnoreError(ctx context.Context, v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		slog.ErrorContext(ctx, err.Error())
		return ""
	}
	return string(data)
}
