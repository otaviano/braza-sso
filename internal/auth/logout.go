package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/otaviano/braza-sso/internal/user"
)

// LogoutTokenStore is the TokenStore subset needed by LogoutHandler.
type LogoutTokenStore interface {
	ConsumeRefreshToken(ctx context.Context, token string) (string, error)
	RevokeAllUserSessions(ctx context.Context, userID string) error
}

// LogoutClientRepository is used to notify service providers on back-channel logout.
type LogoutClientRepository interface {
	FindByID(clientID string) (*user.User, error) // placeholder — SPs stored in oauth_clients
}

// LogoutHandler handles POST /auth/logout.
type LogoutHandler struct {
	tokens  LogoutTokenStore
	jwt     *TokenService
}

func NewLogoutHandler(tokens *TokenStore, jwt *TokenService) *LogoutHandler {
	return &LogoutHandler{tokens: tokens, jwt: jwt}
}

// Logout handles POST /auth/logout — deletes refresh token, clears cookie.
// Idempotent: always returns 200 even if session not found.
func (h *LogoutHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err == nil && cookie.Value != "" {
		// Best-effort consume — ignore errors (idempotent)
		userIDStr, err := h.tokens.ConsumeRefreshToken(r.Context(), cookie.Value)
		if err == nil {
			// Optionally send back-channel logout to registered SPs
			go h.backChannelLogout(r.Context(), userIDStr)
		}
	}

	// Clear cookie regardless
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/auth/token/refresh",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	w.WriteHeader(http.StatusOK)
}

// backChannelLogout sends a signed logout token to each SP's logout URI.
// This is a best-effort fire-and-forget operation.
func (h *LogoutHandler) backChannelLogout(ctx context.Context, userIDStr string) {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return
	}

	// Issue a logout token (JWT with event claim)
	logoutToken, err := h.jwt.IssueAccessToken(userID.String(), "", false, "logout")
	if err != nil {
		return
	}

	// In a full implementation, iterate registered SP logout URIs from the DB.
	// For now, this is the hook point — SPs would be looked up and notified here.
	_ = logoutToken
}

// RevokeAll handles POST /auth/logout/all — revokes all sessions for the authenticated user.
func (h *LogoutHandler) RevokeAll(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}
	h.tokens.RevokeAllUserSessions(r.Context(), userIDStr)
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/auth/token/refresh",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	w.WriteHeader(http.StatusOK)
}

// BackChannelLogoutReceiver handles POST /auth/backchannel-logout from external IdPs.
func BackChannelLogoutReceiver(tokens *TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			LogoutToken string `json:"logout_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// In production: verify the logout token signature, extract sub, revoke sessions.
		_ = body.LogoutToken
		w.WriteHeader(http.StatusOK)
	}
}

// sendLogoutToken is a helper used by back-channel logout to POST to an SP.
func sendLogoutToken(uri, token string) {
	body, _ := json.Marshal(map[string]string{"logout_token": token})
	client := &http.Client{Timeout: 5 * time.Second}
	client.Post(uri, "application/json", bytes.NewReader(body)) //nolint:errcheck
}
