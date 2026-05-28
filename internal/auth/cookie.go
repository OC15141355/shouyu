package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

// signCookieValue produces "<id>.<sigB64URL>" where sig = HMAC-SHA256(secret, id).
// Cookies on the wire are this string; readers must call verifyCookieValue.
func signCookieValue(id, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(id))
	sig := mac.Sum(nil)
	return id + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// verifyCookieValue extracts the id from "<id>.<sigB64URL>" iff the signature
// matches HMAC-SHA256(secret, id) in constant time. Returns "", false on any
// malformed input or signature mismatch.
//
// LastIndex (not Index) is deliberate: session IDs today are 64-char hex with
// no dot, but using LastIndex keeps the separator unambiguous if the ID format
// ever changes to include a dot.
func verifyCookieValue(value, secret string) (string, bool) {
	dot := strings.LastIndex(value, ".")
	if dot <= 0 || dot == len(value)-1 {
		return "", false
	}
	id, sigB64 := value[:dot], value[dot+1:]
	// Strict() rejects encodings where the trailing pad bits aren't zero.
	// Without it, two distinct cookie strings (e.g. ending in "0" vs "1")
	// decode to the same 32 bytes, letting an attacker flip a low-order
	// bit of the encoded signature without changing what HMAC verifies —
	// silently accepted as a "valid" forgery.
	sig, err := base64.RawURLEncoding.Strict().DecodeString(sigB64)
	if err != nil || len(sig) != sha256.Size {
		return "", false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(id))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return "", false
	}
	return id, true
}
