package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/otaviano/braza-sso/internal/user"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
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
	oidcVerifier *gooidc.IDTokenVerifier
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
) (*FederationHandler, error) {
	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  baseURL + "/auth/federation/google/callback",
		Scopes:       []string{gooidc.ScopeOpenID, "email", "profile"},
		Endpoint:     google.Endpoint,
	}

	provider, err := gooidc.NewProvider(context.Background(), "https://accounts.google.com")
	if err != nil {
		return nil, fmt.Errorf("creating OIDC provider: %w", err)
	}

	verifier := provider.Verifier(&gooidc.Config{ClientID: clientID})

	return &FederationHandler{
		googleCfg:    cfg,
		oidcVerifier: verifier,
		users:        users,
		identities:   identities,
		tokenStore:   tokenStore,
		jwt:          jwt,
		pepper:       pepper,
		jwtIssuer:    jwtIssuer,
	}, nil
}

// NewFederationHandlerWithDeps creates a handler with injected dependencies (for testing).
func NewFederationHandlerWithDeps(
	googleCfg *oauth2.Config,
	oidcVerifier *gooidc.IDTokenVerifier,
	users FederationUserRepository,
	identities FederationIdentityRepository,
	tokenStore FederationStore,
	jwt LoginTokenService,
	pepper, jwtIssuer string,
) *FederationHandler {
	return &FederationHandler{
		googleCfg:    googleCfg,
		oidcVerifier: oidcVerifier,
		users:        users,
		identities:   identities,
		tokenStore:   tokenStore,
		jwt:          jwt,
		pepper:       pepper,
		jwtIssuer:    jwtIssuer,
	}
}

// GoogleRedirect handles GET /auth/federation/google — redirects to Google.
func (h *FederationHandler) GoogleRedirect(w http.ResponseWriter, r *http.Request) {
	state := randomStateToken()
	returnTo := r.URL.Query().Get("return_to")
	if err := h.tokenStore.SetState(r.Context(), state, returnTo); err != nil {
		log.Warn().Err(err).Msg("federation: failed to store OAuth state token")
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	redirectURL := h.googleCfg.AuthCodeURL(state, oauth2.AccessTypeOnline)
	http.Redirect(w, r, redirectURL, http.StatusFound)
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

	// Verify the id_token JWT signature and claims before trusting any identity data.
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		http.Error(w, `{"error":"id_token missing from response"}`, http.StatusBadRequest)
		return
	}

	idToken, err := h.oidcVerifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		http.Error(w, `{"error":"id_token verification failed"}`, http.StatusUnauthorized)
		return
	}

	var claims struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		http.Error(w, `{"error":"failed to parse id_token claims"}`, http.StatusInternalServerError)
		return
	}

	if claims.Email == "" {
		http.Error(w, `{"error":"email missing from id_token"}`, http.StatusBadRequest)
		return
	}

	u, err := h.findOrCreateUser(r.Context(), claims.Sub, claims.Email, claims.EmailVerified)
	if err != nil {
		http.Error(w, `{"error":"account creation failed"}`, http.StatusInternalServerError)
		return
	}

	if err := h.identities.Upsert(u.ID, "google", claims.Sub, claims.Email); err != nil {
		log.Warn().Err(err).Str("user_id", u.ID.String()).Msg("federation: failed to upsert identity record")
	}

	accessToken, err := h.jwt.IssueAccessToken(u.ID.String(), u.Email, u.EmailVerified, h.jwtIssuer)
	if err != nil {
		http.Error(w, `{"error":"token issuance failed"}`, http.StatusInternalServerError)
		return
	}

	refreshToken := randomToken(32)
	if err := h.tokenStore.StoreRefreshToken(r.Context(), refreshToken, u.ID.String(), refreshTokenTTL); err != nil {
		http.Error(w, `{"error":"failed to store session"}`, http.StatusInternalServerError)
		return
	}
	setRefreshCookie(w, refreshToken)

	writeJSON(w, http.StatusOK, loginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   900,
	})
}

func (h *FederationHandler) findOrCreateUser(ctx context.Context, sub, email string, emailVerified bool) (*user.User, error) {
	existing, err := h.users.FindByEmail(email)
	if err == nil {
		return existing, nil
	}

	newUser := &user.User{
		ID:            uuid.New(),
		Email:         email,
		EmailVerified: emailVerified,
	}

	if createErr := h.users.Create(newUser); createErr != nil && createErr != user.ErrEmailTaken {
		return nil, createErr
	}

	// Race condition: another request created the user — re-fetch.
	if err == user.ErrEmailTaken {
		return h.users.FindByEmail(email)
	}

	return newUser, nil
}

func randomStateToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand.Read: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
