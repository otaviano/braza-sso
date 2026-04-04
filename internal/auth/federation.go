package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/otaviano/braza-sso/internal/user"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	stateTokenTTL   = 10 * time.Minute
	prefixOAuthState = "oauth_state:"
)

// FederationStore manages short-lived OAuth state tokens in Redis.
type FederationStore interface {
	SetState(ctx context.Context, state, returnTo string) error
	ConsumeState(ctx context.Context, state string) (string, error)
	StoreRefreshToken(ctx context.Context, token, userID string, ttl time.Duration) error
}

// FederationUserRepository looks up and creates users for federated logins.
type FederationUserRepository interface {
	FindByEmail(email string) (*user.User, error)
	Create(u *user.User) error
}

// FederationIdentityRepository persists federated identity records.
type FederationIdentityRepository interface {
	Upsert(userID uuid.UUID, provider, providerUserID, email string) error
}

// FederationHandler handles Google OAuth2 federation.
type FederationHandler struct {
	googleCfg    *oauth2.Config
	users        FederationUserRepository
	identities   FederationIdentityRepository
	tokenStore   FederationStore
	jwt          LoginTokenService
	pepper       string
	jwtIssuer    string
}

func NewFederationHandler(
	clientID, clientSecret, baseURL string,
	users *user.Repository,
	identities *user.FederatedIdentityRepository,
	tokenStore *TokenStore,
	jwt *TokenService,
	pepper, jwtIssuer string,
) *FederationHandler {
	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  baseURL + "/auth/federation/google/callback",
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
	return &FederationHandler{
		googleCfg:  cfg,
		users:      users,
		identities: identities,
		tokenStore: tokenStore,
		jwt:        jwt,
		pepper:     pepper,
		jwtIssuer:  jwtIssuer,
	}
}

// NewFederationHandlerWithDeps creates a handler with injected dependencies (for testing).
func NewFederationHandlerWithDeps(
	googleCfg *oauth2.Config,
	users FederationUserRepository,
	identities FederationIdentityRepository,
	tokenStore FederationStore,
	jwt LoginTokenService,
	pepper, jwtIssuer string,
) *FederationHandler {
	return &FederationHandler{
		googleCfg:  googleCfg,
		users:      users,
		identities: identities,
		tokenStore: tokenStore,
		jwt:        jwt,
		pepper:     pepper,
		jwtIssuer:  jwtIssuer,
	}
}

// GoogleRedirect handles GET /auth/federation/google — redirects to Google.
func (h *FederationHandler) GoogleRedirect(w http.ResponseWriter, r *http.Request) {
	state := randomStateToken()
	returnTo := r.URL.Query().Get("return_to")
	h.tokenStore.SetState(r.Context(), state, returnTo)
	url := h.googleCfg.AuthCodeURL(state, oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusFound)
}

// GoogleCallback handles GET /auth/federation/google/callback.
func (h *FederationHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	if _, err := h.tokenStore.ConsumeState(r.Context(), state); err != nil {
		http.Error(w, `{"error":"invalid state"}`, http.StatusBadRequest)
		return
	}

	token, err := h.googleCfg.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, `{"error":"code exchange failed"}`, http.StatusBadRequest)
		return
	}

	info, err := fetchGoogleUserInfo(r.Context(), h.googleCfg, token)
	if err != nil {
		http.Error(w, `{"error":"failed to fetch user info"}`, http.StatusInternalServerError)
		return
	}

	// Link or create account
	u, err := h.users.FindByEmail(info.Email)
	if err != nil {
		// Auto-create account — no password (federated only)
		u = &user.User{
			ID:            uuid.New(),
			Email:         info.Email,
			EmailVerified: info.EmailVerified,
		}
		if err := h.users.Create(u); err != nil && err != user.ErrEmailTaken {
			http.Error(w, `{"error":"account creation failed"}`, http.StatusInternalServerError)
			return
		}
		if err == user.ErrEmailTaken {
			// Re-fetch
			u, _ = h.users.FindByEmail(info.Email)
		}
	}

	// Store federated identity record
	h.identities.Upsert(u.ID, "google", info.Sub, info.Email)

	// Issue tokens
	accessToken, err := h.jwt.IssueAccessToken(u.ID.String(), u.Email, u.EmailVerified, h.jwtIssuer)
	if err != nil {
		http.Error(w, `{"error":"token issuance failed"}`, http.StatusInternalServerError)
		return
	}

	refreshToken := randomToken(32)
	h.tokenStore.StoreRefreshToken(r.Context(), refreshToken, u.ID.String(), refreshTokenTTL)

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

type googleUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
}

func fetchGoogleUserInfo(ctx context.Context, cfg *oauth2.Config, token *oauth2.Token) (*googleUserInfo, error) {
	client := cfg.Client(ctx, token)
	resp, err := client.Get("https://openidconnect.googleapis.com/v1/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	if info.Email == "" {
		return nil, fmt.Errorf("empty email from Google")
	}
	return &info, nil
}

func randomStateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
