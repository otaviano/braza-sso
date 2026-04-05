package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResetRequest_InvalidJSON_StillReturns200(t *testing.T) {
	repo := newFakePwdRepo()
	store := newFakePwdTokenStore()

	req := httptest.NewRequest(http.MethodPost, "/auth/password/reset-request", bytes.NewReader([]byte(`{bad json`)))
	rec := httptest.NewRecorder()

	newPwdResetHandler(repo, store).ResetRequest(rec, req)

	// Should return 200 silently even on bad JSON
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 even for invalid JSON body, got %d", rec.Code)
	}
}

func TestReset_InvalidJSON_Returns400(t *testing.T) {
	repo := newFakePwdRepo()
	store := newFakePwdTokenStore()

	req := httptest.NewRequest(http.MethodPost, "/auth/password/reset", bytes.NewReader([]byte(`{bad json`)))
	rec := httptest.NewRecorder()

	newPwdResetHandler(repo, store).Reset(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", rec.Code)
	}
}

func TestReset_MissingToken_Returns400(t *testing.T) {
	repo := newFakePwdRepo()
	store := newFakePwdTokenStore()

	body, _ := json.Marshal(map[string]string{"token": "", "password": "NewValidPass1!"})
	req := httptest.NewRequest(http.MethodPost, "/auth/password/reset", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newPwdResetHandler(repo, store).Reset(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when token is empty, got %d", rec.Code)
	}
}

func TestReset_MalformedUserIDInStore_Returns500(t *testing.T) {
	repo := newFakePwdRepo()
	store := newFakePwdTokenStore()

	// Manually place a token that resolves to a non-UUID value
	store.resetTokens["bad-id-token"] = "not-a-uuid"

	body, _ := json.Marshal(map[string]string{"token": "bad-id-token", "password": "NewValidPass1!"})
	req := httptest.NewRequest(http.MethodPost, "/auth/password/reset", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	newPwdResetHandler(repo, store).Reset(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for malformed stored user ID, got %d", rec.Code)
	}
}
