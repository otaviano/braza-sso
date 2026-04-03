package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/otaviano/braza-sso/internal/email"
	"github.com/otaviano/braza-sso/internal/user"
)

const passwordResetTTL = 1 * time.Hour

// PasswordResetRepository is the user.Repository subset needed for password reset.
type PasswordResetRepository interface {
	FindByEmail(email string) (*user.User, error)
	UpdatePassword(id uuid.UUID, hash string) error
}

// PasswordResetTokenStore is the TokenStore subset needed for password reset.
type PasswordResetTokenStore interface {
	CreatePasswordResetToken(ctx context.Context, userID string, ttl time.Duration) (string, error)
	ConsumePasswordResetToken(ctx context.Context, token string) (string, error)
	RevokeAllUserSessions(ctx context.Context, userID string) error
}

// PasswordResetHandler handles POST /auth/password/reset-request and POST /auth/password/reset.
type PasswordResetHandler struct {
	users   PasswordResetRepository
	tokens  PasswordResetTokenStore
	mailer  email.Sender
	pepper  string
	baseURL string
}

func NewPasswordResetHandler(
	users *user.Repository,
	tokens *TokenStore,
	mailer email.Sender,
	pepper, baseURL string,
) *PasswordResetHandler {
	return &PasswordResetHandler{
		users:   users,
		tokens:  tokens,
		mailer:  mailer,
		pepper:  pepper,
		baseURL: baseURL,
	}
}

// NewPasswordResetHandlerWithDeps creates a handler with injected dependencies (for testing).
func NewPasswordResetHandlerWithDeps(
	users PasswordResetRepository,
	tokens PasswordResetTokenStore,
	mailer email.Sender,
	pepper, baseURL string,
) *PasswordResetHandler {
	return &PasswordResetHandler{
		users:   users,
		tokens:  tokens,
		mailer:  mailer,
		pepper:  pepper,
		baseURL: baseURL,
	}
}

type resetRequestBody struct {
	Email string `json:"email"`
}

// ResetRequest handles POST /auth/password/reset-request.
// Always returns 200 to prevent email enumeration.
func (h *PasswordResetHandler) ResetRequest(w http.ResponseWriter, r *http.Request) {
	var req resetRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	u, err := h.users.FindByEmail(req.Email)
	if err != nil {
		// User not found — return 200 silently
		w.WriteHeader(http.StatusOK)
		return
	}

	token, err := h.tokens.CreatePasswordResetToken(r.Context(), u.ID.String(), passwordResetTTL)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	resetURL := fmt.Sprintf("%s/auth/password/reset?token=%s", h.baseURL, token)
	go h.mailer.SendPasswordReset(u.Email, resetURL)

	w.WriteHeader(http.StatusOK)
}

type resetBody struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// Reset handles POST /auth/password/reset.
func (h *PasswordResetHandler) Reset(w http.ResponseWriter, r *http.Request) {
	var req resetBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	violations := ValidatePasswordPolicy(req.Password)
	if len(violations) > 0 {
		msgs := make([]string, len(violations))
		for i, v := range violations {
			msgs[i] = v.Message
		}
		writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error":      "password does not meet policy requirements",
			"violations": msgs,
		})
		return
	}

	userIDStr, err := h.tokens.ConsumePasswordResetToken(r.Context(), req.Token)
	if err != nil {
		writeError(w, http.StatusBadRequest, "token not found or expired")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invalid stored user id")
		return
	}

	hash, err := HashPassword(req.Password, h.pepper)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	if err := h.users.UpdatePassword(userID, hash); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update password")
		return
	}

	// Invalidate all active sessions so stolen refresh tokens are revoked
	h.tokens.RevokeAllUserSessions(r.Context(), userIDStr)

	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}
