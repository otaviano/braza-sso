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

const (
	refreshCookieName = "refresh_token"
	refreshTokenTTL   = 7 * 24 * time.Hour
)

// LoginRepository is the subset of user.Repository used by LoginHandler.
type LoginRepository interface {
	FindByEmail(email string) (*user.User, error)
	UpdateFailedAttempts(id uuid.UUID, attempts int, lockedUntil *time.Time) error
	UpdateFailedAttemptsReset(id uuid.UUID) error
}

// LoginTokenStore is the subset of TokenStore used by LoginHandler.
type LoginTokenStore interface {
	IncrLoginAttempts(ctx context.Context, userID string) (int64, error)
	ResetLoginAttempts(ctx context.Context, userID string) error
	StoreRefreshToken(ctx context.Context, token, userID string, ttl time.Duration) error
	ConsumeRefreshToken(ctx context.Context, token string) (string, error)
	RevokeAllUserSessions(ctx context.Context, userID string) error
	StoreMFASession(ctx context.Context, token, userID string, ttl time.Duration) error
}

// LoginTokenService is the subset of TokenService used by LoginHandler.
type LoginTokenService interface {
	IssueAccessToken(userID, email string, emailVerified bool, audience string) (string, error)
}

// LoginHandler handles POST /auth/login and POST /auth/token/refresh.
type LoginHandler struct {
	users   LoginRepository
	tokens  LoginTokenStore
	jwt     LoginTokenService
	mailer  email.Sender
	pepper  string
	baseURL string
	issuer  string
}

func NewLoginHandler(
	users *user.Repository,
	tokens *TokenStore,
	jwt *TokenService,
	mailer email.Sender,
	pepper, baseURL, issuer string,
) *LoginHandler {
	return &LoginHandler{
		users:   users,
		tokens:  tokens,
		jwt:     jwt,
		mailer:  mailer,
		pepper:  pepper,
		baseURL: baseURL,
		issuer:  issuer,
	}
}

// NewLoginHandlerWithDeps creates a handler with injected dependencies (for testing).
func NewLoginHandlerWithDeps(
	users LoginRepository,
	tokens LoginTokenStore,
	jwt LoginTokenService,
	mailer email.Sender,
	pepper, baseURL, issuer string,
) *LoginHandler {
	return &LoginHandler{
		users:   users,
		tokens:  tokens,
		jwt:     jwt,
		mailer:  mailer,
		pepper:  pepper,
		baseURL: baseURL,
		issuer:  issuer,
	}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type mfaRequiredResponse struct {
	MFARequired  bool   `json:"mfa_required"`
	MFASessionID string `json:"mfa_session_id"`
}

// Login handles POST /auth/login.
func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	u, err := h.users.FindByEmail(req.Email)
	if err != nil {
		// Generic error — don't reveal whether email exists
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Check account lockout
	if u.LockedUntil != nil && time.Now().Before(*u.LockedUntil) {
		writeError(w, http.StatusForbidden, "account is temporarily locked")
		return
	}

	// Verify password
	if err := VerifyPassword(req.Password, h.pepper, u.PasswordHash); err != nil {
		h.handleFailedAttempt(r.Context(), u)
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Successful authentication — reset failed attempts
	h.tokens.ResetLoginAttempts(r.Context(), u.ID.String())
	if u.FailedAttempts > 0 {
		h.users.UpdateFailedAttemptsReset(u.ID)
	}

	// If MFA is enabled, issue an intermediate session token instead of tokens
	if u.TOTPEnabled {
		mfaToken := randomToken(32)
		if err := h.tokens.StoreMFASession(r.Context(), mfaToken, u.ID.String(), 5*time.Minute); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create mfa session")
			return
		}
		writeJSON(w, http.StatusOK, mfaRequiredResponse{
			MFARequired:  true,
			MFASessionID: mfaToken,
		})
		return
	}

	h.issueTokenPair(w, r, u)
}

// Refresh handles POST /auth/token/refresh.
func (h *LoginHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "refresh token missing")
		return
	}

	userIDStr, err := h.tokens.ConsumeRefreshToken(r.Context(), cookie.Value)
	if err != nil {
		// Token not found — possible reuse attack: revoke all sessions
		// We can't know the userID here, so just return 401
		writeError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "malformed session data")
		return
	}

	// Look up user to get current claims
	u, err := h.findUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}

	h.issueTokenPair(w, r, u)
}

// issueTokenPair issues a JWT access token and rotates the refresh token cookie.
func (h *LoginHandler) issueTokenPair(w http.ResponseWriter, r *http.Request, u *user.User) {
	accessToken, err := h.jwt.IssueAccessToken(u.ID.String(), u.Email, u.EmailVerified, h.issuer)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue access token")
		return
	}

	refreshToken := randomToken(32)
	if err := h.tokens.StoreRefreshToken(r.Context(), refreshToken, u.ID.String(), refreshTokenTTL); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store refresh token")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    refreshToken,
		Path:     "/auth/token/refresh",
		MaxAge:   int(refreshTokenTTL.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	writeJSON(w, http.StatusOK, loginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   900, // 15 minutes in seconds
	})
}

// handleFailedAttempt increments the failure counter and locks the account if threshold reached.
func (h *LoginHandler) handleFailedAttempt(ctx context.Context, u *user.User) {
	count, err := h.tokens.IncrLoginAttempts(ctx, u.ID.String())
	if err != nil {
		return
	}

	newAttempts := int(count)
	var lockedUntil *time.Time

	if count >= int64(MaxLoginAttempts()) {
		t := time.Now().Add(LockoutDuration())
		lockedUntil = &t
		unlockURL := fmt.Sprintf("%s/auth/unlock", h.baseURL)
		go h.mailer.SendAccountLocked(u.Email, unlockURL)
	}

	h.users.UpdateFailedAttempts(u.ID, newAttempts, lockedUntil)
}

// findUserByID is a helper to look up a user by UUID. LoginRepository uses email lookups;
// we need a FindByID for the refresh flow. We store it as an optional interface upgrade.
func (h *LoginHandler) findUserByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	type byID interface {
		FindByID(id uuid.UUID) (*user.User, error)
	}
	if repo, ok := h.users.(byID); ok {
		return repo.FindByID(id)
	}
	return nil, fmt.Errorf("FindByID not supported by this repository")
}
