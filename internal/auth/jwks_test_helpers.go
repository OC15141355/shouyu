//go:build test_helpers

package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
)

func writeRSAJWKS(w http.ResponseWriter, pub *rsa.PublicKey) {
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes())
	_ = json.NewEncoder(w).Encode(map[string]any{
		"keys": []map[string]any{{
			"kty": "RSA", "kid": "stub-key", "use": "sig", "alg": "RS256",
			"n": n, "e": e,
		}},
	})
}
