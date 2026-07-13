package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubPinger struct{ err error }

func (s stubPinger) Ping(context.Context) error { return s.err }

func TestHealthEndpointsExposeLivenessAndDatabaseReadiness(t *testing.T) {
	handler := NewHealthHandler(stubPinger{})

	for _, test := range []struct {
		path string
		want int
	}{{"/health/live", http.StatusOK}, {"/health/ready", http.StatusOK}} {
		req := httptest.NewRequest(http.MethodGet, test.path, nil)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != test.want {
			t.Fatalf("GET %s status = %d, want %d", test.path, res.Code, test.want)
		}
	}

	unready := NewHealthHandler(stubPinger{err: errors.New("database unavailable")})
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	res := httptest.NewRecorder()
	unready.ServeHTTP(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /health/ready status = %d, want %d", res.Code, http.StatusServiceUnavailable)
	}
}
