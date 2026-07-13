//go:build integration

package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"stockrush-go/internal/config"
	"stockrush-go/internal/database"
	"stockrush-go/internal/store"
)

func TestReservationEndpointReplaysOriginalResource(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://stockrush:stockrush@localhost:5432/stockrush?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Skipf("PostgreSQL unavailable: %v", err)
	}
	defer pool.Close()
	if err := database.Migrate(ctx, pool, "../../db/migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	dataStore := store.New(pool, 2*time.Minute)
	status, err := dataStore.ResetDemo(ctx, 10)
	if err != nil {
		t.Fatalf("ResetDemo: %v", err)
	}
	handler := New(config.Config{RateLimitRequests: 1000, RateLimitBurst: 1000, CORSAllowedOrigins: []string{"http://localhost:5173"}}, pool, dataStore)

	var original string
	for attempt, wantStatus := range []int{http.StatusCreated, http.StatusOK} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sales/"+status.Sale.ID+"/reservations", strings.NewReader(`{"userId":"api-buyer","quantity":1}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "api-retry")
		req.Header.Set("X-User-ID", "api-buyer")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != wantStatus {
			t.Fatalf("attempt %d status=%d body=%s", attempt, res.Code, res.Body.String())
		}
		var body struct {
			Data struct {
				Reservation struct {
					ID string `json:"id"`
				} `json:"reservation"`
			} `json:"data"`
		}
		if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if attempt == 0 {
			original = body.Data.Reservation.ID
		} else if body.Data.Reservation.ID != original {
			t.Fatalf("retry reservation=%s want=%s", body.Data.Reservation.ID, original)
		}
	}
}
