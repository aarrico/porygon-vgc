package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

func Open(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: open pool: %w", err)
	}
	return pool, nil
}

type PoolPinger struct {
	pool *pgxpool.Pool
}

func NewPinger(pool *pgxpool.Pool) PoolPinger {
	return PoolPinger{pool: pool}
}

func (p PoolPinger) Ping(ctx context.Context) error {
	if err := p.pool.Ping(ctx); err != nil {
		return fmt.Errorf("postgres: ping: %w", err)
	}
	return nil
}
