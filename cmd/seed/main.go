package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"stockrush-go/internal/config"
	"stockrush-go/internal/database"
	"stockrush-go/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fail(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := database.OpenWithOptions(ctx, cfg.DatabaseURL, database.Options{
		MaxConns: cfg.DatabaseMaxConns, MinConns: cfg.DatabaseMinConns, MaxConnLifetime: cfg.DatabaseMaxConnLifetime,
		StatementTimeout: cfg.DatabaseStatementTimeout, TransactionTimeout: cfg.DatabaseTransactionTimeout,
	})
	if err != nil {
		fail(err)
	}
	defer pool.Close()
	status, err := store.New(pool, cfg.ReservationTTL).EnsurePublicDemo(ctx, 100)
	if err != nil {
		fail(err)
	}
	fmt.Printf("public demo ready: sale=%s inventory=%d\n", status.Sale.ID, status.Product.Available)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
