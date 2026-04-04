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

// --- fakes ---

type fakeUserRepo struct {
	users map[string]*user.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: make(map[string]*user.User)}
}

func (r *fakeUserRepo) Create(u *user.User) error {
	if _, exists := r.users[u.Email]; exists {
		return user.ErrEmailTaken
	}
	r.users[u.Email] = u
	return nil
}

func (r *fakeUserRepo) FindByEmail(email string) (*user.User, error) {
	u, ok := r.users[email]
	if !ok {
		return nil, user.ErrNotFound
	}
	return u, nil
}

func (r *fakeUserRepo) SetEmailVerified(id uuid.UUID) error {
	for _, u := range r.users {
		if u.ID == id {
			u.EmailVerified = true
			return nil
		}
	}
	return user.ErrNotFound
}

type fakeTokenStore struct {
	tokens map[string]string
}

func newFakeTokenStore() *fakeTokenStore {
	return &fakeTokenStore{tokens: make(map[string]string)}
}

func (s *fakeTokenStore) CreateEmailVerificationToken(_ context.Context, userID string, _ time.Duration) (string, error) {
	token := "test-token-" + userID
	s.tokens[token] = userID
	return token, nil
}

func (s *fakeTokenStore) ConsumeEmailVerificationToken(_ context.Context, token string) (string, error) {
	uid, ok := s.tokens[token]
	if !ok {
		return "", auth.ErrTokenNotFound
	}
	delete(s.tokens, token)
	return uid, nil
}

type fakeMailer struct {
	sent []string
}

func (m *fakeMailer) SendVerification(to, _ string) error {
	m.sent = append(m.sent, to)
	return nil
}

func (m *fakeMailer) SendPasswordReset(_, _ string) error  { return nil }
func (m *fakeMailer) SendAccountLocked(_, _ string) error  { return nil }

// --- registration handler adapter ---

// registrationHandlerAdapter wraps the fake stores to satisfy handler interfaces.
type registrationAdapter struct {
	users  *fakeUserRepo
	tokens *fakeTokenStore
	mailer *fakeMailer
}

func newAdapter() *registrationAdapter {
	return &registrationAdapter{
		users:  newFakeUserRepo(),
		tokens: newFakeTokenStore(),
		mailer: &fakeMailer{},
	}
}

func (a *registrationAdapter) handler() *auth.RegistrationHandler {
	return auth.NewRegistrationHandlerWithDeps(a.users, a.tokens, a.mailer, "pepper", "http://localhost")
}

// --- tests ---

func TestRegister_Success(t *testing.T) {
	a := newAdapter()
	h := a.handler()

	body, _ := json.Marshal(map[string]string{
		"email":    "user@example.com",
		"password": "ValidPass1!aaa",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	if _, err := a.users.FindByEmail("user@example.com"); err != nil {
		t.Fatal("user should be persisted")
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	a := newAdapter()
	h := a.handler()

	body, _ := json.Marshal(map[string]string{
		"email":    "user@example.com",
		"password": "weak",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Register(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rec.Code)
	}
}

func TestRegister_DuplicateEmail_SilentSuccess(t *testing.T) {
	a := newAdapter()
	h := a.handler()

	body, _ := json.Marshal(map[string]string{
		"email":    "user@example.com",
		"password": "ValidPass1!aaa",
	})

	req1 := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	rec1 := httptest.NewRecorder()
	h.Register(rec1, req1)

	body2, _ := json.Marshal(map[string]string{
		"email":    "user@example.com",
		"password": "ValidPass1!aaa",
	})
	req2 := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body2))
	rec2 := httptest.NewRecorder()
	h.Register(rec2, req2)

	// Should still be 201 (silent) to prevent enumeration
	if rec2.Code != http.StatusCreated {
		t.Fatalf("expected 201 for duplicate email, got %d", rec2.Code)
	}
}

func TestVerifyEmail_Success(t *testing.T) {
	a := newAdapter()
	h := a.handler()

	// Create a user
	u := &user.User{ID: uuid.New(), Email: "user@example.com"}
	_ = a.users.Create(u)

	// Create a token manually
	token, _ := a.tokens.CreateEmailVerificationToken(context.Background(), u.ID.String(), time.Hour)

	req := httptest.NewRequest(http.MethodGet, "/auth/verify-email?token="+token, nil)
	rec := httptest.NewRecorder()

	h.VerifyEmail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	stored, _ := a.users.FindByEmail("user@example.com")
	if !stored.EmailVerified {
		t.Fatal("email should be marked as verified")
	}
}

func TestVerifyEmail_InvalidToken(t *testing.T) {
	a := newAdapter()
	h := a.handler()

	req := httptest.NewRequest(http.MethodGet, "/auth/verify-email?token=bad-token", nil)
	rec := httptest.NewRecorder()

	h.VerifyEmail(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestVerifyEmail_TokenConsumedOnce(t *testing.T) {
	a := newAdapter()
	h := a.handler()

	u := &user.User{ID: uuid.New(), Email: "user@example.com"}
	_ = a.users.Create(u)

	token, _ := a.tokens.CreateEmailVerificationToken(context.Background(), u.ID.String(), time.Hour)

	// First use
	req1 := httptest.NewRequest(http.MethodGet, "/auth/verify-email?token="+token, nil)
	h.VerifyEmail(httptest.NewRecorder(), req1)

	// Second use — token should be consumed
	req2 := httptest.NewRequest(http.MethodGet, "/auth/verify-email?token="+token, nil)
	rec2 := httptest.NewRecorder()
	h.VerifyEmail(rec2, req2)

	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on second use, got %d", rec2.Code)
	}
}

func TestResendVerification_AlwaysReturns200(t *testing.T) {
	a := newAdapter()
	h := a.handler()

	body, _ := json.Marshal(map[string]string{"email": "nonexistent@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/auth/resend-verification", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.ResendVerification(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
