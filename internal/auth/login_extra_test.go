package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/otaviano/braza-sso/internal/user"
)

// --- error-injecting fakes ---

type errLoginRepo struct {
	findByEmailErr error
	base           *fakeLoginRepo
}

func (r *errLoginRepo) FindByEmail(email string) (*user.User, error) {
	if r.findByEmailErr != nil {
		return nil, r.findByEmailErr
	}
	return r.base.FindByEmail(email)
}

func (r *errLoginRepo) FindByID(id uuid.UUID) (*user.User, error) {
	return r.base.FindByID(id)
}

func (r *errLoginRepo) UpdateFailedAttempts(id uuid.UUID, attempts int, lockedUntil *time.Time) error {
	return r.base.UpdateFailedAttempts(id, attempts, lockedUntil)
}

func (r *errLoginRepo) UpdateFailedAttemptsReset(id uuid.UUID) error {
	return r.base.UpdateFailedAttemptsReset(id)
}

type errJWT struct{ err error }

func (f *errJWT) IssueAccessToken(_, _ string, _ bool, _ string) (string, error) {
	return "", f.err
}

// errStoreLoginTokenStore wraps fakeLoginTokenStore and injects an error on StoreRefreshToken.
type errStoreLoginTokenStore struct {
	*fakeLoginTokenStore
	storeRefreshErr error
}

func (s *errStoreLoginTokenStore) StoreRefreshToken(_ context.Context, _, _ string, _ time.Duration) error {
	return s.storeRefreshErr
}

// noByIDRepo implements LoginRepository but NOT the byID interface upgrade.
type noByIDRepo struct {
	base *fakeLoginRepo
}

func (r *noByIDRepo) FindByEmail(email string) (*user.User, error) {
	return r.base.FindByEmail(email)
}

func (r *noByIDRepo) UpdateFailedAttempts(id uuid.UUID, attempts int, lockedUntil *time.Time) error {
	return r.base.UpdateFailedAttempts(id, attempts, lockedUntil)
}

func (r *noByIDRepo) UpdateFailedAttemptsReset(id uuid.UUID) error {
	return r.base.UpdateFailedAttemptsReset(id)
}

// --- tests ---

func TestLogin_EmptyEmail_Returns400(t *testing.T) {
	repo := newFakeLoginRepo()
	store := newFakeLoginTokenStore()

	body, _ := json.Marshal(map[string]string{"email": "", "password": "ValidPass1!aaa"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newLoginHandler(repo, store).Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty email, got %d", rec.Code)
	}
}

func TestLogin_EmptyPassword_Returns400(t *testing.T) {
	repo := newFakeLoginRepo()
	store := newFakeLoginTokenStore()

	body, _ := json.Marshal(map[string]string{"email": "user@example.com", "password": ""})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newLoginHandler(repo, store).Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty password, got %d", rec.Code)
	}
}

func TestLogin_InvalidJSON_Returns400(t *testing.T) {
	repo := newFakeLoginRepo()
	store := newFakeLoginTokenStore()

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte(`{bad json`)))
	rec := httptest.NewRecorder()

	newLoginHandler(repo, store).Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rec.Code)
	}
}

func TestLogin_EmailNormalized(t *testing.T) {
	repo := newFakeLoginRepo()
	store := newFakeLoginTokenStore()
	u := mustUserWithPassword("user@example.com", "ValidPass1!aaa", testPepper)
	repo.add(u)

	// email with uppercase and surrounding whitespace
	body, _ := json.Marshal(map[string]string{"email": "  USER@EXAMPLE.COM  ", "password": "ValidPass1!aaa"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newLoginHandler(repo, store).Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 after email normalization, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRefresh_MalformedUserIDInStore_Returns500(t *testing.T) {
	repo := newFakeLoginRepo()
	store := newFakeLoginTokenStore()

	// Store a refresh token whose value is not a valid UUID
	store.refreshTokens["token-bad-uuid"] = "not-a-uuid"

	req := httptest.NewRequest(http.MethodPost, "/auth/token/refresh", nil)
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "token-bad-uuid"})
	rec := httptest.NewRecorder()

	newLoginHandler(repo, store).Refresh(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for malformed UUID in store, got %d", rec.Code)
	}
}

func TestRefresh_UserNotFound_Returns401(t *testing.T) {
	repo := newFakeLoginRepo()
	store := newFakeLoginTokenStore()

	// Token maps to a valid UUID, but no user exists in repo
	missingID := uuid.New()
	store.refreshTokens["orphan-token"] = missingID.String()

	req := httptest.NewRequest(http.MethodPost, "/auth/token/refresh", nil)
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "orphan-token"})
	rec := httptest.NewRecorder()

	newLoginHandler(repo, store).Refresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when user not found, got %d", rec.Code)
	}
}

func TestRefresh_RepoWithoutFindByID_Returns401(t *testing.T) {
	base := newFakeLoginRepo()
	noByID := &noByIDRepo{base: base}
	store := newFakeLoginTokenStore()

	validID := uuid.New()
	store.refreshTokens["some-token"] = validID.String()

	h := NewLoginHandlerWithDeps(noByID, store, &fakeJWT{}, &loginFakeMailer{}, testPepper, "http://localhost", "braza-sso")

	req := httptest.NewRequest(http.MethodPost, "/auth/token/refresh", nil)
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "some-token"})
	rec := httptest.NewRecorder()

	h.Refresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when FindByID not supported, got %d", rec.Code)
	}
}

func TestLogin_AccountAlreadyLocked(t *testing.T) {
	repo := newFakeLoginRepo()
	store := newFakeLoginTokenStore()

	future := time.Now().Add(1 * time.Hour)
	u := mustUserWithPassword("locked@example.com", "ValidPass1!aaa", testPepper)
	u.LockedUntil = &future
	repo.add(u)

	body, _ := json.Marshal(map[string]string{"email": "locked@example.com", "password": "ValidPass1!aaa"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newLoginHandler(repo, store).Login(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for pre-locked account, got %d", rec.Code)
	}
}

func TestLogin_JWTError_Returns500(t *testing.T) {
	repo := newFakeLoginRepo()
	store := newFakeLoginTokenStore()
	u := mustUserWithPassword("user@example.com", "ValidPass1!aaa", testPepper)
	repo.add(u)

	jwtErr := &errJWT{err: errors.New("key broken")}
	h := NewLoginHandlerWithDeps(repo, store, jwtErr, &loginFakeMailer{}, testPepper, "http://localhost", "braza-sso")

	body, _ := json.Marshal(map[string]string{"email": "user@example.com", "password": "ValidPass1!aaa"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when JWT issuance fails, got %d", rec.Code)
	}
}

func TestLogin_StoreRefreshError_Returns500(t *testing.T) {
	repo := newFakeLoginRepo()
	base := newFakeLoginTokenStore()
	store := &errStoreLoginTokenStore{
		fakeLoginTokenStore: base,
		storeRefreshErr:     errors.New("redis down"),
	}
	u := mustUserWithPassword("user@example.com", "ValidPass1!aaa", testPepper)
	repo.add(u)

	h := NewLoginHandlerWithDeps(repo, store, &fakeJWT{}, &loginFakeMailer{}, testPepper, "http://localhost", "braza-sso")

	body, _ := json.Marshal(map[string]string{"email": "user@example.com", "password": "ValidPass1!aaa"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when refresh token store fails, got %d", rec.Code)
	}
}

// errIncrLoginTokenStore errors on IncrLoginAttempts.
type errIncrLoginTokenStore struct {
	*fakeLoginTokenStore
}

func (s *errIncrLoginTokenStore) IncrLoginAttempts(_ context.Context, _ string) (int64, error) {
	return 0, errors.New("redis error")
}

func TestLogin_IncrLoginAttemptsError_HandledGracefully(t *testing.T) {
	// When IncrLoginAttempts errors, handleFailedAttempt returns early — no panic, still 401.
	repo := newFakeLoginRepo()
	base := newFakeLoginTokenStore()
	store := &errIncrLoginTokenStore{fakeLoginTokenStore: base}

	u := mustUserWithPassword("user@example.com", "ValidPass1!aaa", testPepper)
	repo.add(u)

	body, _ := json.Marshal(map[string]string{"email": "user@example.com", "password": "WrongPass1!aaa"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h := NewLoginHandlerWithDeps(repo, store, &fakeJWT{}, &loginFakeMailer{}, testPepper, "http://localhost", "braza-sso")
	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on wrong password (even with Redis error), got %d", rec.Code)
	}
}

func TestLogin_FindByEmailError_Returns401(t *testing.T) {
	base := newFakeLoginRepo()
	store := newFakeLoginTokenStore()
	repo := &errLoginRepo{findByEmailErr: errors.New("db error"), base: base}

	body, _ := json.Marshal(map[string]string{"email": "user@example.com", "password": "ValidPass1!aaa"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	NewLoginHandlerWithDeps(repo, store, &fakeJWT{}, &loginFakeMailer{}, testPepper, "http://localhost", "braza-sso").Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when FindByEmail errors, got %d", rec.Code)
	}
}

func TestLogin_SuccessResetsFailedAttempts(t *testing.T) {
	repo := newFakeLoginRepo()
	store := newFakeLoginTokenStore()

	u := mustUserWithPassword("user@example.com", "ValidPass1!aaa", testPepper)
	u.FailedAttempts = 2
	repo.add(u)

	body, _ := json.Marshal(map[string]string{"email": "user@example.com", "password": "ValidPass1!aaa"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newLoginHandler(repo, store).Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	// ResetLoginAttempts should have been called in Redis store
	if _, hasAttempts := store.loginAttempts[u.ID.String()]; hasAttempts {
		t.Error("expected login attempts to be cleared from store")
	}
	// Repo's in-memory user should have attempts reset
	if repo.users[u.ID].FailedAttempts != 0 {
		t.Errorf("expected FailedAttempts=0, got %d", repo.users[u.ID].FailedAttempts)
	}
}
