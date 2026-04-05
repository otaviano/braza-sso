package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/otaviano/braza-sso/internal/auth"
	"github.com/otaviano/braza-sso/internal/user"
)

// --- additional fakes ---

// errTokenStore always errors on CreateEmailVerificationToken.
type errTokenStore struct{}

func (s *errTokenStore) CreateEmailVerificationToken(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", auth.ErrTokenNotFound
}

func (s *errTokenStore) ConsumeEmailVerificationToken(_ context.Context, token string) (string, error) {
	return "", auth.ErrTokenNotFound
}

// badIDTokenStore returns a token whose consumed value is not a valid UUID.
type badIDTokenStore struct {
	fakeTokenStore
}

func newBadIDTokenStore() *badIDTokenStore {
	return &badIDTokenStore{fakeTokenStore: fakeTokenStore{tokens: make(map[string]string)}}
}

func (s *badIDTokenStore) ConsumeEmailVerificationToken(_ context.Context, token string) (string, error) {
	if _, ok := s.tokens[token]; !ok {
		return "", auth.ErrTokenNotFound
	}
	delete(s.tokens, token)
	return "not-a-valid-uuid", nil
}

// errSetVerifiedRepo errors on SetEmailVerified.
type errSetVerifiedRepo struct {
	*fakeUserRepo
}

func (r *errSetVerifiedRepo) SetEmailVerified(_ uuid.UUID) error {
	return user.ErrNotFound
}

// --- tests ---

func TestRegister_EmptyEmail_Returns400(t *testing.T) {
	a := newAdapter()
	h := a.handler()

	body, _ := json.Marshal(map[string]string{"email": "", "password": "ValidPass1!aaa"})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty email, got %d", rec.Code)
	}
}

func TestRegister_InvalidJSON_Returns400(t *testing.T) {
	a := newAdapter()
	h := a.handler()

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte(`{bad json`)))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rec.Code)
	}
}

func TestRegister_EmailNormalized(t *testing.T) {
	a := newAdapter()
	h := a.handler()

	body, _ := json.Marshal(map[string]string{
		"email":    "  USER@EXAMPLE.COM  ",
		"password": "ValidPass1!aaa",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 after email normalization, got %d", rec.Code)
	}

	if _, err := a.users.FindByEmail("user@example.com"); err != nil {
		t.Fatal("normalized email should be persisted")
	}
}

func TestRegister_TokenCreationFailure_StillReturns201(t *testing.T) {
	// Even if the token store fails, registration succeeds silently.
	store := &errTokenStore{}
	mailer := &fakeMailer{}
	users := newFakeUserRepo()
	h := auth.NewRegistrationHandlerWithDeps(users, store, mailer, "pepper", "http://localhost")

	body, _ := json.Marshal(map[string]string{"email": "user@example.com", "password": "ValidPass1!aaa"})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 even when token store fails, got %d", rec.Code)
	}
}

func TestVerifyEmail_MissingToken_Returns400(t *testing.T) {
	a := newAdapter()
	h := a.handler()

	// No token query param at all
	req := httptest.NewRequest(http.MethodGet, "/auth/verify-email", nil)
	rec := httptest.NewRecorder()

	h.VerifyEmail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when token param is missing, got %d", rec.Code)
	}
}

func TestVerifyEmail_MalformedUserIDInStore_Returns500(t *testing.T) {
	users := newFakeUserRepo()
	store := newBadIDTokenStore()
	mailer := &fakeMailer{}

	// Seed a token directly
	store.tokens["some-token"] = "any-value"

	h := auth.NewRegistrationHandlerWithDeps(users, store, mailer, "pepper", "http://localhost")

	req := httptest.NewRequest(http.MethodGet, "/auth/verify-email?token=some-token", nil)
	rec := httptest.NewRecorder()

	h.VerifyEmail(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for malformed stored user ID, got %d", rec.Code)
	}
}

func TestVerifyEmail_SetEmailVerifiedError_Returns500(t *testing.T) {
	base := newFakeUserRepo()
	errRepo := &errSetVerifiedRepo{fakeUserRepo: base}
	store := newFakeTokenStore()
	mailer := &fakeMailer{}

	u := &user.User{ID: uuid.New(), Email: "user@example.com"}
	if err := base.Create(u); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	token, _ := store.CreateEmailVerificationToken(context.Background(), u.ID.String(), time.Hour)

	h := auth.NewRegistrationHandlerWithDeps(errRepo, store, mailer, "pepper", "http://localhost")

	req := httptest.NewRequest(http.MethodGet, "/auth/verify-email?token="+token, nil)
	rec := httptest.NewRecorder()

	h.VerifyEmail(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when SetEmailVerified fails, got %d", rec.Code)
	}
}

func TestResendVerification_InvalidJSON_Returns400(t *testing.T) {
	a := newAdapter()
	h := a.handler()

	req := httptest.NewRequest(http.MethodPost, "/auth/resend-verification", bytes.NewReader([]byte(`{bad json`)))
	rec := httptest.NewRecorder()

	h.ResendVerification(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rec.Code)
	}
}

func TestResendVerification_AlreadyVerified_Returns200Silently(t *testing.T) {
	a := newAdapter()
	h := a.handler()

	// Register and verify the user
	u := &user.User{ID: uuid.New(), Email: "verified@example.com", EmailVerified: true}
	if err := a.users.Create(u); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"email": "verified@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/auth/resend-verification", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.ResendVerification(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for already-verified email, got %d", rec.Code)
	}
	// No verification email should be sent
	if len(a.mailer.sent) != 0 {
		t.Error("expected no email sent for already-verified user")
	}
}

func TestResendVerification_KnownUnverifiedUser_Sends200(t *testing.T) {
	a := newAdapter()
	h := a.handler()

	u := &user.User{ID: uuid.New(), Email: "unverified@example.com", EmailVerified: false}
	if err := a.users.Create(u); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"email": "unverified@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/auth/resend-verification", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.ResendVerification(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
