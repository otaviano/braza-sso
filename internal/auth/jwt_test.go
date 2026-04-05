package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
	"time"
)

func writePEMToTemp(t *testing.T, block *pem.Block) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "key*.pem")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if err := pem.Encode(f, block); err != nil {
		t.Fatalf("pem encode: %v", err)
	}
	_ = f.Close()
	return f.Name()
}

func newTestRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	return key
}

func newPKCS1KeyFile(t *testing.T) (string, *rsa.PrivateKey) {
	t.Helper()
	key := newTestRSAKey(t)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}
	return writePEMToTemp(t, block), key
}

func newPKCS8RSAKeyFile(t *testing.T) string {
	t.Helper()
	key := newTestRSAKey(t)
	keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal PKCS8: %v", err)
	}
	block := &pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes}
	return writePEMToTemp(t, block)
}

func newTokenServiceFromKey(t *testing.T) *TokenService {
	t.Helper()
	keyPath, _ := newPKCS1KeyFile(t)
	svc, err := NewTokenService(keyPath, "test-issuer", time.Minute)
	if err != nil {
		t.Fatalf("NewTokenService: %v", err)
	}
	return svc
}

func TestGenerateRSAKeyPair_ReturnsValidKey(t *testing.T) {
	key, err := GenerateRSAKeyPair()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key == nil {
		t.Fatal("expected non-nil key")
	}
	if err := key.Validate(); err != nil {
		t.Fatalf("generated key is invalid: %v", err)
	}
}

func TestNewTokenService_PKCS1Key(t *testing.T) {
	keyPath, _ := newPKCS1KeyFile(t)
	svc, err := NewTokenService(keyPath, "issuer", time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewTokenService_PKCS8RSAKey(t *testing.T) {
	keyPath := newPKCS8RSAKeyFile(t)
	svc, err := NewTokenService(keyPath, "issuer", time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestNewTokenService_FileNotFound_ReturnsError(t *testing.T) {
	_, err := NewTokenService("/nonexistent/path/key.pem", "issuer", time.Minute)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestNewTokenService_InvalidPEM_ReturnsError(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "key*.pem")
	_, _ = f.WriteString("not a pem block")
	_ = f.Close()
	_, err := NewTokenService(f.Name(), "issuer", time.Minute)
	if err == nil {
		t.Fatal("expected error for invalid PEM")
	}
}

func TestNewTokenService_InvalidKeyBytes_ReturnsError(t *testing.T) {
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: []byte("garbage")}
	keyPath := writePEMToTemp(t, block)
	_, err := NewTokenService(keyPath, "issuer", time.Minute)
	if err == nil {
		t.Fatal("expected error for invalid key bytes")
	}
}

func TestNewTokenService_ECKeyNotRSA_ReturnsError(t *testing.T) {
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate EC key: %v", err)
	}
	keyBytes, _ := x509.MarshalPKCS8PrivateKey(ecKey)
	block := &pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes}
	keyPath := writePEMToTemp(t, block)
	_, err = NewTokenService(keyPath, "issuer", time.Minute)
	if err == nil {
		t.Fatal("expected error for non-RSA key")
	}
}

func TestIssueAndVerifyAccessToken_RoundTrip(t *testing.T) {
	svc := newTokenServiceFromKey(t)

	tokenStr, err := svc.IssueAccessToken("user-123", "user@example.com", true, "api")
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("expected non-empty token string")
	}

	claims, err := svc.VerifyAccessToken(tokenStr)
	if err != nil {
		t.Fatalf("VerifyAccessToken: %v", err)
	}
	if claims.Subject != "user-123" {
		t.Errorf("expected subject user-123, got %s", claims.Subject)
	}
	if claims.Email != "user@example.com" {
		t.Errorf("expected email user@example.com, got %s", claims.Email)
	}
	if !claims.EmailVerified {
		t.Error("expected EmailVerified=true")
	}
}

func TestVerifyAccessToken_InvalidToken_ReturnsError(t *testing.T) {
	svc := newTokenServiceFromKey(t)
	_, err := svc.VerifyAccessToken("not.a.valid.jwt")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestVerifyAccessToken_WrongAlgorithm_ReturnsError(t *testing.T) {
	svc := newTokenServiceFromKey(t)
	// HS256 token — wrong algorithm for this service
	hs256Token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyLTEyMyJ9.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	_, err := svc.VerifyAccessToken(hs256Token)
	if err == nil {
		t.Fatal("expected error for wrong-algorithm token")
	}
}

func TestPublicKeyJWK_ContainsRequiredFields(t *testing.T) {
	svc := newTokenServiceFromKey(t)
	jwk := svc.PublicKeyJWK()

	for _, field := range []string{"kty", "use", "alg", "kid", "n", "e"} {
		val, ok := jwk[field]
		if !ok {
			t.Errorf("missing JWK field %q", field)
			continue
		}
		if val == "" {
			t.Errorf("JWK field %q is empty", field)
		}
	}
	if jwk["kty"] != "RSA" {
		t.Errorf("expected kty=RSA, got %v", jwk["kty"])
	}
}
