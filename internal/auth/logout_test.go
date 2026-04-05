package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- fakes for logout ---

type fakeLogoutTokenStore struct {
	refreshTokens map[string]string
	revokedAll    []string
}

func newFakeLogoutTokenStore() *fakeLogoutTokenStore {
	return &fakeLogoutTokenStore{refreshTokens: make(map[string]string)}
}

func (s *fakeLogoutTokenStore) ConsumeRefreshToken(_ context.Context, token string) (string, error) {
	uid, ok := s.refreshTokens[token]
	if !ok {
		return "", ErrTokenNotFound
	}
	delete(s.refreshTokens, token)
	return uid, nil
}

func (s *fakeLogoutTokenStore) RevokeAllUserSessions(_ context.Context, userID string) error {
	s.revokedAll = append(s.revokedAll, userID)
	return nil
}

func newLogoutHandler(store *fakeLogoutTokenStore) *LogoutHandler {
	return &LogoutHandler{tokens: store, jwt: nil, notifier: nil}
}

// --- tests ---

func TestLogout_WithValidCookie_ConsumesToken(t *testing.T) {
	store := newFakeLogoutTokenStore()
	store.refreshTokens["active-refresh-token"] = "user-uuid-123"

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "active-refresh-token"})
	rec := httptest.NewRecorder()

	newLogoutHandler(store).Logout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if _, ok := store.refreshTokens["active-refresh-token"]; ok {
		t.Error("refresh token should be consumed on logout")
	}

	// Cookie should be cleared
	cookies := rec.Result().Cookies()
	for _, c := range cookies {
		if c.Name == refreshCookieName && c.MaxAge != -1 {
			t.Errorf("expected refresh cookie to be expired (MaxAge=-1), got MaxAge=%d", c.MaxAge)
		}
	}
}

func TestLogout_WithoutCookie_StillReturns200(t *testing.T) {
	store := newFakeLogoutTokenStore()

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rec := httptest.NewRecorder()

	newLogoutHandler(store).Logout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 even without cookie, got %d", rec.Code)
	}
}

func TestLogout_WithInvalidToken_StillReturns200(t *testing.T) {
	store := newFakeLogoutTokenStore()

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "unknown-token"})
	rec := httptest.NewRecorder()

	newLogoutHandler(store).Logout(rec, req)

	// Idempotent — always 200
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for unknown token, got %d", rec.Code)
	}
}

func TestRevokeAll_WithAuthContext_RevokesAndReturns200(t *testing.T) {
	store := newFakeLogoutTokenStore()
	h := newLogoutHandler(store)

	ctx := context.WithValue(context.Background(), ContextKeyUserID, "user-uuid-456")
	req := httptest.NewRequest(http.MethodPost, "/auth/logout/all", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	h.RevokeAll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if len(store.revokedAll) == 0 || store.revokedAll[0] != "user-uuid-456" {
		t.Errorf("expected user-uuid-456 to be revoked, got %v", store.revokedAll)
	}
}

func TestRevokeAll_WithoutAuthContext_Returns200(t *testing.T) {
	store := newFakeLogoutTokenStore()
	h := newLogoutHandler(store)

	// No user_id in context
	req := httptest.NewRequest(http.MethodPost, "/auth/logout/all", nil)
	rec := httptest.NewRecorder()

	h.RevokeAll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 even without auth context, got %d", rec.Code)
	}
	if len(store.revokedAll) != 0 {
		t.Error("expected no revocation when user_id not in context")
	}
}

func TestLogout_ClearsCookiePath(t *testing.T) {
	store := newFakeLogoutTokenStore()

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rec := httptest.NewRecorder()

	newLogoutHandler(store).Logout(rec, req)

	found := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == refreshCookieName {
			found = true
		}
	}
	if !found {
		t.Fatal("expected refresh_token cookie to be set (cleared) in response")
	}
}

func TestBackChannelLogoutReceiver_ValidBody_Returns200(t *testing.T) {
	// Build a concrete *TokenStore substitute — BackChannelLogoutReceiver takes *TokenStore
	// but only reads the body and ignores it. Pass nil to exercise the stub path.
	handler := BackChannelLogoutReceiver(nil)

	body := `{"logout_token":"some-signed-jwt"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/backchannel-logout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestBackChannelLogoutReceiver_InvalidJSON_Returns400(t *testing.T) {
	handler := BackChannelLogoutReceiver(nil)

	req := httptest.NewRequest(http.MethodPost, "/auth/backchannel-logout", strings.NewReader(`{bad`))
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rec.Code)
	}
}

// capturingNotifier records back-channel logout calls for test assertions.
type capturingNotifier struct {
	called chan struct{}
	err    error
}

func (n *capturingNotifier) NotifyLogout(_ context.Context, _, _ string) error {
	close(n.called)
	return n.err
}

// errRevokeAllStore errors on RevokeAllUserSessions.
type errRevokeAllStore struct {
	*fakeLogoutTokenStore
}

func (s *errRevokeAllStore) RevokeAllUserSessions(_ context.Context, _ string) error {
	return errors.New("redis error")
}

func TestLogout_WithNotifier_CallsBackChannelLogout(t *testing.T) {
	keyPath, _ := newPKCS1KeyFile(t)
	jwt, err := NewTokenService(keyPath, "test-issuer", time.Minute)
	if err != nil {
		t.Fatalf("NewTokenService: %v", err)
	}

	store := newFakeLogoutTokenStore()
	store.refreshTokens["valid-token"] = "user-uuid-789"
	notifier := &capturingNotifier{called: make(chan struct{})}

	h := &LogoutHandler{tokens: store, jwt: jwt, notifier: notifier}

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "valid-token"})
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	// Wait for back-channel goroutine to complete.
	select {
	case <-notifier.called:
	case <-time.After(2 * time.Second):
		t.Fatal("back-channel logout goroutine did not complete in time")
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestLogout_WithNotifier_NotifyError_StillReturns200(t *testing.T) {
	keyPath, _ := newPKCS1KeyFile(t)
	jwt, err := NewTokenService(keyPath, "test-issuer", time.Minute)
	if err != nil {
		t.Fatalf("NewTokenService: %v", err)
	}

	store := newFakeLogoutTokenStore()
	store.refreshTokens["valid-token"] = "user-uuid-err"
	notifier := &capturingNotifier{called: make(chan struct{}), err: errors.New("notification failed")}

	h := &LogoutHandler{tokens: store, jwt: jwt, notifier: notifier}

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: refreshCookieName, Value: "valid-token"})
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	select {
	case <-notifier.called:
	case <-time.After(2 * time.Second):
		t.Fatal("back-channel logout goroutine did not complete in time")
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 even when notifier errors, got %d", rec.Code)
	}
}

func TestRevokeAll_StoreError_StillReturns200(t *testing.T) {
	base := newFakeLogoutTokenStore()
	store := &errRevokeAllStore{fakeLogoutTokenStore: base}

	h := &LogoutHandler{tokens: store, jwt: nil, notifier: nil}

	ctx := context.WithValue(context.Background(), ContextKeyUserID, "user-uuid-error")
	req := httptest.NewRequest(http.MethodPost, "/auth/logout/all", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	h.RevokeAll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 even when RevokeAll errors, got %d", rec.Code)
	}
}

// Compile-time check that fakeLogoutTokenStore satisfies LogoutTokenStore.
var _ LogoutTokenStore = (*fakeLogoutTokenStore)(nil)

// refreshTokenTTLExport is used by logout_test.go to avoid import cycle.
// We reference the package-level constant to ensure the test stays in sync.
var _ = time.Duration(refreshTokenTTL)
