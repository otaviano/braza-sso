package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT payload.
type Claims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	jwt.RegisteredClaims
}

// TokenService handles JWT signing and verification.
type TokenService struct {
	privateKey *rsa.PrivateKey
	issuer     string
	accessTTL  time.Duration
}

func NewTokenService(privateKeyPath, issuer string, accessTTL time.Duration) (*TokenService, error) {
	keyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("reading private key: %w", err)
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, errors.New("failed to decode PEM block from private key file")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8
		key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("parsing private key: %w", err)
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("private key is not RSA")
		}
	}

	return &TokenService{
		privateKey: privateKey,
		issuer:     issuer,
		accessTTL:  accessTTL,
	}, nil
}

// IssueAccessToken creates a signed RS256 JWT access token.
func (ts *TokenService) IssueAccessToken(userID, email string, emailVerified bool, audience string) (string, error) {
	now := time.Now()
	claims := Claims{
		Email:         email,
		EmailVerified: emailVerified,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ts.issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{audience},
			ExpiresAt: jwt.NewNumericDate(now.Add(ts.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        newJTI(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(ts.privateKey)
}

// VerifyAccessToken parses and validates a JWT, returning its claims.
func (ts *TokenService) VerifyAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return &ts.privateKey.PublicKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// PublicKeyJWK returns the RSA public key in JWK format for the JWKS endpoint.
func (ts *TokenService) PublicKeyJWK() map[string]interface{} {
	pub := &ts.privateKey.PublicKey
	return map[string]interface{}{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": "braza-sso-key-1",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}
}

func newJTI() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// GenerateRSAKeyPair is a helper for the keygen script — not used in production path.
func GenerateRSAKeyPair() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 4096)
}
