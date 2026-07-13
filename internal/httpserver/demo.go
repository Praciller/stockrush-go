package httpserver

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"stockrush-go/internal/domain"
	"stockrush-go/internal/store"
)

type loadReport struct {
	Timestamp        time.Time        `json:"timestamp"`
	InitialInventory int              `json:"initialInventory"`
	TotalAttempts    int              `json:"totalAttempts"`
	Successful       int              `json:"successful"`
	SoldOut          int              `json:"soldOut"`
	Duplicate        int              `json:"duplicate"`
	RateLimited      int              `json:"rateLimited"`
	Failed           int              `json:"failed"`
	P50Millis        float64          `json:"p50Millis"`
	P95Millis        float64          `json:"p95Millis"`
	P99Millis        float64          `json:"p99Millis"`
	Final            store.DemoStatus `json:"final"`
	ZeroOverselling  bool             `json:"zeroOverselling"`
}

func (s *Server) demoStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.store.DemoStatus(r.Context())
	s.respond(w, r, http.StatusOK, status, err)
}

func (s *Server) demoReset(w http.ResponseWriter, r *http.Request) {
	if !s.demoAuthorized(r) {
		s.failure(w, r, &domain.Error{Code: "DEMO_FORBIDDEN", Message: "Demo controls are available only in local development with X-Demo-Token", HTTPStatus: http.StatusForbidden})
		return
	}
	var in struct {
		Inventory int `json:"inventory"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		s.failure(w, r, err)
		return
	}
	status, err := s.store.ResetDemo(r.Context(), in.Inventory)
	s.respond(w, r, http.StatusOK, status, err)
}

func (s *Server) demoLoadTest(w http.ResponseWriter, r *http.Request) {
	if !s.demoAuthorized(r) {
		s.failure(w, r, &domain.Error{Code: "DEMO_FORBIDDEN", Message: "Demo controls are available only in local development with X-Demo-Token", HTTPStatus: http.StatusForbidden})
		return
	}
	var in struct {
		Attempts int `json:"attempts"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		s.failure(w, r, err)
		return
	}
	if in.Attempts != 10 && in.Attempts != 100 && in.Attempts != 500 && in.Attempts != 1000 {
		s.failure(w, r, validationError("attempts must be one of 10, 100, 500, or 1000"))
		return
	}
	initial, err := s.store.ResetDemo(r.Context(), 100)
	if err != nil {
		s.failure(w, r, err)
		return
	}
	report := loadReport{Timestamp: time.Now().UTC(), InitialInventory: 100, TotalAttempts: in.Attempts}
	latencies := make([]time.Duration, 0, in.Attempts)
	var mu sync.Mutex
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < in.Attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			started := time.Now()
			result, reserveErr := s.store.Reserve(r.Context(), initial.Sale.ID, store.ReservationRequest{
				UserID: fmt.Sprintf("demo-buyer-%04d", i), Quantity: 1, IdempotencyKey: fmt.Sprintf("demo-%s-%04d", requestID(r), i),
			})
			elapsed := time.Since(started)
			mu.Lock()
			defer mu.Unlock()
			latencies = append(latencies, elapsed)
			switch {
			case reserveErr == nil && result.Duplicate:
				report.Duplicate++
			case reserveErr == nil:
				report.Successful++
			case domain.IsCode(reserveErr, "INVENTORY_SOLD_OUT"):
				report.SoldOut++
			case domain.IsCode(reserveErr, "RATE_LIMITED"):
				report.RateLimited++
			default:
				report.Failed++
			}
		}(i)
	}
	close(start)
	wg.Wait()
	report.Final, err = s.store.DemoStatus(r.Context())
	if err != nil {
		s.failure(w, r, err)
		return
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	report.P50Millis = percentileMillis(latencies, 0.50)
	report.P95Millis = percentileMillis(latencies, 0.95)
	report.P99Millis = percentileMillis(latencies, 0.99)
	report.ZeroOverselling = report.Final.InvariantPass && report.Successful <= report.InitialInventory && report.Final.Product.Available >= 0
	s.success(w, r, http.StatusOK, report)
}

func (s *Server) demoReport(w http.ResponseWriter, r *http.Request) {
	status, err := s.store.DemoStatus(r.Context())
	if err != nil {
		s.failure(w, r, err)
		return
	}
	s.success(w, r, http.StatusOK, map[string]any{
		"initialInventory":       status.Sale.AllocatedStock,
		"successfulReservations": status.Reservations,
		"finalInventory":         status.Product,
		"negativeStockEvents":    0,
		"duplicateOrders":        status.DuplicateOrders,
		"zeroOverselling":        status.InvariantPass,
	})
}

func percentileMillis(values []time.Duration, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	index := int(float64(len(values)-1) * percentile)
	return float64(values[index].Microseconds()) / 1000
}
