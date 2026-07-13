package ratelimit

import "testing"

func TestLimiterAllowsBurstThenRejects(t *testing.T) {
	limiter := New(1, 2)
	if !limiter.Allow("buyer-1") || !limiter.Allow("buyer-1") {
		t.Fatal("initial burst should be allowed")
	}
	if limiter.Allow("buyer-1") {
		t.Fatal("request beyond burst should be rejected")
	}
	if !limiter.Allow("buyer-2") {
		t.Fatal("one buyer must not consume another buyer's budget")
	}
}
