package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kirill-shtrykov/minimon/internal/conf"
	"github.com/kirill-shtrykov/minimon/internal/db/generated"
)

type Repo struct {
	db      *sql.DB
	queries *generated.Queries
}

func (r *Repo) Metric(
	ctx context.Context,
	key string,
	minDate time.Time,
	maxDate time.Time,
) ([]generated.Metric, error) {
	m, err := r.queries.Metric(ctx,
		generated.MetricParams{Key: key + "%", MinDate: minDate, MaxDate: maxDate})
	if err != nil {
		return nil, fmt.Errorf("failed to get metric: %w", err)
	}

	return m, nil
}

func (r *Repo) MetricsByDate(
	ctx context.Context,
	minDate time.Time,
	maxDate time.Time,
) ([]generated.Metric, error) {
	m, err := r.queries.MetricsByDate(ctx,
		generated.MetricsByDateParams{MinDate: minDate, MaxDate: maxDate})
	if err != nil {
		return nil, fmt.Errorf("failed to get metric: %w", err)
	}

	return m, nil
}

func (r *Repo) AddValue(
	ctx context.Context,
	arg generated.AddValueParams,
) error {
	_, err := r.queries.AddValue(ctx, arg)
	if err != nil {
		return fmt.Errorf("failed to add value: %w", err)
	}

	return nil
}

func New(cfg conf.SQLiteConfig) (*Repo, error) {
	db, err := sql.Open("sqlite3", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	queries := generated.New(db)

	return &Repo{db: db, queries: queries}, nil
}
