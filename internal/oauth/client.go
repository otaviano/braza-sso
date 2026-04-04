package oauth

import (
	"errors"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

var ErrClientNotFound = errors.New("oauth client not found")
var ErrInvalidRedirectURI = errors.New("invalid redirect_uri")

// Client represents an OAuth2 application registered in Braza SSO.
type Client struct {
	ID                   string
	SecretHash           string
	RedirectURIs         []string
	Scopes               []string
	Name                 string
	LogoURL              string
	BackChannelLogoutURI string // optional; notified on user logout (OIDC back-channel logout)
}

// ClientRepository manages OAuth clients stored in Cassandra.
type ClientRepository struct {
	session *gocql.Session
}

func NewClientRepository(session *gocql.Session) *ClientRepository {
	return &ClientRepository{session: session}
}

func (r *ClientRepository) FindByID(clientID string) (*Client, error) {
	c := &Client{}
	err := r.session.Query(`
		SELECT client_id, client_secret_hash, redirect_uris, scopes, name, logo_url, backchannel_logout_uri
		FROM oauth_clients WHERE client_id = ? LIMIT 1`, clientID).
		Scan(&c.ID, &c.SecretHash, &c.RedirectURIs, &c.Scopes, &c.Name, &c.LogoURL, &c.BackChannelLogoutURI)
	if errors.Is(err, gocql.ErrNotFound) {
		return nil, ErrClientNotFound
	}
	return c, err
}

// FindByIDs returns all clients whose IDs are in the provided slice.
func (r *ClientRepository) FindByIDs(clientIDs []string) ([]*Client, error) {
	clients := make([]*Client, 0, len(clientIDs))
	for _, id := range clientIDs {
		c, err := r.FindByID(id)
		if err != nil {
			continue // skip unknown/deleted clients
		}
		clients = append(clients, c)
	}
	return clients, nil
}

// ConsentRepository manages user consent records in Cassandra.
type ConsentRepository struct {
	session *gocql.Session
}

func NewConsentRepository(session *gocql.Session) *ConsentRepository {
	return &ConsentRepository{session: session}
}

// HasConsent returns true if the user has previously consented to all requested scopes.
func (r *ConsentRepository) HasConsent(userID uuid.UUID, clientID string, scopes []string) (bool, error) {
	var grantedScopes []string
	err := r.session.Query(`
		SELECT scopes FROM user_consents WHERE user_id = ? AND client_id = ? LIMIT 1`,
		userID, clientID).Scan(&grantedScopes)
	if errors.Is(err, gocql.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	granted := make(map[string]bool)
	for _, s := range grantedScopes {
		granted[s] = true
	}
	for _, s := range scopes {
		if !granted[s] {
			return false, nil
		}
	}
	return true, nil
}

// StoreConsent persists the user's consent for the given client and scopes.
func (r *ConsentRepository) StoreConsent(userID uuid.UUID, clientID string, scopes []string) error {
	return r.session.Query(`
		INSERT INTO user_consents (user_id, client_id, scopes, granted_at) VALUES (?, ?, ?, ?)`,
		userID, clientID, scopes, time.Now().UTC()).Exec()
}

// ListConsentedClientIDs returns all client IDs the user has granted consent to.
func (r *ConsentRepository) ListConsentedClientIDs(userID uuid.UUID) ([]string, error) {
	iter := r.session.Query(`
		SELECT client_id FROM user_consents WHERE user_id = ?`, userID).Iter()
	var clientIDs []string
	var clientID string
	for iter.Scan(&clientID) {
		clientIDs = append(clientIDs, clientID)
	}
	return clientIDs, iter.Close()
}
