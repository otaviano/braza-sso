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
	"github.com/pquerna/otp/totp"
)

// --- fake TOTP user repository ---

type fakeTOTPUserRepo struct {
	users map[uuid.UUID]*user.User
}

func newFakeTOTPUserRepo() *fakeTOTPUserRepo {
	return &fakeTOTPUserRepo{users: make(map[uuid.UUID]*user.User)}
}

func (r *fakeTOTPUserRepo) add(u *user.User) { r.users[u.ID] = u }

func (r *fakeTOTPUserRepo) FindByID(id uuid.UUID) (*user.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, user.ErrNotFound
	}
	return u, nil
}

func (r *fakeTOTPUserRepo) UpdateTOTP(id uuid.UUID, secret string, enabled bool) error {
	if u, ok := r.users[id]; ok {
		u.TOTPSecret = secret
		u.TOTPEnabled = enabled
	}
	return nil
}

// --- fake recovery code store ---

type fakeCodes struct {
	codes map[uuid.UUID][]struct {
		CodeID   uuid.UUID
		CodeHash string
		Used     bool
	}
}

func newFakeCodes() *fakeCodes {
	return &fakeCodes{codes: make(map[uuid.UUID][]struct {
		CodeID   uuid.UUID
		CodeHash string
		Used     bool
	})}
}

func (c *fakeCodes) ReplaceAll(userID uuid.UUID, hashedCodes []string) error {
	var entries []struct {
		CodeID   uuid.UUID
		CodeHash string
		Used     bool
	}
	for _, h := range hashedCodes {
		entries = append(entries, struct {
			CodeID   uuid.UUID
			CodeHash string
			Used     bool
		}{uuid.New(), h, false})
	}
	c.codes[userID] = entries
	return nil
}

func (c *fakeCodes) ListUnused(userID uuid.UUID) ([]struct {
	CodeID   uuid.UUID
	CodeHash string
}, error) {
	var result []struct {
		CodeID   uuid.UUID
		CodeHash string
	}
	for _, e := range c.codes[userID] {
		if !e.Used {
			result = append(result, struct {
				CodeID   uuid.UUID
				CodeHash string
			}{e.CodeID, e.CodeHash})
		}
	}
	return result, nil
}

func (c *fakeCodes) MarkUsed(userID, codeID uuid.UUID) error {
	for i, e := range c.codes[userID] {
		if e.CodeID == codeID {
			c.codes[userID][i].Used = true
			return nil
		}
	}
	return nil
}

// --- fake TOTP token store ---

type fakeTOTPTokenStore struct {
	mfaSessions   map[string]string
	refreshTokens map[string]string
}

func newFakeTOTPTokenStore() *fakeTOTPTokenStore {
	return &fakeTOTPTokenStore{
		mfaSessions:   make(map[string]string),
		refreshTokens: make(map[string]string),
	}
}

func (s *fakeTOTPTokenStore) ConsumeMFASession(_ context.Context, token string) (string, error) {
	uid, ok := s.mfaSessions[token]
	if !ok {
		return "", ErrTokenNotFound
	}
	delete(s.mfaSessions, token)
	return uid, nil
}

func (s *fakeTOTPTokenStore) StoreRefreshToken(_ context.Context, token, userID string, _ time.Duration) error {
	s.refreshTokens[token] = userID
	return nil
}

// --- helper to build handler with auth context injected ---

func newTOTPHandler(repo *fakeTOTPUserRepo, codes *fakeCodes, store *fakeTOTPTokenStore) *TOTPHandler {
	return NewTOTPHandlerWithDeps(repo, codes, store, &fakeJWT{}, &loginFakeMailer{}, "pepper", "braza-sso")
}

func withUserID(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), ContextKeyUserID, userID)
	return r.WithContext(ctx)
}

// --- tests ---

func TestEnroll_Success(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()
	u := &user.User{ID: uuid.New(), Email: "user@example.com"}
	repo.add(u)

	req := withUserID(httptest.NewRequest(http.MethodPost, "/account/2fa/enroll", nil), u.ID.String())
	rec := httptest.NewRecorder()

	newTOTPHandler(repo, codes, store).Enroll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp enrollResponse
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	if resp.Secret == "" {
		t.Error("expected secret in response")
	}
	if resp.OTPURI == "" {
		t.Error("expected otp_uri in response")
	}
	if len(resp.RecoveryCodes) != recoveryCodeCount {
		t.Errorf("expected %d recovery codes, got %d", recoveryCodeCount, len(resp.RecoveryCodes))
	}

	// TOTP should be stored but not yet enabled
	if repo.users[u.ID].TOTPEnabled {
		t.Error("TOTP should not be enabled before confirmation")
	}
	if repo.users[u.ID].TOTPSecret == "" {
		t.Error("TOTP secret should be stored")
	}
}

func TestConfirm_Success(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	// Pre-enroll
	u := &user.User{ID: uuid.New(), Email: "user@example.com"}
	repo.add(u)

	enrollReq := withUserID(httptest.NewRequest(http.MethodPost, "/account/2fa/enroll", nil), u.ID.String())
	h := newTOTPHandler(repo, codes, store)
	h.Enroll(httptest.NewRecorder(), enrollReq)

	secret := repo.users[u.ID].TOTPSecret

	// Generate valid TOTP code
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("failed to generate TOTP code: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"code": code})
	req := withUserID(httptest.NewRequest(http.MethodPost, "/account/2fa/confirm", bytes.NewReader(body)), u.ID.String())
	rec := httptest.NewRecorder()
	h.Confirm(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !repo.users[u.ID].TOTPEnabled {
		t.Error("TOTP should be enabled after confirm")
	}
}

func TestConfirm_InvalidCode(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	u := &user.User{ID: uuid.New(), Email: "user@example.com", TOTPSecret: "JBSWY3DPEHPK3PXP"}
	repo.add(u)

	body, _ := json.Marshal(map[string]string{"code": "000000"})
	req := withUserID(httptest.NewRequest(http.MethodPost, "/account/2fa/confirm", bytes.NewReader(body)), u.ID.String())
	rec := httptest.NewRecorder()

	newTOTPHandler(repo, codes, store).Confirm(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestVerify_Success(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	u := &user.User{ID: uuid.New(), Email: "user@example.com", TOTPEnabled: true}
	repo.add(u)

	// Pre-enroll to get a real secret
	h := newTOTPHandler(repo, codes, store)
	enrollReq := withUserID(httptest.NewRequest(http.MethodPost, "/account/2fa/enroll", nil), u.ID.String())
	h.Enroll(httptest.NewRecorder(), enrollReq)
	secret := repo.users[u.ID].TOTPSecret

	// Store MFA session
	store.mfaSessions["mfa-token-123"] = u.ID.String()

	code, _ := totp.GenerateCode(secret, time.Now())
	body, _ := json.Marshal(map[string]string{"mfa_session_id": "mfa-token-123", "code": code})
	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/verify", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.Verify(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestVerify_ExpiredSession(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	body, _ := json.Marshal(map[string]string{"mfa_session_id": "expired", "code": "123456"})
	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/verify", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newTOTPHandler(repo, codes, store).Verify(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRecovery_Success(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codesStore := newFakeCodes()
	store := newFakeTOTPTokenStore()

	u := &user.User{ID: uuid.New(), Email: "user@example.com", TOTPEnabled: true}
	repo.add(u)

	h := newTOTPHandler(repo, codesStore, store)

	// Enroll to generate recovery codes
	enrollReq := withUserID(httptest.NewRequest(http.MethodPost, "/account/2fa/enroll", nil), u.ID.String())
	enrollRec := httptest.NewRecorder()
	h.Enroll(enrollRec, enrollReq)

	var enrollResp enrollResponse
	_ = json.NewDecoder(enrollRec.Body).Decode(&enrollResp)
	plainCode := enrollResp.RecoveryCodes[0]

	store.mfaSessions["mfa-recovery-session"] = u.ID.String()

	body, _ := json.Marshal(map[string]string{
		"mfa_session_id": "mfa-recovery-session",
		"recovery_code":  plainCode,
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/recovery", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.Recovery(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// TOTP should be disabled (prompt re-enrollment)
	if repo.users[u.ID].TOTPEnabled {
		t.Error("TOTP should be disabled after recovery code use")
	}
}

func TestRecovery_InvalidCode(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codesStore := newFakeCodes()
	store := newFakeTOTPTokenStore()

	u := &user.User{ID: uuid.New(), Email: "user@example.com", TOTPEnabled: true}
	repo.add(u)

	// No codes stored
	store.mfaSessions["mfa-session"] = u.ID.String()

	body, _ := json.Marshal(map[string]string{
		"mfa_session_id": "mfa-session",
		"recovery_code":  "wrong-code",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/recovery", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newTOTPHandler(repo, codesStore, store).Recovery(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
