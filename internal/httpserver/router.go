package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"stockrush-go/internal/config"
	"stockrush-go/internal/domain"
	metricspkg "stockrush-go/internal/metrics"
	"stockrush-go/internal/ratelimit"
	"stockrush-go/internal/store"
)

type requestIDKey struct{}

type Server struct {
	db       Pinger
	store    *store.Store
	cfg      config.Config
	limiter  *ratelimit.Limiter
	metrics  *metricspkg.Metrics
	registry *prometheus.Registry
}

func New(cfg config.Config, db Pinger, dataStore *store.Store) http.Handler {
	registry := prometheus.NewRegistry()
	s := &Server{db: db, store: dataStore, cfg: cfg, limiter: ratelimit.New(cfg.RateLimitRequests, cfg.RateLimitBurst), metrics: metricspkg.New(registry), registry: registry}
	r := chi.NewRouter()
	r.Use(middleware.Recoverer, s.requestID, s.securityHeaders, s.cors, s.requestLog)
	r.Get("/health/live", s.live)
	r.Get("/health/ready", s.ready)
	r.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	r.Get("/openapi.yaml", s.openapi)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(s.rateLimit)
		r.Get("/products", s.listProducts)
		r.Post("/products", s.createProduct)
		r.Get("/products/{id}", s.getProduct)
		r.Patch("/products/{id}", s.updateProduct)
		r.Get("/sales", s.listSales)
		r.Post("/sales", s.createSale)
		r.Get("/sales/{id}", s.getSale)
		r.Post("/sales/{id}/activate", s.activateSale)
		r.Post("/sales/{id}/end", s.endSale)
		r.Post("/sales/{id}/reservations", s.createReservation)
		r.Get("/reservations/{id}", s.getReservation)
		r.Post("/reservations/{id}/confirm", s.confirmReservation)
		r.Post("/reservations/{id}/cancel", s.cancelReservation)
		r.Get("/orders", s.listOrders)
		r.Get("/orders/{id}", s.getOrder)
		r.Post("/payments/simulate", s.simulatePayment)
		r.Get("/demo/status", s.demoStatus)
		r.Post("/demo/reset", s.demoReset)
		r.Post("/demo/load-test", s.demoLoadTest)
		r.Get("/demo/report", s.demoReport)
	})
	return r
}

func (s *Server) live(w http.ResponseWriter, r *http.Request) {
	s.success(w, r, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Ping(r.Context()); err != nil {
		s.failure(w, r, &domain.Error{Code: "DATABASE_UNAVAILABLE", Message: "PostgreSQL is unavailable", HTTPStatus: http.StatusServiceUnavailable})
		return
	}
	s.success(w, r, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) listProducts(w http.ResponseWriter, r *http.Request) {
	products, err := s.store.ListProducts(r.Context())
	s.respond(w, r, http.StatusOK, products, err)
}

func (s *Server) createProduct(w http.ResponseWriter, r *http.Request) {
	var in store.CreateProductInput
	if err := decodeJSON(w, r, &in); err != nil {
		s.failure(w, r, err)
		return
	}
	product, err := s.store.CreateProduct(r.Context(), in)
	s.respond(w, r, http.StatusCreated, product, err)
}

func (s *Server) getProduct(w http.ResponseWriter, r *http.Request) {
	product, err := s.store.GetProduct(r.Context(), chi.URLParam(r, "id"))
	s.respond(w, r, http.StatusOK, product, err)
}

func (s *Server) updateProduct(w http.ResponseWriter, r *http.Request) {
	var in store.UpdateProductInput
	if err := decodeJSON(w, r, &in); err != nil {
		s.failure(w, r, err)
		return
	}
	product, err := s.store.UpdateProduct(r.Context(), chi.URLParam(r, "id"), in)
	s.respond(w, r, http.StatusOK, product, err)
}

func (s *Server) listSales(w http.ResponseWriter, r *http.Request) {
	sales, err := s.store.ListSales(r.Context())
	s.respond(w, r, http.StatusOK, sales, err)
}

func (s *Server) createSale(w http.ResponseWriter, r *http.Request) {
	var in store.CreateSaleInput
	if err := decodeJSON(w, r, &in); err != nil {
		s.failure(w, r, err)
		return
	}
	sale, err := s.store.CreateSale(r.Context(), in)
	s.respond(w, r, http.StatusCreated, sale, err)
}

func (s *Server) getSale(w http.ResponseWriter, r *http.Request) {
	sale, err := s.store.GetSale(r.Context(), chi.URLParam(r, "id"))
	s.respond(w, r, http.StatusOK, sale, err)
}

func (s *Server) activateSale(w http.ResponseWriter, r *http.Request) {
	sale, err := s.store.SetSaleState(r.Context(), chi.URLParam(r, "id"), domain.SaleActive)
	s.respond(w, r, http.StatusOK, sale, err)
}

func (s *Server) endSale(w http.ResponseWriter, r *http.Request) {
	sale, err := s.store.SetSaleState(r.Context(), chi.URLParam(r, "id"), domain.SaleEnded)
	s.respond(w, r, http.StatusOK, sale, err)
}

func (s *Server) createReservation(w http.ResponseWriter, r *http.Request) {
	var in store.ReservationRequest
	if err := decodeJSON(w, r, &in); err != nil {
		s.failure(w, r, err)
		return
	}
	in.IdempotencyKey = r.Header.Get("Idempotency-Key")
	result, err := s.store.Reserve(r.Context(), chi.URLParam(r, "id"), in)
	s.metrics.ReservationAttempts.Inc()
	if err == nil && !result.Duplicate {
		s.metrics.ReservationSuccesses.Inc()
	}
	if result.Duplicate {
		s.metrics.Duplicates.Inc()
	}
	if domain.IsCode(err, "INVENTORY_SOLD_OUT") {
		s.metrics.SoldOut.Inc()
	}
	status := http.StatusCreated
	if result.Duplicate {
		status = http.StatusOK
	}
	s.respond(w, r, status, result, err)
}

func (s *Server) getReservation(w http.ResponseWriter, r *http.Request) {
	reservation, err := s.store.GetReservation(r.Context(), chi.URLParam(r, "id"))
	s.respond(w, r, http.StatusOK, reservation, err)
}

func (s *Server) confirmReservation(w http.ResponseWriter, r *http.Request) {
	reservation, err := s.store.TransitionReservation(r.Context(), chi.URLParam(r, "id"), domain.ReservationConfirmed)
	s.respond(w, r, http.StatusOK, reservation, err)
}

func (s *Server) cancelReservation(w http.ResponseWriter, r *http.Request) {
	reservation, err := s.store.TransitionReservation(r.Context(), chi.URLParam(r, "id"), domain.ReservationCancelled)
	s.respond(w, r, http.StatusOK, reservation, err)
}

func (s *Server) listOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := s.store.ListOrders(r.Context())
	s.respond(w, r, http.StatusOK, orders, err)
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	order, err := s.store.GetOrder(r.Context(), chi.URLParam(r, "id"))
	s.respond(w, r, http.StatusOK, order, err)
}

func (s *Server) simulatePayment(w http.ResponseWriter, r *http.Request) {
	var in store.PaymentRequest
	if err := decodeJSON(w, r, &in); err != nil {
		s.failure(w, r, err)
		return
	}
	in.IdempotencyKey = r.Header.Get("Idempotency-Key")
	result, err := s.store.SimulatePayment(r.Context(), in)
	status := http.StatusCreated
	if result.Duplicate {
		status = http.StatusOK
	}
	s.respond(w, r, status, result, err)
}

func (s *Server) respond(w http.ResponseWriter, r *http.Request, status int, data any, err error) {
	if err != nil {
		s.failure(w, r, err)
		return
	}
	s.success(w, r, status, data)
}

func (s *Server) success(w http.ResponseWriter, r *http.Request, status int, data any) {
	writeJSON(w, status, map[string]any{"data": data, "meta": map[string]string{"requestId": requestID(r)}})
}

func (s *Server) failure(w http.ResponseWriter, r *http.Request, err error) {
	domainErr := &domain.Error{Code: "INTERNAL_ERROR", Message: "An internal error occurred", HTTPStatus: http.StatusInternalServerError}
	if !errors.As(err, &domainErr) {
		slog.Error("request failed", "request_id", requestID(r), "error", err)
	}
	writeJSON(w, domainErr.HTTPStatus, map[string]any{"error": map[string]string{"code": domainErr.Code, "message": domainErr.Message, "requestId": requestID(r)}})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return &domain.Error{Code: "INVALID_JSON", Message: "Request body must be valid JSON with known fields", HTTPStatus: 400}
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return &domain.Error{Code: "INVALID_JSON", Message: "Request body must contain one JSON value", HTTPStatus: 400}
	}
	return nil
}

func (s *Server) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if id == "" || len(id) > 128 {
			generated, err := domain.NewID()
			if err != nil {
				http.Error(w, "request ID unavailable", http.StatusInternalServerError)
				return
			}
			id = generated
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

func requestID(r *http.Request) string {
	id, _ := r.Context().Value(requestIDKey{}).(string)
	return id
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) cors(next http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(s.cfg.CORSAllowedOrigins))
	for _, origin := range s.cfg.CORSAllowedOrigins {
		allowed[origin] = struct{}{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := allowed[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key, X-Demo-Token, X-Request-ID, X-User-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (s *Server) requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(wrapped, r)
		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}
		s.metrics.RecordHTTP(r.Method, route, wrapped.status, time.Since(start))
		slog.Info("HTTP request", "request_id", requestID(r), "method", r.Method, "path", r.URL.Path, "status", wrapped.status, "duration_ms", time.Since(start).Milliseconds())
	})
}

func (s *Server) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		client := strings.TrimSpace(r.Header.Get("X-User-ID"))
		if client == "" {
			client, _, _ = net.SplitHostPort(r.RemoteAddr)
		}
		if client == "" {
			client = "unknown"
		}
		if !s.limiter.Allow(client) {
			s.metrics.RateLimited.Inc()
			s.failure(w, r, &domain.Error{Code: "RATE_LIMITED", Message: "Too many requests", HTTPStatus: http.StatusTooManyRequests})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) demoAuthorized(r *http.Request) bool {
	return s.cfg.AppEnv == "development" && r.Header.Get("X-Demo-Token") == s.cfg.DemoToken
}

func validationError(message string) error {
	return &domain.Error{Code: "VALIDATION_ERROR", Message: fmt.Sprintf("Invalid request: %s", message), HTTPStatus: 400}
}
