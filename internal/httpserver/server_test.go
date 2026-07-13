package httpserver

import (
	"context"
	"crypto/sha256"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"stockrush-go/internal/config"
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

func TestProductionPrivilegedMutationsRequireAdminKey(t *testing.T) {
	key := "correct-horse-battery-staple-public-demo-admin-key"
	hash := sha256.Sum256([]byte(key))
	handler := New(config.Config{
		AppEnv: "production", RateLimitRequests: 100, RateLimitBurst: 100,
		CORSAllowedOrigins: []string{"https://praciller.github.io"}, AdminAPIKeyHash: hash[:],
	}, stubPinger{}, nil)

	request := func(token, contentType string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/products", strings.NewReader(`{}`))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		return res
	}

	if res := request("", "application/json"); res.Code != http.StatusNotFound {
		t.Fatalf("unauthenticated status = %d, want 404", res.Code)
	}
	if res := request("wrong", "application/json"); res.Code != http.StatusNotFound {
		t.Fatalf("invalid key status = %d, want 404", res.Code)
	}
	if res := request(key, ""); res.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("missing content type status = %d, want 415", res.Code)
	}
}

func TestVersionContainsOnlySafeBuildMetadata(t *testing.T) {
	handler := New(config.Config{RateLimitRequests: 10, RateLimitBurst: 10, CORSAllowedOrigins: []string{"http://localhost:5173"}}, stubPinger{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"version"`) || strings.Contains(res.Body.String(), "DATABASE_URL") {
		t.Fatalf("GET /version status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestForwardedClientIPRequiresTrustedProxy(t *testing.T) {
	trusted := netip.MustParsePrefix("10.0.0.0/8")
	server := &Server{cfg: config.Config{TrustedProxyCIDRs: []netip.Prefix{trusted}}}

	trustedRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	trustedRequest.RemoteAddr = "10.1.2.3:1234"
	trustedRequest.Header.Set("X-Forwarded-For", "203.0.113.9, 10.1.2.3")
	if got := server.clientIP(trustedRequest); got != "203.0.113.9" {
		t.Fatalf("trusted proxy client IP = %q", got)
	}

	untrustedRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	untrustedRequest.RemoteAddr = "198.51.100.7:1234"
	untrustedRequest.Header.Set("X-Forwarded-For", "203.0.113.9")
	if got := server.clientIP(untrustedRequest); got != "198.51.100.7" {
		t.Fatalf("untrusted proxy client IP = %q", got)
	}
}

func TestCORSRejectsDisallowedOrigin(t *testing.T) {
	handler := New(config.Config{RateLimitRequests: 10, RateLimitBurst: 10, CORSAllowedOrigins: []string{"https://praciller.github.io"}}, stubPinger{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	req.Header.Set("Origin", "https://attacker.example")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("disallowed origin status = %d, want 403", res.Code)
	}
}
