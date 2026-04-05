package auth

import (
	"strings"
	"testing"
)

// TestValidatePasswordPolicy covers each individual rule violation and the happy path.
func TestValidatePasswordPolicy(t *testing.T) {
	tests := []struct {
		name            string
		password        string
		wantViolations  []string // rule names expected in violations
		wantNoViolation bool
	}{
		{
			name:            "valid password passes all rules",
			password:        "ValidPass1!aaa",
			wantNoViolation: true,
		},
		{
			name:           "too short",
			password:       "Short1!",
			wantViolations: []string{"min_length"},
		},
		{
			name:           "no uppercase letter",
			password:       "nouppercase1!aa",
			wantViolations: []string{"uppercase"},
		},
		{
			name:           "no lowercase letter",
			password:       "NOLOWERCASE1!AA",
			wantViolations: []string{"lowercase"},
		},
		{
			name:           "no digit",
			password:       "NoDigitPass!!!",
			wantViolations: []string{"digit"},
		},
		{
			name:           "no special character",
			password:       "NoSpecialPass1",
			wantViolations: []string{"special"},
		},
		{
			name:           "all rules violated",
			password:       "weak",
			wantViolations: []string{"min_length", "uppercase", "digit", "special"},
		},
		{
			name:           "exactly 12 chars but missing special",
			password:       "ValidPass1234",
			wantViolations: []string{"special"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := ValidatePasswordPolicy(tt.password)

			if tt.wantNoViolation {
				if len(violations) != 0 {
					t.Errorf("expected no violations, got %+v", violations)
				}
				return
			}

			ruleSet := make(map[string]bool, len(violations))
			for _, v := range violations {
				ruleSet[v.Rule] = true
			}

			for _, rule := range tt.wantViolations {
				if !ruleSet[rule] {
					t.Errorf("expected violation rule %q, got violations: %+v", rule, violations)
				}
			}
		})
	}
}

// TestHashAndVerifyPassword covers the round-trip and error paths.
func TestHashAndVerifyPassword(t *testing.T) {
	t.Run("round-trip succeeds", func(t *testing.T) {
		hash, err := HashPassword("MyPassword1!", "pepper")
		if err != nil {
			t.Fatalf("unexpected error hashing: %v", err)
		}
		if err := VerifyPassword("MyPassword1!", "pepper", hash); err != nil {
			t.Fatalf("expected verification to succeed, got: %v", err)
		}
	})

	t.Run("wrong password returns ErrHashMismatch", func(t *testing.T) {
		hash, _ := HashPassword("MyPassword1!", "pepper")
		err := VerifyPassword("WrongPassword1!", "pepper", hash)
		if err != ErrHashMismatch {
			t.Errorf("expected ErrHashMismatch, got: %v", err)
		}
	})

	t.Run("wrong pepper returns ErrHashMismatch", func(t *testing.T) {
		hash, _ := HashPassword("MyPassword1!", "pepper")
		err := VerifyPassword("MyPassword1!", "wrong-pepper", hash)
		if err != ErrHashMismatch {
			t.Errorf("expected ErrHashMismatch, got: %v", err)
		}
	})

	t.Run("completely malformed hash returns ErrInvalidHash", func(t *testing.T) {
		err := VerifyPassword("MyPassword1!", "pepper", "not-a-hash-at-all")
		if err != ErrInvalidHash {
			t.Errorf("expected ErrInvalidHash, got: %v", err)
		}
	})

	t.Run("hash with wrong algorithm returns ErrInvalidHash", func(t *testing.T) {
		err := VerifyPassword("MyPassword1!", "pepper", "$bcrypt$bad$stuff$here$extra")
		if err != ErrInvalidHash {
			t.Errorf("expected ErrInvalidHash, got: %v", err)
		}
	})

	t.Run("hash with bad version field returns ErrInvalidHash", func(t *testing.T) {
		err := VerifyPassword("MyPassword1!", "pepper", "$argon2id$badversion$m=65536,t=3,p=4$salt$hash")
		if err != ErrInvalidHash {
			t.Errorf("expected ErrInvalidHash, got: %v", err)
		}
	})

	t.Run("hash with bad param field returns ErrInvalidHash", func(t *testing.T) {
		err := VerifyPassword("MyPassword1!", "pepper", "$argon2id$v=19$badparams$salt$hash")
		if err != ErrInvalidHash {
			t.Errorf("expected ErrInvalidHash, got: %v", err)
		}
	})

	t.Run("hash with invalid base64 salt returns ErrInvalidHash", func(t *testing.T) {
		err := VerifyPassword("MyPassword1!", "pepper", "$argon2id$v=19$m=65536,t=3,p=4$!!!invalidsalt!!!$hash")
		if err != ErrInvalidHash {
			t.Errorf("expected ErrInvalidHash, got: %v", err)
		}
	})

	t.Run("hash with invalid base64 hash returns ErrInvalidHash", func(t *testing.T) {
		// Valid base64 salt (16 bytes), invalid hash
		err := VerifyPassword("MyPassword1!", "pepper", "$argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHRzYWx0c2E$!!!invalidhash!!!")
		if err != ErrInvalidHash {
			t.Errorf("expected ErrInvalidHash, got: %v", err)
		}
	})

	t.Run("different hashes for same password are both valid", func(t *testing.T) {
		h1, _ := HashPassword("MyPassword1!", "pepper")
		h2, _ := HashPassword("MyPassword1!", "pepper")
		// Salts should differ — hashes should not be equal
		if h1 == h2 {
			t.Error("expected different hashes due to random salt")
		}
		// But both should verify correctly
		if err := VerifyPassword("MyPassword1!", "pepper", h1); err != nil {
			t.Errorf("h1 verification failed: %v", err)
		}
		if err := VerifyPassword("MyPassword1!", "pepper", h2); err != nil {
			t.Errorf("h2 verification failed: %v", err)
		}
	})

	t.Run("too few parts in hash returns ErrInvalidHash", func(t *testing.T) {
		err := VerifyPassword("pass", "pepper", "$argon2id$v=19$m=65536,t=3,p=4$onlyfourparts")
		if err != ErrInvalidHash {
			t.Errorf("expected ErrInvalidHash, got: %v", err)
		}
	})

	t.Run("empty hash returns ErrInvalidHash", func(t *testing.T) {
		err := VerifyPassword("pass", "pepper", "")
		if err != ErrInvalidHash {
			t.Errorf("expected ErrInvalidHash, got: %v", err)
		}
	})
}

// TestValidatePasswordPolicy_ExactBoundary verifies the minimum length boundary precisely.
func TestValidatePasswordPolicy_ExactBoundary(t *testing.T) {
	// 11 characters — one below threshold, with all other requirements met
	short := "ValidPass1!a" // 12 chars — should pass
	if v := ValidatePasswordPolicy(short); len(v) != 0 {
		t.Errorf("12-char password should pass min_length; got violations: %v", v)
	}

	// 11 chars — should fail min_length
	tooShort := "ValidPass1!"
	if len(tooShort) != 11 {
		t.Fatalf("test setup error: expected 11 chars, got %d", len(tooShort))
	}
	violations := ValidatePasswordPolicy(tooShort)
	found := false
	for _, v := range violations {
		if v.Rule == "min_length" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected min_length violation for 11-char password, got: %v", violations)
	}
}

// TestValidatePasswordPolicy_ViolationMessages checks that violation messages are non-empty.
func TestValidatePasswordPolicy_ViolationMessages(t *testing.T) {
	violations := ValidatePasswordPolicy("weak")
	for _, v := range violations {
		if strings.TrimSpace(v.Message) == "" {
			t.Errorf("violation %q has empty message", v.Rule)
		}
	}
}
