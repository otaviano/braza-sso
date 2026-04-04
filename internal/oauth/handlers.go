package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/otaviano/braza-sso/internal/auth"
	"github.com/otaviano/braza-sso/internal/user"
	"github.com/redis/go-redis/v9"
)

const authCodeTTL = 60 * time.Second

// OAuthTokenService is the JWT service used by OAuth handlers.
type OAuthTokenService interface {
	IssueAccessToken(userID, email string, emailVerified bool, audience string) (string, error)
}

// OAuthHandlers groups all OAuth2/OIDC endpoint handlers.
type OAuthHandlers struct {
	clients   *ClientRepository
	consents  *ConsentRepository
	users     *user.Repository
	redis     *redis.Client
	jwt       OAuthTokenService
	issuer    string
	baseURL   string
	tokenSvc  *auth.TokenService
}

func NewOAuthHandlers(
	clients *ClientRepository,
	consents *ConsentRepository,
	users *user.Repository,
	redis *redis.Client,
	jwt *auth.TokenService,
	issuer, baseURL string,
) *OAuthHandlers {
	return &OAuthHandlers{
		clients:  clients,
		consents: consents,
		users:    users,
		redis:    redis,
		jwt:      jwt,
		issuer:   issuer,
		baseURL:  baseURL,
		tokenSvc: jwt,
	}
}

// Authorize handles GET /oauth/authorize.
func (h *OAuthHandlers) Authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	responseType := q.Get("response_type")
	scopeStr := q.Get("scope")
	state := q.Get("state")

	if responseType != "code" {
		http.Error(w, "unsupported_response_type", http.StatusBadRequest)
		return
	}

	client, err := h.clients.FindByID(clientID)
	if err != nil {
		http.Error(w, "invalid_client", http.StatusBadRequest)
		return
	}

	if !validRedirectURI(client.RedirectURIs, redirectURI) {
		http.Error(w, "invalid_redirect_uri", http.StatusBadRequest)
		return
	}

	scopes := strings.Fields(scopeStr)

	// Check if user is authenticated via JWT cookie/header
	userIDStr, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		// Not logged in — redirect to login page, passing original authorize params
		loginURL := fmt.Sprintf("%s/login?%s", h.baseURL, r.URL.RawQuery)
		http.Redirect(w, r, loginURL, http.StatusFound)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "invalid_session", http.StatusInternalServerError)
		return
	}

	// Check consent
	hasConsent, _ := h.consents.HasConsent(userID, clientID, scopes)
	if !hasConsent {
		// Redirect to consent page
		consentURL := fmt.Sprintf("%s/consent?%s", h.baseURL, r.URL.RawQuery)
		http.Redirect(w, r, consentURL, http.StatusFound)
		return
	}

	// Issue authorization code
	code, err := h.issueAuthCode(r.Context(), userIDStr, clientID, redirectURI, scopes)
	if err != nil {
		http.Error(w, "server_error", http.StatusInternalServerError)
		return
	}

	redirectTo, _ := url.Parse(redirectURI)
	q2 := redirectTo.Query()
	q2.Set("code", code)
	if state != "" {
		q2.Set("state", state)
	}
	redirectTo.RawQuery = q2.Encode()
	http.Redirect(w, r, redirectTo.String(), http.StatusFound)
}

// Token handles POST /oauth/token.
func (h *OAuthHandlers) Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		oauthError(w, "invalid_request", http.StatusBadRequest)
		return
	}

	grantType := r.FormValue("grant_type")

	switch grantType {
	case "authorization_code":
		h.handleAuthCode(w, r)
	case "client_credentials":
		h.handleClientCredentials(w, r)
	default:
		oauthError(w, "unsupported_grant_type", http.StatusBadRequest)
	}
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	IDToken     string `json:"id_token,omitempty"`
}

func (h *OAuthHandlers) handleAuthCode(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	redirectURI := r.FormValue("redirect_uri")
	clientID := r.FormValue("client_id")

	// Consume the auth code from Redis
	codeKey := "oauth_code:" + code
	val, err := h.redis.GetDel(r.Context(), codeKey).Result()
	if err != nil {
		oauthError(w, "invalid_grant", http.StatusBadRequest)
		return
	}

	// val is JSON: {user_id, client_id, redirect_uri, scopes}
	var meta authCodeMeta
	if err := json.Unmarshal([]byte(val), &meta); err != nil {
		oauthError(w, "server_error", http.StatusInternalServerError)
		return
	}

	if meta.ClientID != clientID || meta.RedirectURI != redirectURI {
		oauthError(w, "invalid_grant", http.StatusBadRequest)
		return
	}

	u, err := h.users.FindByID(uuid.MustParse(meta.UserID))
	if err != nil {
		oauthError(w, "invalid_grant", http.StatusBadRequest)
		return
	}

	accessToken, err := h.jwt.IssueAccessToken(u.ID.String(), u.Email, u.EmailVerified, clientID)
	if err != nil {
		oauthError(w, "server_error", http.StatusInternalServerError)
		return
	}

	// ID token (same as access token for simplicity — extends Claims)
	idToken, _ := h.jwt.IssueAccessToken(u.ID.String(), u.Email, u.EmailVerified, clientID)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(tokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   900,
		IDToken:     idToken,
	})
}

func (h *OAuthHandlers) handleClientCredentials(w http.ResponseWriter, r *http.Request) {
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")

	client, err := h.clients.FindByID(clientID)
	if err != nil {
		oauthError(w, "invalid_client", http.StatusUnauthorized)
		return
	}

	if err := auth.VerifyPassword(clientSecret, "", client.SecretHash); err != nil {
		oauthError(w, "invalid_client", http.StatusUnauthorized)
		return
	}

	accessToken, err := h.jwt.IssueAccessToken(clientID, "", false, h.issuer)
	if err != nil {
		oauthError(w, "server_error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(tokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   900,
	})
}

// Userinfo handles GET /oauth/userinfo.
func (h *OAuthHandlers) Userinfo(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	userID, _ := uuid.Parse(userIDStr)
	u, err := h.users.FindByID(userID)
	if err != nil {
		http.Error(w, `{"error":"user_not_found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sub":            u.ID.String(),
		"email":          u.Email,
		"email_verified": u.EmailVerified,
	})
}

// Discovery handles GET /.well-known/openid-configuration.
func (h *OAuthHandlers) Discovery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"issuer":                                h.issuer,
		"authorization_endpoint":                h.baseURL + "/oauth/authorize",
		"token_endpoint":                        h.baseURL + "/oauth/token",
		"userinfo_endpoint":                     h.baseURL + "/oauth/userinfo",
		"jwks_uri":                              h.baseURL + "/oauth/jwks.json",
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "email", "profile"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post"},
		"claims_supported":                      []string{"sub", "email", "email_verified", "iss", "aud", "exp", "iat"},
	})
}

// Consent handles POST /oauth/consent — stores user consent and redirects back.
func (h *OAuthHandlers) Consent(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}

	userIDStr, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID, _ := uuid.Parse(userIDStr)
	clientID := r.FormValue("client_id")
	redirectURI := r.FormValue("redirect_uri")
	scopeStr := r.FormValue("scope")
	state := r.FormValue("state")
	scopes := strings.Fields(scopeStr)

	h.consents.StoreConsent(userID, clientID, scopes)

	// Issue auth code and redirect
	code, err := h.issueAuthCode(r.Context(), userIDStr, clientID, redirectURI, scopes)
	if err != nil {
		http.Error(w, "server_error", http.StatusInternalServerError)
		return
	}

	redirectTo, _ := url.Parse(redirectURI)
	q := redirectTo.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	redirectTo.RawQuery = q.Encode()
	http.Redirect(w, r, redirectTo.String(), http.StatusFound)
}

type authCodeMeta struct {
	UserID      string   `json:"user_id"`
	ClientID    string   `json:"client_id"`
	RedirectURI string   `json:"redirect_uri"`
	Scopes      []string `json:"scopes"`
}

func (h *OAuthHandlers) issueAuthCode(ctx context.Context, userID, clientID, redirectURI string, scopes []string) (string, error) {
	code := randomCode()
	meta := authCodeMeta{UserID: userID, ClientID: clientID, RedirectURI: redirectURI, Scopes: scopes}
	data, err := json.Marshal(meta)
	if err != nil {
		return "", err
	}
	return code, h.redis.Set(ctx, "oauth_code:"+code, string(data), authCodeTTL).Err()
}

func randomCode() string {
	b := make([]byte, 24)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func validRedirectURI(allowed []string, uri string) bool {
	for _, a := range allowed {
		if a == uri {
			return true
		}
	}
	return false
}

func oauthError(w http.ResponseWriter, errCode string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": errCode})
}
