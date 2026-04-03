package user

import (
	"time"

	"github.com/google/uuid"
)

// Status of email verification.
type Status string

const (
	StatusPending  Status = "pending"
	StatusActive   Status = "active"
	StatusLocked   Status = "locked"
)

// User represents a registered account.
type User struct {
	ID             uuid.UUID
	Email          string
	PasswordHash   string
	EmailVerified  bool
	TOTPEnabled    bool
	TOTPSecret     string
	LockedUntil    *time.Time
	FailedAttempts int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
