package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/otaviano/braza-sso/internal/email"
	"github.com/otaviano/braza-sso/internal/user"
	"github.com/pquerna/otp/totp"
)

const (
	totpIssuer          = "Braza SSO"
	recoveryCodeCount   = 8
	mfaSessionTTL       = 5 * time.Minute
)

// TOTPUserRepository is the user.Repository subset needed by TOTPHandler.
type TOTPUserRepository interface {
	FindByID(id uuid.UUID) (*user.User, error)
	UpdateTOTP(id uuid.UUID, secret string, enabled bool) error
}

// RecoveryCodeStore manages hashed recovery codes.
type RecoveryCodeStore interface {
	ReplaceAll(userID uuid.UUID, hashedCodes []string) error
	ListUnused(userID uuid.UUID) ([]struct {
		CodeID   uuid.UUID
		CodeHash string
	}, error)
	MarkUsed(userID, codeID uuid.UUID) error
}

// TOTPTokenStore is the TokenStore subset needed by TOTPHandler.
type TOTPTokenStore interface {
	ConsumeMFASession(ctx context.Context, token string) (string, error)
	StoreRefreshToken(ctx context.Context, token, userID string, ttl time.Duration) error
}

// TOTPTokenService is the JWT service subset needed by TOTPHandler.
type TOTPTokenService interface {
	IssueAccessToken(userID, email string, emailVerified bool, audience string) (string, error)
}

// TOTPHandler handles TOTP enrollment, confirmation, verification, and recovery.
type TOTPHandler struct {
	users     TOTPUserRepository
	codes     RecoveryCodeStore
	tokens    TOTPTokenStore
	jwt       TOTPTokenService
	mailer    email.Sender
	pepper    string
	issuer    string
	jwtIssuer string
}

func NewTOTPHandler(
	users *user.Repository,
	codes *user.RecoveryCodeRepository,
	tokens *TokenStore,
	jwt *TokenService,
	mailer email.Sender,
	pepper, jwtIssuer string,
) *TOTPHandler {
	return &TOTPHandler{
		users:     users,
		codes:     codes,
		tokens:    tokens,
		jwt:       jwt,
		mailer:    mailer,
		pepper:    pepper,
		issuer:    totpIssuer,
		jwtIssuer: jwtIssuer,
	}
}

// NewTOTPHandlerWithDeps creates a handler with injected dependencies (for testing).
func NewTOTPHandlerWithDeps(
	users TOTPUserRepository,
	codes RecoveryCodeStore,
	tokens TOTPTokenStore,
	jwt TOTPTokenService,
	mailer email.Sender,
	pepper, jwtIssuer string,
) *TOTPHandler {
	return &TOTPHandler{
		users:     users,
		codes:     codes,
		tokens:    tokens,
		jwt:       jwt,
		mailer:    mailer,
		pepper:    pepper,
		issuer:    totpIssuer,
		jwtIssuer: jwtIssuer,
	}
}

type enrollResponse struct {
	Secret        string   `json:"secret"`
	OTPURI        string   `json:"otp_uri"`
	RecoveryCodes []string `json:"recovery_codes"`
}

// Enroll handles POST /account/2fa/enroll.
// Requires a valid JWT (user_id injected by RequireAuth middleware).
func (h *TOTPHandler) Enroll(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := userIDFromCtx(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	u, err := h.users.FindByID(userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      h.issuer,
		AccountName: u.Email,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate TOTP key")
		return
	}

	// Store secret but don't enable yet — confirmed in /2fa/confirm
	if err := h.users.UpdateTOTP(userID, key.Secret(), false); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store TOTP secret")
		return
	}

	// Generate plaintext recovery codes
	plainCodes := make([]string, recoveryCodeCount)
	for i := range plainCodes {
		plainCodes[i] = randomToken(10)
	}

	// Hash and persist recovery codes
	hashed := make([]string, recoveryCodeCount)
	for i, c := range plainCodes {
		h, err := HashPassword(c, h.pepper)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to hash recovery codes")
			return
		}
		hashed[i] = h
	}
	if err := h.codes.ReplaceAll(userID, hashed); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store recovery codes")
		return
	}

	writeJSON(w, http.StatusOK, enrollResponse{
		Secret:        key.Secret(),
		OTPURI:        key.URL(),
		RecoveryCodes: plainCodes,
	})
}

type confirmRequest struct {
	Code string `json:"code"`
}

// Confirm handles POST /account/2fa/confirm.
func (h *TOTPHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := userIDFromCtx(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req confirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	u, err := h.users.FindByID(userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	if !totp.Validate(req.Code, u.TOTPSecret) {
		writeError(w, http.StatusUnauthorized, "invalid TOTP code")
		return
	}

	if err := h.users.UpdateTOTP(userID, u.TOTPSecret, true); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to enable 2FA")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "2FA enabled"})
}

type mfaVerifyRequest struct {
	MFASessionID string `json:"mfa_session_id"`
	Code         string `json:"code"`
}

// Verify handles POST /auth/2fa/verify — completes login after TOTP input.
func (h *TOTPHandler) Verify(w http.ResponseWriter, r *http.Request) {
	var req mfaVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.MFASessionID == "" || req.Code == "" {
		writeError(w, http.StatusBadRequest, "mfa_session_id and code are required")
		return
	}

	userIDStr, err := h.tokens.ConsumeMFASession(r.Context(), req.MFASessionID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "mfa session not found or expired")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "malformed session data")
		return
	}

	u, err := h.users.FindByID(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}

	if !totp.Validate(req.Code, u.TOTPSecret) {
		writeError(w, http.StatusUnauthorized, "invalid TOTP code")
		return
	}

	h.issueTokenPairForUser(w, r, u)
}

type recoveryRequest struct {
	MFASessionID string `json:"mfa_session_id"`
	RecoveryCode string `json:"recovery_code"`
}

// Recovery handles POST /auth/2fa/recovery — completes login using a recovery code.
func (h *TOTPHandler) Recovery(w http.ResponseWriter, r *http.Request) {
	var req recoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.MFASessionID == "" || req.RecoveryCode == "" {
		writeError(w, http.StatusBadRequest, "mfa_session_id and recovery_code are required")
		return
	}

	userIDStr, err := h.tokens.ConsumeMFASession(r.Context(), req.MFASessionID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "mfa session not found or expired")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "malformed session data")
		return
	}

	// Find matching unused recovery code
	unused, err := h.codes.ListUnused(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read recovery codes")
		return
	}

	var matchedCodeID *uuid.UUID
	for _, entry := range unused {
		if err := VerifyPassword(req.RecoveryCode, h.pepper, entry.CodeHash); err == nil {
			id := entry.CodeID
			matchedCodeID = &id
			break
		}
	}

	if matchedCodeID == nil {
		writeError(w, http.StatusUnauthorized, "invalid recovery code")
		return
	}

	if err := h.codes.MarkUsed(userID, *matchedCodeID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to invalidate recovery code")
		return
	}

	// Disable TOTP to prompt re-enrollment
	u, err := h.users.FindByID(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "user not found")
		return
	}
	h.users.UpdateTOTP(userID, u.TOTPSecret, false)

	resp := map[string]interface{}{
		"message":        "recovery code accepted; please re-enroll 2FA",
		"reenroll_required": true,
	}

	// Still issue tokens so the user can access the re-enrollment page
	accessToken, err := h.jwt.IssueAccessToken(userID.String(), u.Email, u.EmailVerified, h.jwtIssuer)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue access token")
		return
	}
	refreshToken := randomToken(32)
	h.tokens.StoreRefreshToken(r.Context(), refreshToken, userID.String(), refreshTokenTTL)

	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    refreshToken,
		Path:     "/auth/token/refresh",
		MaxAge:   int(refreshTokenTTL.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})

	resp["access_token"] = accessToken
	resp["token_type"] = "Bearer"
	writeJSON(w, http.StatusOK, resp)
}

// issueTokenPairForUser issues JWT + refresh token and sets the cookie.
func (h *TOTPHandler) issueTokenPairForUser(w http.ResponseWriter, r *http.Request, u *user.User) {
	accessToken, err := h.jwt.IssueAccessToken(u.ID.String(), u.Email, u.EmailVerified, h.jwtIssuer)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue access token")
		return
	}

	refreshToken := randomToken(32)
	h.tokens.StoreRefreshToken(r.Context(), refreshToken, u.ID.String(), refreshTokenTTL)

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
		ExpiresIn:   900,
	})
}

// userIDFromCtx extracts user_id injected by the RequireAuth middleware.
func userIDFromCtx(r *http.Request) (string, bool) {
	return UserIDFromContext(r.Context())
}
