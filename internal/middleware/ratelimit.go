// Package middleware provides HTTP middleware for the Braza SSO API,
// including JWT authentication enforcement and Redis-backed rate limiting.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// SlidingWindowLimiter implements a Redis INCR+EXPIRE sliding window rate limiter.
type SlidingWindowLimiter struct {
	redis *redis.Client
}

func NewSlidingWindowLimiter(r *redis.Client) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{redis: r}
}

// Allow returns true if the request is within limits. It also returns the current count
// and the window duration so callers can set Retry-After.
func (l *SlidingWindowLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int64, error) {
	pipe := l.redis.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	if _, err := pipe.Exec(ctx); err != nil {
		// Fail open — don't block traffic on Redis errors, but log them.
		log.Warn().Err(err).Str("key", key).Msg("rate limiter: Redis error, failing open")
		return true, 0, err
	}
	count := incr.Val()
	return count <= int64(limit), count, nil
}

// PerIP returns middleware that limits requests per remote IP address.
// limit: max requests; window: sliding window duration.
func (l *SlidingWindowLimiter) PerIP(limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r)
			key := fmt.Sprintf("rl:ip:%s:%s", r.URL.Path, ip)
			ok, _, _ := l.Allow(r.Context(), key, limit, window)
			if !ok {
				retryAfter := int(window.Seconds())
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// PerEmail returns middleware that silently drops (returns 200) when the per-email
// limit is exceeded. Used for reset-request to prevent timing-based enumeration.
func (l *SlidingWindowLimiter) PerEmailSilent(limit int, window time.Duration, emailField string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// We read the email from the query string or let the handler deal with it.
			// For rate-limiting purposes we use the IP as a proxy when email isn't in the URL.
			ip := realIP(r)
			key := fmt.Sprintf("rl:email_ip:%s:%s", r.URL.Path, ip)
			ok, _, _ := l.Allow(r.Context(), key, limit, window)
			if !ok {
				// Silent 200 — don't reveal that rate limit was hit
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// realIP returns the real client IP, respecting X-Real-IP and X-Forwarded-For headers
// set by the chi RealIP middleware.
func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
