package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/unspokenteam/golang-tg-dbot/internal/configs"
	"github.com/unspokenteam/golang-tg-dbot/internal/db/querier"
)

type Client struct {
	connectionPool *pgxpool.Pool
	Queries        *querier.Queries
}

func (dbClient *Client) Close() {
	dbClient.connectionPool.Close()
}

func CreateConnection(cfg *configs.PostgresConfig, ctx context.Context) (*Client, error) {
	connectionPool, poolCreationError := pgxpool.New(ctx, cfg.GetConnectionString())
	if poolCreationError != nil {
		return nil, poolCreationError
	}

	return &Client{
		connectionPool: connectionPool,
		Queries:        querier.New(connectionPool),
	}, nil
}
