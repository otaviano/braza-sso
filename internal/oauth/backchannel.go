// Package oauth provides OAuth2/OIDC authorization server handlers and repositories.
package oauth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// BackChannelLogoutService implements auth.BackChannelNotifier.
// It finds all relying parties that hold user consent and POSTs a logout token to each.
type BackChannelLogoutService struct {
	clients  *ClientRepository
	consents *ConsentRepository
}

// NewBackChannelLogoutService constructs a BackChannelLogoutService.
func NewBackChannelLogoutService(clients *ClientRepository, consents *ConsentRepository) *BackChannelLogoutService {
	return &BackChannelLogoutService{clients: clients, consents: consents}
}

// NotifyLogout implements auth.BackChannelNotifier.
// It is called in a goroutine and logs but does not propagate errors.
func (s *BackChannelLogoutService) NotifyLogout(ctx context.Context, userIDStr string, logoutToken string) error {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return err
	}

	clientIDs, err := s.consents.ListConsentedClientIDs(userID)
	if err != nil {
		return err
	}

	clients, err := s.clients.FindByIDs(clientIDs)
	if err != nil {
		return err
	}

	for _, client := range clients {
		if client.BackChannelLogoutURI == "" {
			continue
		}
		go s.postLogoutToken(client.BackChannelLogoutURI, logoutToken)
	}

	return nil
}

func (s *BackChannelLogoutService) postLogoutToken(uri, logoutToken string) {
	body, err := json.Marshal(map[string]string{"logout_token": logoutToken})
	if err != nil {
		log.Warn().Err(err).Str("uri", uri).Msg("back-channel logout: failed to marshal token")
		return
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Post(uri, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Warn().Err(err).Str("uri", uri).Msg("back-channel logout: POST failed")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Warn().
			Str("uri", uri).
			Int("status", resp.StatusCode).
			Msg("back-channel logout: relying party returned error")
	}
}
