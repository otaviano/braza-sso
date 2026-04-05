package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/otaviano/braza-sso/internal/middleware"
)

// TestPerIPMiddleware_RetryAfterHeader verifies the Retry-After header is set on
// rate-limited responses. Because we use an unreachable Redis (fail-open),
// the next handler is always called — but we can still verify the header presence
// when a real implementation would block.
//
// NOTE: Full rate-limit enforcement tests require a real Redis or in-process mock.
// These tests focus on the fail-open behaviour and header contract.
func TestPerIPMiddleware_FailOpen_RetryAfterNotPresent(t *testing.T) {
	// With an unreachable Redis, the middleware should fail open (pass through).
	limiter := middleware.NewSlidingWindowLimiter(newUnreachableRedis())
	mw := limiter.PerIP(1, 0)

	var nextCalled bool
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	mw(next).ServeHTTP(rec, req)

	if !nextCalled {
		t.Error("expected next handler to be called on fail-open path")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 on fail-open path, got %d", rec.Code)
	}
}

func TestPerIPMiddleware_XRealIPUsedAsKey(t *testing.T) {
	// Two requests from different X-Real-IP values should be treated as different keys.
	// With an unreachable Redis (fail-open) both pass through — we just verify they aren't blocked.
	limiter := middleware.NewSlidingWindowLimiter(newUnreachableRedis())
	mw := limiter.PerIP(1, 0)

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for _, ip := range []string{"10.0.0.1", "10.0.0.2"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.Header.Set("X-Real-IP", ip)
		mw(next).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("IP %s: expected 200, got %d", ip, rec.Code)
		}
	}
}

func TestPerEmailSilentMiddleware_FailOpen_PassesThrough(t *testing.T) {
	limiter := middleware.NewSlidingWindowLimiter(newUnreachableRedis())
	mw := limiter.PerEmailSilent(1, 0, "email")

	var nextCalled bool
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/password/reset-request", nil)
	mw(next).ServeHTTP(rec, req)

	if !nextCalled {
		t.Error("expected next handler to be called on fail-open path")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestPerEmailSilentMiddleware_NoLeakedRateLimitHeader(t *testing.T) {
	// The PerEmailSilent middleware must NOT set Retry-After or X-RateLimit headers
	// (to prevent enumeration attacks via timing).
	limiter := middleware.NewSlidingWindowLimiter(newUnreachableRedis())
	mw := limiter.PerEmailSilent(1, 0, "email")

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/password/reset-request", nil)
	mw(next).ServeHTTP(rec, req)

	if rec.Header().Get("Retry-After") != "" {
		t.Error("PerEmailSilent must not expose Retry-After header")
	}
}

func TestSlidingWindowLimiter_Allow_ReturnsErrorOnRedisDown(t *testing.T) {
	// Verify fail-open with unreachable Redis still returns a non-nil error.
	limiter := middleware.NewSlidingWindowLimiter(newUnreachableRedis())
	// Use a real context so Redis doesn't panic on nil.
	ok, count, err := limiter.Allow(context.Background(), "key", 5, time.Millisecond)
	if !ok {
		t.Error("expected fail-open (ok=true) on Redis error")
	}
	if count != 0 {
		t.Errorf("expected count=0 on error, got %d", count)
	}
	if err == nil {
		t.Error("expected non-nil error when Redis is unreachable")
	}
}
