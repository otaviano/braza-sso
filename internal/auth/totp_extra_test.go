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
	"github.com/pquerna/otp/totp"
)

func TestEnroll_NoUserIDInContext_Returns401(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	// No user ID in context
	req := httptest.NewRequest(http.MethodPost, "/account/2fa/enroll", nil)
	rec := httptest.NewRecorder()

	newTOTPHandler(repo, codes, store).Enroll(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when no user_id in context, got %d", rec.Code)
	}
}

func TestEnroll_InvalidUserIDFormat_Returns400(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	req := withUserID(httptest.NewRequest(http.MethodPost, "/account/2fa/enroll", nil), "not-a-uuid")
	rec := httptest.NewRecorder()

	newTOTPHandler(repo, codes, store).Enroll(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d", rec.Code)
	}
}

func TestEnroll_UserNotFound_Returns404(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	unknownID := uuid.New()
	req := withUserID(httptest.NewRequest(http.MethodPost, "/account/2fa/enroll", nil), unknownID.String())
	rec := httptest.NewRecorder()

	newTOTPHandler(repo, codes, store).Enroll(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when user not found, got %d", rec.Code)
	}
}

func TestConfirm_NoUserIDInContext_Returns401(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	body, _ := json.Marshal(map[string]string{"code": "123456"})
	req := httptest.NewRequest(http.MethodPost, "/account/2fa/confirm", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newTOTPHandler(repo, codes, store).Confirm(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when no user_id in context, got %d", rec.Code)
	}
}

func TestConfirm_InvalidUserIDFormat_Returns400(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	body, _ := json.Marshal(map[string]string{"code": "123456"})
	req := withUserID(httptest.NewRequest(http.MethodPost, "/account/2fa/confirm", bytes.NewReader(body)), "not-a-uuid")
	rec := httptest.NewRecorder()

	newTOTPHandler(repo, codes, store).Confirm(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d", rec.Code)
	}
}

func TestConfirm_EmptyCode_Returns400(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	u := &user.User{ID: uuid.New(), Email: "user@example.com", TOTPSecret: "JBSWY3DPEHPK3PXP"}
	repo.add(u)

	body, _ := json.Marshal(map[string]string{"code": ""})
	req := withUserID(httptest.NewRequest(http.MethodPost, "/account/2fa/confirm", bytes.NewReader(body)), u.ID.String())
	rec := httptest.NewRecorder()

	newTOTPHandler(repo, codes, store).Confirm(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty code, got %d", rec.Code)
	}
}

func TestConfirm_InvalidJSON_Returns400(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	u := &user.User{ID: uuid.New(), Email: "user@example.com", TOTPSecret: "JBSWY3DPEHPK3PXP"}
	repo.add(u)

	req := withUserID(httptest.NewRequest(http.MethodPost, "/account/2fa/confirm", bytes.NewReader([]byte(`{bad`))), u.ID.String())
	rec := httptest.NewRecorder()

	newTOTPHandler(repo, codes, store).Confirm(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rec.Code)
	}
}

func TestConfirm_UserNotFound_Returns404(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	unknownID := uuid.New()
	body, _ := json.Marshal(map[string]string{"code": "123456"})
	req := withUserID(httptest.NewRequest(http.MethodPost, "/account/2fa/confirm", bytes.NewReader(body)), unknownID.String())
	rec := httptest.NewRecorder()

	newTOTPHandler(repo, codes, store).Confirm(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when user not found, got %d", rec.Code)
	}
}

func TestVerify_MissingFields_Returns400(t *testing.T) {
	tests := []struct {
		name string
		body map[string]string
	}{
		{"missing mfa_session_id", map[string]string{"mfa_session_id": "", "code": "123456"}},
		{"missing code", map[string]string{"mfa_session_id": "session-id", "code": ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeTOTPUserRepo()
			codes := newFakeCodes()
			store := newFakeTOTPTokenStore()

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/auth/2fa/verify", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			newTOTPHandler(repo, codes, store).Verify(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", rec.Code)
			}
		})
	}
}

func TestVerify_InvalidJSON_Returns400(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/verify", bytes.NewReader([]byte(`{bad`)))
	rec := httptest.NewRecorder()

	newTOTPHandler(repo, codes, store).Verify(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestVerify_InvalidTOTPCode_Returns401(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	u := &user.User{ID: uuid.New(), Email: "user@example.com", TOTPEnabled: true, TOTPSecret: "JBSWY3DPEHPK3PXP"}
	repo.add(u)
	store.mfaSessions["valid-mfa-session"] = u.ID.String()

	body, _ := json.Marshal(map[string]string{"mfa_session_id": "valid-mfa-session", "code": "000000"})
	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/verify", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newTOTPHandler(repo, codes, store).Verify(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong TOTP code, got %d", rec.Code)
	}
}

func TestRecovery_MissingFields_Returns400(t *testing.T) {
	tests := []struct {
		name string
		body map[string]string
	}{
		{"missing mfa_session_id", map[string]string{"mfa_session_id": "", "recovery_code": "abc"}},
		{"missing recovery_code", map[string]string{"mfa_session_id": "session-id", "recovery_code": ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeTOTPUserRepo()
			codes := newFakeCodes()
			store := newFakeTOTPTokenStore()

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/auth/2fa/recovery", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			newTOTPHandler(repo, codes, store).Recovery(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", rec.Code)
			}
		})
	}
}

func TestRecovery_InvalidJSON_Returns400(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/recovery", bytes.NewReader([]byte(`{bad`)))
	rec := httptest.NewRecorder()

	newTOTPHandler(repo, codes, store).Recovery(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestRecovery_ExpiredMFASession_Returns401(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	body, _ := json.Marshal(map[string]string{"mfa_session_id": "expired", "recovery_code": "any-code"})
	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/recovery", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newTOTPHandler(repo, codes, store).Recovery(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired MFA session, got %d", rec.Code)
	}
}

// --- error-injecting fakes for TOTP ---

type errUpdateTOTPRepo struct {
	*fakeTOTPUserRepo
}

func (r *errUpdateTOTPRepo) UpdateTOTP(_ uuid.UUID, _ string, _ bool) error {
	return errors.New("storage failure")
}

type errReplaceAllCodes struct {
	*fakeCodes
}

func (c *errReplaceAllCodes) ReplaceAll(_ uuid.UUID, _ []string) error {
	return errors.New("codes storage failure")
}

type errListUnusedCodes struct {
	*fakeCodes
}

func (c *errListUnusedCodes) ListUnused(_ uuid.UUID) ([]struct {
	CodeID   uuid.UUID
	CodeHash string
}, error) {
	return nil, errors.New("list failure")
}

type errMarkUsedCodes struct {
	*fakeCodes
}

func (c *errMarkUsedCodes) MarkUsed(_, _ uuid.UUID) error {
	return errors.New("mark used failure")
}

type errTOTPJWT struct{}

func (j *errTOTPJWT) IssueAccessToken(_, _ string, _ bool, _ string) (string, error) {
	return "", errors.New("jwt failure")
}

type errStoreTOTPTokenStore struct {
	*fakeTOTPTokenStore
}

func (s *errStoreTOTPTokenStore) StoreRefreshToken(_ context.Context, _, _ string, _ time.Duration) error {
	return errors.New("store failure")
}

// --- enroll error path tests ---

func TestEnroll_UpdateTOTPError_Returns500(t *testing.T) {
	base := newFakeTOTPUserRepo()
	u := &user.User{ID: uuid.New(), Email: "user@example.com"}
	base.add(u)
	repo := &errUpdateTOTPRepo{fakeTOTPUserRepo: base}
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	req := withUserID(httptest.NewRequest(http.MethodPost, "/account/2fa/enroll", nil), u.ID.String())
	rec := httptest.NewRecorder()

	NewTOTPHandlerWithDeps(repo, codes, store, &fakeJWT{}, &loginFakeMailer{}, "pepper", "braza-sso").Enroll(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when UpdateTOTP fails, got %d", rec.Code)
	}
}

func TestEnroll_ReplaceAllCodesError_Returns500(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	u := &user.User{ID: uuid.New(), Email: "user@example.com"}
	repo.add(u)
	codes := &errReplaceAllCodes{fakeCodes: newFakeCodes()}
	store := newFakeTOTPTokenStore()

	req := withUserID(httptest.NewRequest(http.MethodPost, "/account/2fa/enroll", nil), u.ID.String())
	rec := httptest.NewRecorder()

	NewTOTPHandlerWithDeps(repo, codes, store, &fakeJWT{}, &loginFakeMailer{}, "pepper", "braza-sso").Enroll(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when ReplaceAll fails, got %d", rec.Code)
	}
}

// --- confirm error path test ---

func TestConfirm_UpdateTOTPError_Returns500(t *testing.T) {
	base := newFakeTOTPUserRepo()
	u := &user.User{ID: uuid.New(), Email: "user@example.com", TOTPSecret: "JBSWY3DPEHPK3PXP"}
	base.add(u)
	repo := &errUpdateTOTPRepo{fakeTOTPUserRepo: base}
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	code, err := totp.GenerateCode(u.TOTPSecret, time.Now())
	if err != nil {
		t.Fatalf("generate TOTP code: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"code": code})
	req := withUserID(httptest.NewRequest(http.MethodPost, "/account/2fa/confirm", bytes.NewReader(body)), u.ID.String())
	rec := httptest.NewRecorder()

	NewTOTPHandlerWithDeps(repo, codes, store, &fakeJWT{}, &loginFakeMailer{}, "pepper", "braza-sso").Confirm(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when UpdateTOTP fails during confirm, got %d", rec.Code)
	}
}

// --- verify error path tests (issueTokenPairForUser) ---

func TestVerify_MalformedSessionUUID_Returns500(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()
	store.mfaSessions["session-1"] = "not-a-uuid"

	body, _ := json.Marshal(map[string]string{"mfa_session_id": "session-1", "code": "123456"})
	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/verify", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newTOTPHandler(repo, codes, store).Verify(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for malformed session UUID, got %d", rec.Code)
	}
}

func TestVerify_UserNotFoundAfterSession_Returns401(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()
	store.mfaSessions["session-1"] = uuid.New().String()

	body, _ := json.Marshal(map[string]string{"mfa_session_id": "session-1", "code": "123456"})
	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/verify", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newTOTPHandler(repo, codes, store).Verify(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when user not found, got %d", rec.Code)
	}
}

func TestVerify_JWTError_Returns500(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	u := &user.User{ID: uuid.New(), Email: "user@example.com", TOTPEnabled: true}
	repo.add(u)

	code, err := totp.GenerateCode("JBSWY3DPEHPK3PXP", time.Now())
	if err != nil {
		t.Fatalf("generate TOTP code: %v", err)
	}
	u.TOTPSecret = "JBSWY3DPEHPK3PXP"
	store.mfaSessions["session-1"] = u.ID.String()

	body, _ := json.Marshal(map[string]string{"mfa_session_id": "session-1", "code": code})
	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/verify", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	NewTOTPHandlerWithDeps(repo, codes, store, &errTOTPJWT{}, &loginFakeMailer{}, "pepper", "braza-sso").Verify(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when JWT issuance fails, got %d", rec.Code)
	}
}

func TestVerify_StoreRefreshError_Returns500(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	base := newFakeTOTPTokenStore()
	store := &errStoreTOTPTokenStore{fakeTOTPTokenStore: base}

	u := &user.User{ID: uuid.New(), Email: "user@example.com", TOTPEnabled: true, TOTPSecret: "JBSWY3DPEHPK3PXP"}
	repo.add(u)
	base.mfaSessions["session-1"] = u.ID.String()

	code, err := totp.GenerateCode(u.TOTPSecret, time.Now())
	if err != nil {
		t.Fatalf("generate TOTP code: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"mfa_session_id": "session-1", "code": code})
	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/verify", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	NewTOTPHandlerWithDeps(repo, codes, store, &fakeJWT{}, &loginFakeMailer{}, "pepper", "braza-sso").Verify(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when StoreRefreshToken fails, got %d", rec.Code)
	}
}

// --- recovery error path tests ---

func TestRecovery_MalformedSessionUUID_Returns500(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()
	store.mfaSessions["session-1"] = "not-a-uuid"

	body, _ := json.Marshal(map[string]string{"mfa_session_id": "session-1", "recovery_code": "somecode"})
	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/recovery", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newTOTPHandler(repo, codes, store).Recovery(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for malformed session UUID, got %d", rec.Code)
	}
}

func TestRecovery_ListUnusedError_Returns500(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	base := newFakeCodes()
	codes := &errListUnusedCodes{fakeCodes: base}
	store := newFakeTOTPTokenStore()

	u := &user.User{ID: uuid.New(), Email: "user@example.com", TOTPEnabled: true}
	repo.add(u)
	store.mfaSessions["session-1"] = u.ID.String()

	body, _ := json.Marshal(map[string]string{"mfa_session_id": "session-1", "recovery_code": "somecode"})
	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/recovery", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	NewTOTPHandlerWithDeps(repo, codes, store, &fakeJWT{}, &loginFakeMailer{}, "pepper", "braza-sso").Recovery(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when ListUnused fails, got %d", rec.Code)
	}
}

func TestRecovery_NoMatchingCode_Returns401(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	u := &user.User{ID: uuid.New(), Email: "user@example.com", TOTPEnabled: true}
	repo.add(u)
	store.mfaSessions["session-1"] = u.ID.String()
	// No recovery codes stored — matchedCodeID will be nil
	_ = codes.ReplaceAll(u.ID, []string{})

	body, _ := json.Marshal(map[string]string{"mfa_session_id": "session-1", "recovery_code": "wrongcode"})
	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/recovery", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newTOTPHandler(repo, codes, store).Recovery(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when no matching recovery code, got %d", rec.Code)
	}
}

func TestRecovery_MarkUsedError_Returns500(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	base := newFakeCodes()
	codes := &errMarkUsedCodes{fakeCodes: base}
	store := newFakeTOTPTokenStore()

	u := &user.User{ID: uuid.New(), Email: "user@example.com", TOTPEnabled: true}
	repo.add(u)
	store.mfaSessions["session-1"] = u.ID.String()

	plainCode := "myrecoverycode1"
	hash, _ := HashPassword(plainCode, "pepper")
	_ = base.ReplaceAll(u.ID, []string{hash})

	body, _ := json.Marshal(map[string]string{"mfa_session_id": "session-1", "recovery_code": plainCode})
	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/recovery", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	NewTOTPHandlerWithDeps(repo, codes, store, &fakeJWT{}, &loginFakeMailer{}, "pepper", "braza-sso").Recovery(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when MarkUsed fails, got %d", rec.Code)
	}
}

func TestRecovery_FindByIDError_Returns500(t *testing.T) {
	base := newFakeTOTPUserRepo()
	codes := newFakeCodes()
	store := newFakeTOTPTokenStore()

	u := &user.User{ID: uuid.New(), Email: "user@example.com", TOTPEnabled: true}
	store.mfaSessions["session-1"] = u.ID.String()
	// User NOT added to repo — FindByID will fail

	plainCode := "myrecoverycode2"
	hash, _ := HashPassword(plainCode, "pepper")
	_ = codes.ReplaceAll(u.ID, []string{hash})

	body, _ := json.Marshal(map[string]string{"mfa_session_id": "session-1", "recovery_code": plainCode})
	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/recovery", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newTOTPHandler(base, codes, store).Recovery(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when FindByID fails after MarkUsed, got %d", rec.Code)
	}
}

func TestRecovery_JWTError_Returns500(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	base := newFakeCodes()
	store := newFakeTOTPTokenStore()

	u := &user.User{ID: uuid.New(), Email: "user@example.com", TOTPEnabled: true}
	repo.add(u)
	store.mfaSessions["session-1"] = u.ID.String()

	plainCode := "myrecoverycode3"
	hash, _ := HashPassword(plainCode, "pepper")
	_ = base.ReplaceAll(u.ID, []string{hash})

	body, _ := json.Marshal(map[string]string{"mfa_session_id": "session-1", "recovery_code": plainCode})
	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/recovery", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	NewTOTPHandlerWithDeps(repo, base, store, &errTOTPJWT{}, &loginFakeMailer{}, "pepper", "braza-sso").Recovery(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when JWT issuance fails in recovery, got %d", rec.Code)
	}
}

func TestRecovery_StoreRefreshError_Returns500(t *testing.T) {
	repo := newFakeTOTPUserRepo()
	base := newFakeCodes()
	baseStore := newFakeTOTPTokenStore()
	store := &errStoreTOTPTokenStore{fakeTOTPTokenStore: baseStore}

	u := &user.User{ID: uuid.New(), Email: "user@example.com", TOTPEnabled: true}
	repo.add(u)
	baseStore.mfaSessions["session-1"] = u.ID.String()

	plainCode := "myrecoverycode4"
	hash, _ := HashPassword(plainCode, "pepper")
	_ = base.ReplaceAll(u.ID, []string{hash})

	body, _ := json.Marshal(map[string]string{"mfa_session_id": "session-1", "recovery_code": plainCode})
	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/recovery", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	NewTOTPHandlerWithDeps(repo, base, store, &fakeJWT{}, &loginFakeMailer{}, "pepper", "braza-sso").Recovery(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when StoreRefreshToken fails in recovery, got %d", rec.Code)
	}
}
