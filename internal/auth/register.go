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

const emailVerifyTTL = 24 * time.Hour

// UserRepository is the subset of user.Repository used by RegistrationHandler.
type UserRepository interface {
	Create(u *user.User) error
	FindByEmail(email string) (*user.User, error)
	SetEmailVerified(id uuid.UUID) error
}

// EmailVerificationStore is the subset of TokenStore used for email verification.
type EmailVerificationStore interface {
	CreateEmailVerificationToken(ctx context.Context, userID string, ttl time.Duration) (string, error)
	ConsumeEmailVerificationToken(ctx context.Context, token string) (string, error)
}

// RegistrationHandler handles POST /auth/register and related verification endpoints.
type RegistrationHandler struct {
	users   UserRepository
	tokens  EmailVerificationStore
	mailer  email.Sender
	pepper  string
	baseURL string
}

func NewRegistrationHandler(
	users *user.Repository,
	tokens *TokenStore,
	mailer email.Sender,
	pepper, baseURL string,
) *RegistrationHandler {
	return &RegistrationHandler{
		users:   users,
		tokens:  tokens,
		mailer:  mailer,
		pepper:  pepper,
		baseURL: baseURL,
	}
}

// NewRegistrationHandlerWithDeps creates a handler with injected dependencies (for testing).
func NewRegistrationHandlerWithDeps(
	users UserRepository,
	tokens EmailVerificationStore,
	mailer email.Sender,
	pepper, baseURL string,
) *RegistrationHandler {
	return &RegistrationHandler{
		users:   users,
		tokens:  tokens,
		mailer:  mailer,
		pepper:  pepper,
		baseURL: baseURL,
	}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register handles POST /auth/register.
func (h *RegistrationHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email is required")
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

	hash, err := HashPassword(req.Password, h.pepper)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to process password")
		return
	}

	u := &user.User{
		ID:            uuid.New(),
		Email:         req.Email,
		PasswordHash:  hash,
		EmailVerified: false,
	}

	if err := h.users.Create(u); err != nil {
		if err == user.ErrEmailTaken {
			// Silently return 201 to prevent email enumeration
			w.WriteHeader(http.StatusCreated)
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	token, err := h.tokens.CreateEmailVerificationToken(r.Context(), u.ID.String(), emailVerifyTTL)
	if err != nil {
		// Account created; email will be resent on demand
		w.WriteHeader(http.StatusCreated)
		return
	}

	verifyURL := fmt.Sprintf("%s/auth/verify-email?token=%s", h.baseURL, token)
	// Fire-and-forget; don't fail registration if email delivery fails
	go h.mailer.SendVerification(u.Email, verifyURL)

	w.WriteHeader(http.StatusCreated)
}

// VerifyEmail handles GET /auth/verify-email?token=<token>.
func (h *RegistrationHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	userIDStr, err := h.tokens.ConsumeEmailVerificationToken(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusBadRequest, "token not found or expired")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invalid stored user id")
		return
	}

	if err := h.users.SetEmailVerified(userID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify email")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "email verified"})
}

type resendRequest struct {
	Email string `json:"email"`
}

// ResendVerification handles POST /auth/resend-verification.
func (h *RegistrationHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var req resendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Always return 200 to prevent email enumeration
	u, err := h.users.FindByEmail(req.Email)
	if err != nil || u.EmailVerified {
		w.WriteHeader(http.StatusOK)
		return
	}

	token, err := h.tokens.CreateEmailVerificationToken(r.Context(), u.ID.String(), emailVerifyTTL)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	verifyURL := fmt.Sprintf("%s/auth/verify-email?token=%s", h.baseURL, token)
	go h.mailer.SendVerification(u.Email, verifyURL)

	w.WriteHeader(http.StatusOK)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// writeJSON encodes v as JSON and writes it with the given status.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
