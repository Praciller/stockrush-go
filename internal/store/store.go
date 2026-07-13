package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"stockrush-go/internal/domain"
)

var ErrNotFound = &domain.Error{Code: "NOT_FOUND", Message: "The requested resource was not found", HTTPStatus: http.StatusNotFound}

type Store struct {
	pool *pgxpool.Pool
	ttl  time.Duration
	now  func() time.Time
}

func New(pool *pgxpool.Pool, ttl time.Duration) *Store {
	return &Store{pool: pool, ttl: ttl, now: time.Now}
}

func event(ctx context.Context, tx pgx.Tx, aggregateType, aggregateID, eventType, previous, next string, metadata any) error {
	id, err := domain.NewID()
	if err != nil {
		return err
	}
	body, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal event metadata: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO domain_events
		(id, aggregate_type, aggregate_id, event_type, previous_state, new_state, metadata)
		VALUES ($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),$7)`, id, aggregateType, aggregateID, eventType, previous, next, body)
	return err
}

func rollback(tx pgx.Tx) { _ = tx.Rollback(context.Background()) }

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
