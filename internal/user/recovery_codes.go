package user

import (
	"errors"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
)

var ErrRecoveryCodeNotFound = errors.New("recovery code not found or already used")

// RecoveryCodeRepository manages hashed TOTP recovery codes in Cassandra.
type RecoveryCodeRepository struct {
	session *gocql.Session
}

func NewRecoveryCodeRepository(session *gocql.Session) *RecoveryCodeRepository {
	return &RecoveryCodeRepository{session: session}
}

// ReplaceAll deletes existing codes for a user and inserts the new hashed set.
func (r *RecoveryCodeRepository) ReplaceAll(userID uuid.UUID, hashedCodes []string) error {
	// Delete existing
	if err := r.session.Query(`DELETE FROM user_recovery_codes WHERE user_id = ?`, userID).Exec(); err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, code := range hashedCodes {
		codeID := uuid.New()
		if err := r.session.Query(`
			INSERT INTO user_recovery_codes (user_id, code_id, code_hash, used, created_at)
			VALUES (?, ?, ?, false, ?)`,
			userID, codeID, code, now).Exec(); err != nil {
			return err
		}
	}
	return nil
}

// ListUnused returns all unused code hashes for a user.
func (r *RecoveryCodeRepository) ListUnused(userID uuid.UUID) ([]struct {
	CodeID   uuid.UUID
	CodeHash string
}, error) {
	iter := r.session.Query(`
		SELECT code_id, code_hash FROM user_recovery_codes
		WHERE user_id = ? AND used = false ALLOW FILTERING`, userID).Iter()

	var results []struct {
		CodeID   uuid.UUID
		CodeHash string
	}
	var codeID uuid.UUID
	var codeHash string
	for iter.Scan(&codeID, &codeHash) {
		results = append(results, struct {
			CodeID   uuid.UUID
			CodeHash string
		}{codeID, codeHash})
	}
	return results, iter.Close()
}

// MarkUsed marks a specific recovery code as used.
func (r *RecoveryCodeRepository) MarkUsed(userID, codeID uuid.UUID) error {
	return r.session.Query(`
		UPDATE user_recovery_codes SET used = true WHERE user_id = ? AND code_id = ?`,
		userID, codeID).Exec()
}
