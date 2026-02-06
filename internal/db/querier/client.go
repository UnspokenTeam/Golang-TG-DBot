package querier

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/unspokenteam/golang-tg-dbot/internal/configs"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

type DbClient struct {
	connectionPool *pgxpool.Pool
	Queries        *Queries
}

func (dbClient *DbClient) Close(ctx context.Context) {
	dbClient.connectionPool.Close()
	slog.InfoContext(ctx, "Postgres has been shut down successfully.")
}

func (dbClient *DbClient) NewTx(ctx context.Context, txOpts *pgx.TxOptions) (context.Context, *Queries) {
	tx, err := dbClient.connectionPool.BeginTx(ctx, *txOpts)
	if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Error starting new transaction: %v", err))
	}
	spanCtx, _ := tracer.Start(ctx, "start_transaction")

	return spanCtx, &Queries{
		db: &DBTXWithLogging{tx},
	}
}

func (dbClient *DbClient) CommitTx(ctx context.Context, queries *Queries) {
	err := queries.db.(*DBTXWithLogging).db.(pgx.Tx).Commit(ctx)
	if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("Error commiting transaction: %v", err))
	}
}

func (dbClient *DbClient) RollbackTx(ctx context.Context, queries *Queries) {
	err := queries.db.(*DBTXWithLogging).db.(pgx.Tx).Rollback(ctx)
	if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		slog.ErrorContext(ctx, fmt.Sprintf("Error rolling back transaction: %v", err))
	}
	trace.SpanFromContext(ctx).End()
}

func CreateClient(cfg *configs.PostgresConfig, ctx context.Context, t trace.Tracer) (*DbClient, error) {
	tracer = t
	config, err := pgxpool.ParseConfig(cfg.GetConnectionString())
	if err != nil {
		return nil, err
	}

	config.ConnConfig.Tracer = otelpgx.NewTracer(
		otelpgx.WithTrimSQLInSpanName(),
		otelpgx.WithIncludeQueryParameters(),
	)

	connectionPool, poolCreationError := pgxpool.NewWithConfig(ctx, config)
	if poolCreationError != nil {
		return nil, poolCreationError
	}

	return &DbClient{
		connectionPool: connectionPool,
		Queries: New(&DBTXWithLogging{
			db: connectionPool,
		}),
	}, nil
}

type DBTXWithLogging struct {
	db DBTX
}

func (d *DBTXWithLogging) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	ctx, queryRowSpan := tracer.Start(ctx, "query_row")
	defer queryRowSpan.End()
	row := d.db.QueryRow(ctx, sql, args...)

	slog.DebugContext(ctx, "db query",
		slog.String("sql", sql),
	)

	return row
}

func (d *DBTXWithLogging) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	ctx, execSpan := tracer.Start(ctx, "exec")
	defer execSpan.End()
	tag, err := d.db.Exec(ctx, sql, args...)

	slog.DebugContext(ctx, "db exec",
		slog.String("sql", sql),
		slog.Int64("rows", tag.RowsAffected()),
	)

	if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("db exec err: %v", err))
	}

	return tag, err
}

func (d *DBTXWithLogging) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	ctx, querySpan := tracer.Start(ctx, "query")
	defer querySpan.End()
	rows, err := d.db.Query(ctx, sql, args...)

	slog.DebugContext(ctx, "db query",
		slog.String("sql", sql),
	)

	if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("db query err: %v", err))
		return rows, err
	}

	return rows, err
}
