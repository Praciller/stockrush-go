package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	HTTPRequests         *prometheus.CounterVec
	HTTPDuration         *prometheus.HistogramVec
	ReservationAttempts  prometheus.Counter
	ReservationSuccesses prometheus.Counter
	SoldOut              prometheus.Counter
	Duplicates           prometheus.Counter
	RateLimited          prometheus.Counter
	Expired              prometheus.Counter
	InventoryRestored    prometheus.Counter
	WorkerErrors         prometheus.Counter
}

func New(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		HTTPRequests:         prometheus.NewCounterVec(prometheus.CounterOpts{Name: "stockrush_http_requests_total", Help: "HTTP requests by method, route, and status."}, []string{"method", "route", "status"}),
		HTTPDuration:         prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "stockrush_http_request_duration_seconds", Help: "HTTP request duration by method and route."}, []string{"method", "route"}),
		ReservationAttempts:  prometheus.NewCounter(prometheus.CounterOpts{Name: "stockrush_reservation_attempts_total", Help: "Reservation attempts."}),
		ReservationSuccesses: prometheus.NewCounter(prometheus.CounterOpts{Name: "stockrush_reservation_successes_total", Help: "Successful reservations."}),
		SoldOut:              prometheus.NewCounter(prometheus.CounterOpts{Name: "stockrush_sold_out_responses_total", Help: "Sold-out responses."}),
		Duplicates:           prometheus.NewCounter(prometheus.CounterOpts{Name: "stockrush_duplicate_requests_total", Help: "Idempotent duplicate requests."}),
		RateLimited:          prometheus.NewCounter(prometheus.CounterOpts{Name: "stockrush_rate_limited_requests_total", Help: "Rate-limited requests."}),
		Expired:              prometheus.NewCounter(prometheus.CounterOpts{Name: "stockrush_expired_reservations_total", Help: "Expired reservations."}),
		InventoryRestored:    prometheus.NewCounter(prometheus.CounterOpts{Name: "stockrush_inventory_restorations_total", Help: "Inventory units restored by expiration."}),
		WorkerErrors:         prometheus.NewCounter(prometheus.CounterOpts{Name: "stockrush_worker_processing_errors_total", Help: "Expiration worker processing errors."}),
	}
	reg.MustRegister(m.HTTPRequests, m.HTTPDuration, m.ReservationAttempts, m.ReservationSuccesses, m.SoldOut, m.Duplicates, m.RateLimited, m.Expired, m.InventoryRestored, m.WorkerErrors)
	return m
}

func (m *Metrics) RecordHTTP(method, route string, status int, duration time.Duration) {
	m.HTTPRequests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
	m.HTTPDuration.WithLabelValues(method, route).Observe(duration.Seconds())
}
