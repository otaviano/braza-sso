package auth

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

// LogoutTokenStore is the TokenStore subset needed by LogoutHandler.
type LogoutTokenStore interface {
	ConsumeRefreshToken(ctx context.Context, token string) (string, error)
	RevokeAllUserSessions(ctx context.Context, userID string) error
}

// BackChannelNotifier sends logout notifications to registered relying parties.
// Implementations must be safe to call from a goroutine.
type BackChannelNotifier interface {
	NotifyLogout(ctx context.Context, userID string, logoutToken string) error
}

// LogoutHandler handles POST /auth/logout.
type LogoutHandler struct {
	tokens   LogoutTokenStore
	jwt      *TokenService
	notifier BackChannelNotifier // optional; nil disables back-channel notifications
}

func NewLogoutHandler(tokens *TokenStore, jwt *TokenService, notifier BackChannelNotifier) *LogoutHandler {
	return &LogoutHandler{tokens: tokens, jwt: jwt, notifier: notifier}
}

// Logout handles POST /auth/logout — deletes refresh token, clears cookie.
// Idempotent: always returns 200 even if session not found.
func (h *LogoutHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err == nil && cookie.Value != "" {
		userIDStr, err := h.tokens.ConsumeRefreshToken(r.Context(), cookie.Value)
		if err == nil && h.notifier != nil {
			go h.backChannelLogout(r.Context(), userIDStr)
		}
	}

	clearRefreshCookie(w)
	w.WriteHeader(http.StatusOK)
}

// backChannelLogout issues a signed logout token and notifies all relying parties.
func (h *LogoutHandler) backChannelLogout(ctx context.Context, userIDStr string) {
	logoutToken, err := h.jwt.IssueAccessToken(userIDStr, "", false, "logout")
	if err != nil {
		log.Warn().Err(err).Str("user_id", userIDStr).Msg("back-channel logout: failed to issue logout token")
		return
	}

	if err := h.notifier.NotifyLogout(ctx, userIDStr, logoutToken); err != nil {
		log.Warn().Err(err).Str("user_id", userIDStr).Msg("back-channel logout: notification failed")
	}
}

// RevokeAll handles POST /auth/logout/all — revokes all sessions for the authenticated user.
func (h *LogoutHandler) RevokeAll(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := UserIDFromContext(r.Context())
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}
	h.tokens.RevokeAllUserSessions(r.Context(), userIDStr)
	clearRefreshCookie(w)
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
