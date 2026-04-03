package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/otaviano/braza-sso/internal/user"
)

// --- fake password reset repository ---

type fakePwdRepo struct {
	byEmail         map[string]*user.User
	updatedPassword map[uuid.UUID]string
}

func newFakePwdRepo() *fakePwdRepo {
	return &fakePwdRepo{
		byEmail:         make(map[string]*user.User),
		updatedPassword: make(map[uuid.UUID]string),
	}
}

func (r *fakePwdRepo) add(u *user.User) { r.byEmail[u.Email] = u }

func (r *fakePwdRepo) FindByEmail(email string) (*user.User, error) {
	u, ok := r.byEmail[email]
	if !ok {
		return nil, user.ErrNotFound
	}
	return u, nil
}

func (r *fakePwdRepo) UpdatePassword(id uuid.UUID, hash string) error {
	r.updatedPassword[id] = hash
	return nil
}

// --- fake password reset token store ---

type fakePwdTokenStore struct {
	resetTokens  map[string]string
	revokedUsers []string
}

func newFakePwdTokenStore() *fakePwdTokenStore {
	return &fakePwdTokenStore{resetTokens: make(map[string]string)}
}

func (s *fakePwdTokenStore) CreatePasswordResetToken(_ context.Context, userID string, _ time.Duration) (string, error) {
	token := "reset-token-" + userID
	s.resetTokens[token] = userID
	return token, nil
}

func (s *fakePwdTokenStore) ConsumePasswordResetToken(_ context.Context, token string) (string, error) {
	uid, ok := s.resetTokens[token]
	if !ok {
		return "", ErrTokenNotFound
	}
	delete(s.resetTokens, token)
	return uid, nil
}

func (s *fakePwdTokenStore) RevokeAllUserSessions(_ context.Context, userID string) error {
	s.revokedUsers = append(s.revokedUsers, userID)
	return nil
}

// --- helper ---

func newPwdResetHandler(repo *fakePwdRepo, store *fakePwdTokenStore) *PasswordResetHandler {
	return NewPasswordResetHandlerWithDeps(repo, store, &loginFakeMailer{}, "pepper", "http://localhost")
}

// --- tests ---

func TestResetRequest_AlwaysReturns200(t *testing.T) {
	repo := newFakePwdRepo()
	store := newFakePwdTokenStore()

	// Non-existent email
	body, _ := json.Marshal(map[string]string{"email": "nobody@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/auth/password/reset-request", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newPwdResetHandler(repo, store).ResetRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if len(store.resetTokens) != 0 {
		t.Error("no token should be created for unknown email")
	}
}

func TestResetRequest_KnownEmail_TokenCreated(t *testing.T) {
	repo := newFakePwdRepo()
	store := newFakePwdTokenStore()
	u := &user.User{ID: uuid.New(), Email: "user@example.com"}
	repo.add(u)

	body, _ := json.Marshal(map[string]string{"email": "user@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/auth/password/reset-request", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newPwdResetHandler(repo, store).ResetRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if len(store.resetTokens) != 1 {
		t.Error("expected one reset token to be created")
	}
}

func TestReset_Success(t *testing.T) {
	repo := newFakePwdRepo()
	store := newFakePwdTokenStore()
	u := &user.User{ID: uuid.New(), Email: "user@example.com"}
	repo.add(u)

	// Pre-create a reset token
	token, _ := store.CreatePasswordResetToken(context.Background(), u.ID.String(), time.Hour)

	body, _ := json.Marshal(map[string]string{
		"token":    token,
		"password": "NewValidPass1!",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/password/reset", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newPwdResetHandler(repo, store).Reset(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Password hash should be updated
	if _, ok := repo.updatedPassword[u.ID]; !ok {
		t.Error("expected password to be updated")
	}

	// All sessions should be revoked
	if len(store.revokedUsers) == 0 {
		t.Error("expected sessions to be revoked")
	}
}

func TestReset_InvalidToken(t *testing.T) {
	repo := newFakePwdRepo()
	store := newFakePwdTokenStore()

	body, _ := json.Marshal(map[string]string{
		"token":    "bad-token",
		"password": "NewValidPass1!",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/password/reset", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newPwdResetHandler(repo, store).Reset(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestReset_WeakPassword(t *testing.T) {
	repo := newFakePwdRepo()
	store := newFakePwdTokenStore()
	u := &user.User{ID: uuid.New(), Email: "user@example.com"}
	repo.add(u)

	token, _ := store.CreatePasswordResetToken(context.Background(), u.ID.String(), time.Hour)

	body, _ := json.Marshal(map[string]string{
		"token":    token,
		"password": "weak",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/password/reset", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newPwdResetHandler(repo, store).Reset(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rec.Code)
	}

	// Token should NOT be consumed on validation failure
	if len(store.resetTokens) == 0 {
		t.Error("token should not be consumed when password validation fails")
	}
}

func TestReset_TokenConsumedOnce(t *testing.T) {
	repo := newFakePwdRepo()
	store := newFakePwdTokenStore()
	u := &user.User{ID: uuid.New(), Email: "user@example.com"}
	repo.add(u)

	token, _ := store.CreatePasswordResetToken(context.Background(), u.ID.String(), time.Hour)

	// First reset succeeds
	body1, _ := json.Marshal(map[string]string{"token": token, "password": "NewValidPass1!"})
	newPwdResetHandler(repo, store).Reset(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/auth/password/reset", bytes.NewReader(body1)))

	// Second reset with same token should fail
	body2, _ := json.Marshal(map[string]string{"token": token, "password": "AnotherPass1!"})
	req2 := httptest.NewRequest(http.MethodPost, "/auth/password/reset", bytes.NewReader(body2))
	rec2 := httptest.NewRecorder()
	newPwdResetHandler(repo, store).Reset(rec2, req2)

	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on token reuse, got %d", rec2.Code)
	}
}
