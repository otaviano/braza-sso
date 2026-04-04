package user

import (
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

// FederatedIdentityRepository manages federated identity records in Cassandra.
type FederatedIdentityRepository struct {
	session *gocql.Session
}

func NewFederatedIdentityRepository(session *gocql.Session) *FederatedIdentityRepository {
	return &FederatedIdentityRepository{session: session}
}

// Upsert inserts or replaces a federated identity record.
func (r *FederatedIdentityRepository) Upsert(userID uuid.UUID, provider, providerUserID, email string) error {
	return r.session.Query(`
		INSERT INTO federated_identities (user_id, provider, provider_user_id, email, linked_at)
		VALUES (?, ?, ?, ?, ?)`,
		userID, provider, providerUserID, email, time.Now().UTC()).Exec()
}
