// Package user defines the User domain model and its Cassandra-backed repository.
package user

import (
	"errors"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
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
//
// Required schema (apply once before deploying):
//
//	CREATE TABLE IF NOT EXISTS users_by_email (
//	    email   TEXT PRIMARY KEY,
//	    user_id UUID
//	);
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
		u.ID, u.Email, u.PasswordHash, u.EmailVerified,
		u.TOTPEnabled, u.TOTPSecret, u.LockedUntil, u.FailedAttempts,
		u.CreatedAt, u.UpdatedAt,
	).Exec(); err != nil {
		return err
	}

	return r.session.Query(`
		INSERT INTO users_by_email (email, user_id) VALUES (?, ?)`,
		u.Email, u.ID,
	).Exec()
}

// FindByID retrieves a user by UUID.
func (r *Repository) FindByID(id uuid.UUID) (*User, error) {
	u := &User{}
	err := r.session.Query(`
		SELECT user_id, email, password_hash, email_verified, totp_enabled, totp_secret,
		       locked_until, failed_attempts, created_at, updated_at
		FROM users WHERE user_id = ? LIMIT 1`, id).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.EmailVerified,
			&u.TOTPEnabled, &u.TOTPSecret, &u.LockedUntil, &u.FailedAttempts,
			&u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, gocql.ErrNotFound) {
		return nil, ErrNotFound
	}
	return u, err
}

// FindByEmail retrieves a user by email using the users_by_email lookup table.
// This avoids ALLOW FILTERING by resolving the email to a user_id first (O(1) partition read),
// then fetching the user by primary key.
func (r *Repository) FindByEmail(email string) (*User, error) {
	var userID uuid.UUID
	err := r.session.Query(`
		SELECT user_id FROM users_by_email WHERE email = ?`, email).
		Scan(&userID)
	if errors.Is(err, gocql.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.FindByID(userID)
}

// SetEmailVerified marks the user's email as verified.
func (r *Repository) SetEmailVerified(id uuid.UUID) error {
	return r.session.Query(`
		UPDATE users SET email_verified = true, updated_at = ? WHERE user_id = ?`,
		time.Now().UTC(), id).Exec()
}

// UpdateFailedAttempts sets the failed login counter and lock timestamp.
func (r *Repository) UpdateFailedAttempts(id uuid.UUID, attempts int, lockedUntil *time.Time) error {
	return r.session.Query(`
		UPDATE users SET failed_attempts = ?, locked_until = ?, updated_at = ? WHERE user_id = ?`,
		attempts, lockedUntil, time.Now().UTC(), id).Exec()
}

// UpdateFailedAttemptsReset clears the failed login counter and lock.
func (r *Repository) UpdateFailedAttemptsReset(id uuid.UUID) error {
	return r.session.Query(`
		UPDATE users SET failed_attempts = 0, locked_until = null, updated_at = ? WHERE user_id = ?`,
		time.Now().UTC(), id).Exec()
}

// UpdatePassword replaces the password hash and resets failed attempts.
func (r *Repository) UpdatePassword(id uuid.UUID, hash string) error {
	return r.session.Query(`
		UPDATE users SET password_hash = ?, failed_attempts = 0, locked_until = null, updated_at = ?
		WHERE user_id = ?`,
		hash, time.Now().UTC(), id).Exec()
}

// UpdateTOTP stores the TOTP secret and enabled flag.
func (r *Repository) UpdateTOTP(id uuid.UUID, secret string, enabled bool) error {
	return r.session.Query(`
		UPDATE users SET totp_secret = ?, totp_enabled = ?, updated_at = ? WHERE user_id = ?`,
		secret, enabled, time.Now().UTC(), id).Exec()
}
