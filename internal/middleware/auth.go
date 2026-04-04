package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/otaviano/braza-sso/internal/auth"
)

// JWTVerifier is the subset of auth.TokenService used by the middleware.
type JWTVerifier interface {
	VerifyAccessToken(tokenStr string) (*auth.Claims, error)
}

// RequireAuth validates a Bearer JWT and injects claims into the request context.
func RequireAuth(verifier JWTVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"authorization required"}`, http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := verifier.VerifyAccessToken(tokenStr)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), auth.ContextKeyUserID, claims.Subject)
			ctx = context.WithValue(ctx, auth.ContextKeyEmail, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
