package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/otaviano/braza-sso/internal/middleware"
	"github.com/redis/go-redis/v9"
)

// newTestRedis creates an in-process Redis client pointed at a test server.
// Since we do not spin up a real Redis in unit tests, we verify behaviour using
// a client configured to a non-existent address (connection error = fail-open path).
func newUnreachableRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: "localhost:1", // nothing listening — every call errors immediately
	})
}

func TestSlidingWindowLimiter_Allow(t *testing.T) {
	t.Run("fails open on Redis error", func(t *testing.T) {
		limiter := middleware.NewSlidingWindowLimiter(newUnreachableRedis())
		ok, count, err := limiter.Allow(context.Background(), "test-key", 5, 0)
		if !ok {
			t.Error("expected fail-open (ok=true) on Redis error")
		}
		if count != 0 {
			t.Errorf("expected count=0 on error, got %d", count)
		}
		if err == nil {
			t.Error("expected non-nil error when Redis is unreachable")
		}
	})
}

func TestPerIPMiddleware(t *testing.T) {
	t.Run("fails open on Redis error — request passes through", func(t *testing.T) {
		limiter := middleware.NewSlidingWindowLimiter(newUnreachableRedis())
		mw := limiter.PerIP(1, 0)

		var called bool
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		mw(next).ServeHTTP(rec, req)

		if !called {
			t.Error("expected next handler to be called when Redis is unreachable (fail open)")
		}
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}

func TestPerEmailSilentMiddleware(t *testing.T) {
	t.Run("fails open on Redis error — request passes through", func(t *testing.T) {
		limiter := middleware.NewSlidingWindowLimiter(newUnreachableRedis())
		mw := limiter.PerEmailSilent(1, 0, "email")

		var called bool
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/auth/password/reset-request", nil)
		mw(next).ServeHTTP(rec, req)

		if !called {
			t.Error("expected next handler to be called when Redis is unreachable (fail open)")
		}
	})
}
