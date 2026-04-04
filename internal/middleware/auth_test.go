package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/otaviano/braza-sso/internal/auth"
	"github.com/otaviano/braza-sso/internal/middleware"
)

// fakeVerifier is a test double for the JWTVerifier interface.
type fakeVerifier struct {
	claims *auth.Claims
	err    error
}

func (f *fakeVerifier) VerifyAccessToken(_ string) (*auth.Claims, error) {
	return f.claims, f.err
}

func okHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func applyAuth(verifier middleware.JWTVerifier) http.Handler {
	return middleware.RequireAuth(verifier)(http.HandlerFunc(okHandler))
}

func TestRequireAuth(t *testing.T) {
	t.Run("missing authorization header returns 401", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		applyAuth(&fakeVerifier{}).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("non-Bearer authorization scheme returns 401", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
		applyAuth(&fakeVerifier{}).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("invalid token returns 401", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer badtoken")
		applyAuth(&fakeVerifier{err: auth.ErrHashMismatch}).ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("valid token injects claims and calls next handler", func(t *testing.T) {
		claims := &auth.Claims{}
		claims.Subject = "user-123"
		claims.Email = "user@example.com"

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer validtoken")
		applyAuth(&fakeVerifier{claims: claims}).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}
