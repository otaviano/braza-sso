package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	prefixEmailVerify    = "email_verify:"
	prefixPasswordReset  = "pwd_reset:"
	prefixRefreshToken   = "refresh:"
	prefixUserSessions   = "user_sessions:"
	prefixMFASession     = "mfa_session:"
	prefixRateLimit      = "rate:"
	prefixLoginAttempts  = "login_attempts:"
)

const (
	maxLoginAttempts    = 5
	loginAttemptWindow  = 15 * time.Minute
	lockoutDuration     = 30 * time.Minute
)

// ErrTokenNotFound is returned when a token does not exist or has expired.
var ErrTokenNotFound = fmt.Errorf("token not found or expired")

// TokenStore manages short-lived tokens in Redis.
type TokenStore struct {
	redis *redis.Client
}

func NewTokenStore(r *redis.Client) *TokenStore {
	return &TokenStore{redis: r}
}

// CreateEmailVerificationToken generates a secure token and stores it in Redis.
func (ts *TokenStore) CreateEmailVerificationToken(ctx context.Context, userID string, ttl time.Duration) (string, error) {
	token := randomToken(32)
	key := prefixEmailVerify + token
	return token, ts.redis.Set(ctx, key, userID, ttl).Err()
}

// ConsumeEmailVerificationToken validates and deletes the token, returning the userID.
func (ts *TokenStore) ConsumeEmailVerificationToken(ctx context.Context, token string) (string, error) {
	key := prefixEmailVerify + token
	userID, err := ts.redis.GetDel(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrTokenNotFound
	}
	return userID, err
}

// CreatePasswordResetToken generates a secure token and stores it in Redis.
func (ts *TokenStore) CreatePasswordResetToken(ctx context.Context, userID string, ttl time.Duration) (string, error) {
	token := randomToken(32)
	key := prefixPasswordReset + token
	return token, ts.redis.Set(ctx, key, userID, ttl).Err()
}

// ConsumePasswordResetToken validates and deletes the token, returning the userID.
func (ts *TokenStore) ConsumePasswordResetToken(ctx context.Context, token string) (string, error) {
	key := prefixPasswordReset + token
	userID, err := ts.redis.GetDel(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("token not found or expired")
	}
	return userID, err
}

// StoreRefreshToken stores an opaque refresh token linked to a userID.
func (ts *TokenStore) StoreRefreshToken(ctx context.Context, token, userID string, ttl time.Duration) error {
	key := prefixRefreshToken + token
	if err := ts.redis.Set(ctx, key, userID, ttl).Err(); err != nil {
		return err
	}
	// Track per-user sessions for full revocation
	return ts.redis.SAdd(ctx, prefixUserSessions+userID, token).Err()
}

// ConsumeRefreshToken validates and atomically deletes a refresh token.
func (ts *TokenStore) ConsumeRefreshToken(ctx context.Context, token string) (string, error) {
	key := prefixRefreshToken + token
	userID, err := ts.redis.GetDel(ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("refresh token not found or expired")
	}
	if err != nil {
		return "", err
	}
	ts.redis.SRem(ctx, prefixUserSessions+userID, token)
	return userID, nil
}

// RevokeAllUserSessions deletes every refresh token for a user (token reuse detection).
func (ts *TokenStore) RevokeAllUserSessions(ctx context.Context, userID string) error {
	setKey := prefixUserSessions + userID
	tokens, err := ts.redis.SMembers(ctx, setKey).Result()
	if err != nil {
		return err
	}
	pipe := ts.redis.Pipeline()
	for _, t := range tokens {
		pipe.Del(ctx, prefixRefreshToken+t)
	}
	pipe.Del(ctx, setKey)
	_, err = pipe.Exec(ctx)
	return err
}

// StoreMFASession stores an intermediate MFA session token.
func (ts *TokenStore) StoreMFASession(ctx context.Context, token, userID string, ttl time.Duration) error {
	return ts.redis.Set(ctx, prefixMFASession+token, userID, ttl).Err()
}

// ConsumeMFASession validates and deletes an MFA session token.
func (ts *TokenStore) ConsumeMFASession(ctx context.Context, token string) (string, error) {
	userID, err := ts.redis.GetDel(ctx, prefixMFASession+token).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("mfa session not found or expired")
	}
	return userID, err
}

// SetState stores an OAuth state token mapped to a returnTo URL.
func (ts *TokenStore) SetState(ctx context.Context, state, returnTo string) error {
	key := prefixRateLimit + "oauth_state:" + state
	return ts.redis.Set(ctx, key, returnTo, 10*time.Minute).Err()
}

// ConsumeState retrieves and deletes an OAuth state token.
func (ts *TokenStore) ConsumeState(ctx context.Context, state string) (string, error) {
	key := prefixRateLimit + "oauth_state:" + state
	val, err := ts.redis.GetDel(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrTokenNotFound
	}
	return val, err
}

// IncrLoginAttempts increments the failed login counter for userID within the sliding window.
// Returns the new count.
func (ts *TokenStore) IncrLoginAttempts(ctx context.Context, userID string) (int64, error) {
	key := prefixLoginAttempts + userID
	pipe := ts.redis.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, loginAttemptWindow)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

// ResetLoginAttempts clears the failed login counter for userID.
func (ts *TokenStore) ResetLoginAttempts(ctx context.Context, userID string) error {
	return ts.redis.Del(ctx, prefixLoginAttempts+userID).Err()
}

// MaxLoginAttempts returns the configured threshold before lockout.
func MaxLoginAttempts() int { return maxLoginAttempts }

// LockoutDuration returns the configured lockout period.
func LockoutDuration() time.Duration { return lockoutDuration }

func randomToken(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
