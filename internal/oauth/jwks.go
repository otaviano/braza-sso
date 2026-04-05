package oauth

import (
	"encoding/json"
	"net/http"

	"github.com/otaviano/braza-sso/internal/auth"
)

// JWKSHandler serves the public key set for RS256 token verification.
func JWKSHandler(ts *auth.TokenService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"keys": []interface{}{ts.PublicKeyJWK()},
		})
	}
}
