// Package user defines the User domain model and its Cassandra-backed repository.
package user

import (
	"errors"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var ErrNotFound = errors.New("user not found")
var ErrEmailTaken = errors.New("email already registered")

// Repository handles user persistence in Cassandra.
type Repository struct {
	session *gocql.Session
}

func NewRepository(session *gocql.Session) *Repository {
	return &Repository{session: session}
}

// Create inserts a new user. Returns ErrEmailTaken if email exists.
// It also writes to the users_by_email lookup table to avoid ALLOW FILTERING.
func (r *Repository) Create(u *User) error {
	// Lightweight duplicate check via the lookup table (O(1) partition key read).
	existing, err := r.FindByEmail(u.Email)
	if err == nil && existing != nil {
		return ErrEmailTaken
	}

	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now

	if err := r.session.Query(`
		INSERT INTO users
		  (user_id, email, password_hash, email_verified, totp_enabled, totp_secret,
		   locked_until, failed_attempts, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		gocql.UUID(u.ID), u.Email, u.PasswordHash, u.EmailVerified,
		u.TOTPEnabled, u.TOTPSecret, u.LockedUntil, u.FailedAttempts,
		u.CreatedAt, u.UpdatedAt,
	).Exec(); err != nil {
		log.Error().Err(err).Msg("insert into users failed")
		return err
	}

	if err := r.session.Query(`
		INSERT INTO users_by_email (email, user_id) VALUES (?, ?)`,
		u.Email, gocql.UUID(u.ID),
	).Exec(); err != nil {
		log.Error().Err(err).Msg("insert into users_by_email failed")
		return err
	}

	return nil
}

// FindByID retrieves a user by UUID.
func (r *Repository) FindByID(id uuid.UUID) (*User, error) {
	u := &User{}
	var gid gocql.UUID
	err := r.session.Query(`
		SELECT user_id, email, password_hash, email_verified, totp_enabled, totp_secret,
		       locked_until, failed_attempts, created_at, updated_at
		FROM users WHERE user_id = ? LIMIT 1`, gocql.UUID(id)).
		Scan(&gid, &u.Email, &u.PasswordHash, &u.EmailVerified,
			&u.TOTPEnabled, &u.TOTPSecret, &u.LockedUntil, &u.FailedAttempts,
			&u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, gocql.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.ID = uuid.UUID(gid)
	return u, nil
}

// FindByEmail retrieves a user by email using the users_by_email lookup table.
// This avoids ALLOW FILTERING by resolving the email to a user_id first (O(1) partition read),
// then fetching the user by primary key.
func (r *Repository) FindByEmail(email string) (*User, error) {
	var gid gocql.UUID
	err := r.session.Query(`
		SELECT user_id FROM users_by_email WHERE email = ?`, email).
		Scan(&gid)
	if errors.Is(err, gocql.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.FindByID(uuid.UUID(gid))
}

// SetEmailVerified marks the user's email as verified.
func (r *Repository) SetEmailVerified(id uuid.UUID) error {
	return r.session.Query(`
		UPDATE users SET email_verified = true, updated_at = ? WHERE user_id = ?`,
		time.Now().UTC(), gocql.UUID(id)).Exec()
}

// UpdateFailedAttempts sets the failed login counter and lock timestamp.
func (r *Repository) UpdateFailedAttempts(id uuid.UUID, attempts int, lockedUntil *time.Time) error {
	return r.session.Query(`
		UPDATE users SET failed_attempts = ?, locked_until = ?, updated_at = ? WHERE user_id = ?`,
		attempts, lockedUntil, time.Now().UTC(), gocql.UUID(id)).Exec()
}

// UpdateFailedAttemptsReset clears the failed login counter and lock.
func (r *Repository) UpdateFailedAttemptsReset(id uuid.UUID) error {
	return r.session.Query(`
		UPDATE users SET failed_attempts = 0, locked_until = null, updated_at = ? WHERE user_id = ?`,
		time.Now().UTC(), gocql.UUID(id)).Exec()
}

// UpdatePassword replaces the password hash and resets failed attempts.
func (r *Repository) UpdatePassword(id uuid.UUID, hash string) error {
	return r.session.Query(`
		UPDATE users SET password_hash = ?, failed_attempts = 0, locked_until = null, updated_at = ?
		WHERE user_id = ?`,
		hash, time.Now().UTC(), gocql.UUID(id)).Exec()
}

// UpdateTOTP stores the TOTP secret and enabled flag.
func (r *Repository) UpdateTOTP(id uuid.UUID, secret string, enabled bool) error {
	return r.session.Query(`
		UPDATE users SET totp_secret = ?, totp_enabled = ?, updated_at = ? WHERE user_id = ?`,
		secret, enabled, time.Now().UTC(), gocql.UUID(id)).Exec()
}
