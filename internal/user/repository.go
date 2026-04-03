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
func (r *Repository) Create(u *User) error {
	// Lightweight duplicate check via the email index
	existing, err := r.FindByEmail(u.Email)
	if err == nil && existing != nil {
		return ErrEmailTaken
	}

	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now

	return r.session.Query(`
		INSERT INTO users
		  (user_id, email, password_hash, email_verified, totp_enabled, totp_secret,
		   locked_until, failed_attempts, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Email, u.PasswordHash, u.EmailVerified,
		u.TOTPEnabled, u.TOTPSecret, u.LockedUntil, u.FailedAttempts,
		u.CreatedAt, u.UpdatedAt,
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

// FindByEmail retrieves a user by email address (uses secondary index).
func (r *Repository) FindByEmail(email string) (*User, error) {
	u := &User{}
	err := r.session.Query(`
		SELECT user_id, email, password_hash, email_verified, totp_enabled, totp_secret,
		       locked_until, failed_attempts, created_at, updated_at
		FROM users WHERE email = ? LIMIT 1 ALLOW FILTERING`, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.EmailVerified,
			&u.TOTPEnabled, &u.TOTPSecret, &u.LockedUntil, &u.FailedAttempts,
			&u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, gocql.ErrNotFound) {
		return nil, ErrNotFound
	}
	return u, err
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
