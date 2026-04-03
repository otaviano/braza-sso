package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"unicode"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory  = 64 * 1024 // 64 MB
	argonTime    = 3
	argonThreads = 4
	argonKeyLen  = 32
	argonSaltLen = 16
)

var (
	ErrWeakPassword    = errors.New("password does not meet policy requirements")
	ErrInvalidHash     = errors.New("invalid password hash format")
	ErrHashMismatch    = errors.New("password does not match")

	reSpecial = regexp.MustCompile(`[^a-zA-Z0-9]`)
)

// PolicyViolation describes an unmet password policy criterion.
type PolicyViolation struct {
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// ValidatePasswordPolicy checks the password against the security policy.
// Returns a list of unmet criteria (empty = valid).
func ValidatePasswordPolicy(password string) []PolicyViolation {
	var violations []PolicyViolation

	if len(password) < 12 {
		violations = append(violations, PolicyViolation{"min_length", "minimum 12 characters required"})
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			hasSpecial = true
		}
	}

	if !hasUpper {
		violations = append(violations, PolicyViolation{"uppercase", "at least one uppercase letter required"})
	}
	if !hasLower {
		violations = append(violations, PolicyViolation{"lowercase", "at least one lowercase letter required"})
	}
	if !hasDigit {
		violations = append(violations, PolicyViolation{"digit", "at least one digit required"})
	}
	if !hasSpecial {
		violations = append(violations, PolicyViolation{"special", "at least one special character required"})
	}

	return violations
}

// HashPassword hashes password+pepper with Argon2id and returns a storable string.
func HashPassword(password, pepper string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password+pepper),
		salt,
		argonTime,
		argonMemory,
		argonThreads,
		argonKeyLen,
	)

	// Format: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory,
		argonTime,
		argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}

// VerifyPassword checks a password+pepper against a stored Argon2id hash.
// Uses constant-time comparison to prevent timing attacks.
func VerifyPassword(password, pepper, encodedHash string) error {
	var version, memory, time, threads uint32
	var saltB64, hashB64 string

	_, err := fmt.Sscanf(
		encodedHash,
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s",
		&version, &memory, &time, &threads, &saltB64,
	)
	if err != nil {
		return ErrInvalidHash
	}

	// Split off the hash part after the last $
	parts := splitLast(encodedHash, "$")
	if len(parts) != 2 {
		return ErrInvalidHash
	}
	hashB64 = parts[1]

	salt, err := base64.RawStdEncoding.DecodeString(saltB64)
	if err != nil {
		return ErrInvalidHash
	}
	expectedHash, err := base64.RawStdEncoding.DecodeString(hashB64)
	if err != nil {
		return ErrInvalidHash
	}

	computed := argon2.IDKey(
		[]byte(password+pepper),
		salt,
		time,
		memory,
		uint8(threads),
		uint32(len(expectedHash)),
	)

	// Constant-time comparison
	if subtle.ConstantTimeCompare(computed, expectedHash) != 1 {
		return ErrHashMismatch
	}

	return nil
}

func splitLast(s, sep string) []string {
	idx := -1
	for i := len(s) - 1; i >= 0; i-- {
		if string(s[i]) == sep {
			idx = i
			break
		}
	}
	if idx < 0 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+1:]}
}
