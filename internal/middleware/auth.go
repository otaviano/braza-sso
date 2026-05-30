package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/otaviano/braza-sso/internal/auth"
)

// SessionLookup looks up a user ID from an SSO session token.
type SessionLookup interface {
	LookupSessionToken(ctx context.Context, token string) (string, error)
}

// OptionalSessionAuth populates the user ID in context if a valid session cookie is present.
// It does not block the request if no cookie is found.
func OptionalSessionAuth(lookup SessionLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token, ok := auth.SessionTokenFromCookie(r); ok {
				if userID, err := lookup.LookupSessionToken(r.Context(), token); err == nil {
					ctx := context.WithValue(r.Context(), auth.ContextKeyUserID, userID)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireSessionAuth validates an SSO session cookie and injects the user ID into context.
func RequireSessionAuth(lookup SessionLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := auth.SessionTokenFromCookie(r)
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"authorization required"}`, http.StatusUnauthorized)
				return
			}
			userID, err := lookup.LookupSessionToken(r.Context(), token)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"invalid or expired session"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), auth.ContextKeyUserID, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

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
