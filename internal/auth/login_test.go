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

// --- fake login repository ---

type fakeLoginRepo struct {
	users          map[uuid.UUID]*user.User
	byEmail        map[string]*user.User
	attemptsSet    map[uuid.UUID]int
	lockedUntilSet map[uuid.UUID]*time.Time
}

func newFakeLoginRepo() *fakeLoginRepo {
	return &fakeLoginRepo{
		users:          make(map[uuid.UUID]*user.User),
		byEmail:        make(map[string]*user.User),
		attemptsSet:    make(map[uuid.UUID]int),
		lockedUntilSet: make(map[uuid.UUID]*time.Time),
	}
}

func (r *fakeLoginRepo) add(u *user.User) {
	r.users[u.ID] = u
	r.byEmail[u.Email] = u
}

func (r *fakeLoginRepo) FindByEmail(email string) (*user.User, error) {
	u, ok := r.byEmail[email]
	if !ok {
		return nil, user.ErrNotFound
	}
	return u, nil
}

func (r *fakeLoginRepo) FindByID(id uuid.UUID) (*user.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, user.ErrNotFound
	}
	return u, nil
}

func (r *fakeLoginRepo) UpdateFailedAttempts(id uuid.UUID, attempts int, lockedUntil *time.Time) error {
	r.attemptsSet[id] = attempts
	r.lockedUntilSet[id] = lockedUntil
	// Also update the in-memory user so lockout checks work
	if u, ok := r.users[id]; ok {
		u.FailedAttempts = attempts
		u.LockedUntil = lockedUntil
	}
	return nil
}

func (r *fakeLoginRepo) UpdateFailedAttemptsReset(id uuid.UUID) error {
	r.attemptsSet[id] = 0
	r.lockedUntilSet[id] = nil
	if u, ok := r.users[id]; ok {
		u.FailedAttempts = 0
		u.LockedUntil = nil
	}
	return nil
}

// --- fake login token store ---

type fakeLoginTokenStore struct {
	loginAttempts  map[string]int64
	refreshTokens  map[string]string
	mfaSessions    map[string]string
	revokedAll     []string
}

func newFakeLoginTokenStore() *fakeLoginTokenStore {
	return &fakeLoginTokenStore{
		loginAttempts: make(map[string]int64),
		refreshTokens: make(map[string]string),
		mfaSessions:   make(map[string]string),
	}
}

func (s *fakeLoginTokenStore) IncrLoginAttempts(_ context.Context, userID string) (int64, error) {
	s.loginAttempts[userID]++
	return s.loginAttempts[userID], nil
}

func (s *fakeLoginTokenStore) ResetLoginAttempts(_ context.Context, userID string) error {
	delete(s.loginAttempts, userID)
	return nil
}

func (s *fakeLoginTokenStore) StoreRefreshToken(_ context.Context, token, userID string, _ time.Duration) error {
	s.refreshTokens[token] = userID
	return nil
}

func (s *fakeLoginTokenStore) ConsumeRefreshToken(_ context.Context, token string) (string, error) {
	uid, ok := s.refreshTokens[token]
	if !ok {
		return "", ErrTokenNotFound
	}
	delete(s.refreshTokens, token)
	return uid, nil
}

func (s *fakeLoginTokenStore) RevokeAllUserSessions(_ context.Context, userID string) error {
	s.revokedAll = append(s.revokedAll, userID)
	return nil
}

func (s *fakeLoginTokenStore) StoreMFASession(_ context.Context, token, userID string, _ time.Duration) error {
	s.mfaSessions[token] = userID
	return nil
}

// --- fake mailer (local to this package test) ---

type loginFakeMailer struct{ sent []string }

func (m *loginFakeMailer) SendVerification(to, _ string) error  { m.sent = append(m.sent, to); return nil }
func (m *loginFakeMailer) SendPasswordReset(_, _ string) error  { return nil }
func (m *loginFakeMailer) SendAccountLocked(to, _ string) error { m.sent = append(m.sent, to); return nil }

// --- fake JWT service ---

type fakeJWT struct{}

func (f *fakeJWT) IssueAccessToken(userID, email string, emailVerified bool, audience string) (string, error) {
	return "fake-access-token-" + userID, nil
}

// --- helper to build a test user with a hashed password ---

func mustUserWithPassword(email, password, pepper string) *user.User {
	hash, err := HashPassword(password, pepper)
	if err != nil {
		panic(err)
	}
	return &user.User{
		ID:            uuid.New(),
		Email:         email,
		PasswordHash:  hash,
		EmailVerified: true,
	}
}

// --- tests ---

const testPepper = "test-pepper"

func newLoginHandler(repo *fakeLoginRepo, store *fakeLoginTokenStore) *LoginHandler {
	return NewLoginHandlerWithDeps(repo, store, &fakeJWT{}, &loginFakeMailer{}, testPepper, "http://localhost", "braza-sso")
}

func TestLogin_Success(t *testing.T) {
	repo := newFakeLoginRepo()
	store := newFakeLoginTokenStore()
	u := mustUserWithPassword("user@example.com", "ValidPass1!aaa", testPepper)
	repo.add(u)

	body, _ := json.Marshal(map[string]string{"email": "user@example.com", "password": "ValidPass1!aaa"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newLoginHandler(repo, store).Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["access_token"] == "" {
		t.Fatal("expected access_token in response")
	}

	// Refresh cookie should be set
	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == refreshCookieName {
			found = true
			if !c.HttpOnly {
				t.Error("refresh cookie must be HttpOnly")
			}
			if !c.Secure {
				t.Error("refresh cookie must be Secure")
			}
		}
	}
	if !found {
		t.Fatal("refresh_token cookie not set")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := newFakeLoginRepo()
	store := newFakeLoginTokenStore()
	u := mustUserWithPassword("user@example.com", "ValidPass1!aaa", testPepper)
	repo.add(u)

	body, _ := json.Marshal(map[string]string{"email": "user@example.com", "password": "WrongPassword1!"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newLoginHandler(repo, store).Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if store.loginAttempts[u.ID.String()] != 1 {
		t.Error("expected 1 failed attempt tracked")
	}
}

func TestLogin_AccountLockoutAfterFiveFailures(t *testing.T) {
	repo := newFakeLoginRepo()
	store := newFakeLoginTokenStore()
	mailer := &loginFakeMailer{}
	u := mustUserWithPassword("user@example.com", "ValidPass1!aaa", testPepper)
	repo.add(u)

	h := NewLoginHandlerWithDeps(repo, store, &fakeJWT{}, mailer, testPepper, "http://localhost", "braza-sso")

	for i := 0; i < MaxLoginAttempts(); i++ {
		body, _ := json.Marshal(map[string]string{"email": "user@example.com", "password": "wrong"})
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
		h.Login(httptest.NewRecorder(), req)
	}

	// Account should now be locked
	if repo.lockedUntilSet[u.ID] == nil {
		t.Fatal("expected account to be locked")
	}

	// Update the in-memory user to reflect lockout for the next request
	u.LockedUntil = repo.lockedUntilSet[u.ID]

	body, _ := json.Marshal(map[string]string{"email": "user@example.com", "password": "ValidPass1!aaa"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for locked account, got %d", rec.Code)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	repo := newFakeLoginRepo()
	store := newFakeLoginTokenStore()

	body, _ := json.Marshal(map[string]string{"email": "nobody@example.com", "password": "ValidPass1!aaa"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newLoginHandler(repo, store).Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestLogin_MFARequired(t *testing.T) {
	repo := newFakeLoginRepo()
	store := newFakeLoginTokenStore()
	u := mustUserWithPassword("mfa@example.com", "ValidPass1!aaa", testPepper)
	u.TOTPEnabled = true
	repo.add(u)

	body, _ := json.Marshal(map[string]string{"email": "mfa@example.com", "password": "ValidPass1!aaa"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newLoginHandler(repo, store).Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["mfa_required"] != true {
		t.Error("expected mfa_required=true")
	}
	if resp["mfa_session_id"] == "" {
		t.Error("expected mfa_session_id")
	}
}

func TestTokenRefresh_Success(t *testing.T) {
	repo := newFakeLoginRepo()
	store := newFakeLoginTokenStore()
	u := mustUserWithPassword("user@example.com", "ValidPass1!aaa", testPepper)
	repo.add(u)

	// Pre-store a refresh token
	store.refreshTokens["valid-refresh-token"] = u.ID.String()

	req := httptest.NewRequest(http.MethodPost, "/auth/token/refresh", nil)
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "valid-refresh-token"})
	rec := httptest.NewRecorder()

	newLoginHandler(repo, store).Refresh(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Old token should be consumed
	if _, ok := store.refreshTokens["valid-refresh-token"]; ok {
		t.Error("old refresh token should be consumed")
	}
}

func TestTokenRefresh_InvalidToken(t *testing.T) {
	repo := newFakeLoginRepo()
	store := newFakeLoginTokenStore()

	req := httptest.NewRequest(http.MethodPost, "/auth/token/refresh", nil)
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "bad-token"})
	rec := httptest.NewRecorder()

	newLoginHandler(repo, store).Refresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestTokenRefresh_MissingCookie(t *testing.T) {
	repo := newFakeLoginRepo()
	store := newFakeLoginTokenStore()

	req := httptest.NewRequest(http.MethodPost, "/auth/token/refresh", nil)
	rec := httptest.NewRecorder()

	newLoginHandler(repo, store).Refresh(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
