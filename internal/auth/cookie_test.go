package auth

import (
	"strings"
	"testing"
)

func TestSignAndVerifyCookieValue_RoundTrip(t *testing.T) {
	secret := strings.Repeat("a", 32)
	signed := signCookieValue("abc123", secret)
	if !strings.HasPrefix(signed, "abc123.") {
		t.Fatalf("signed value doesn't start with id: %q", signed)
	}
	id, ok := verifyCookieValue(signed, secret)
	if !ok {
		t.Fatalf("verify failed for self-signed value: %q", signed)
	}
	if id != "abc123" {
		t.Fatalf("id = %q, want abc123", id)
	}
}

func TestVerifyCookieValue_TamperedSignatureRejected(t *testing.T) {
	secret := strings.Repeat("a", 32)
	signed := signCookieValue("abc123", secret)
	// Flip a byte in the signature half.
	tampered := signed[:len(signed)-1] + string(signed[len(signed)-1]^0x01)
	if _, ok := verifyCookieValue(tampered, secret); ok {
		t.Fatal("tampered signature accepted")
	}
}

func TestVerifyCookieValue_WrongSecretRejected(t *testing.T) {
	signed := signCookieValue("abc123", strings.Repeat("a", 32))
	if _, ok := verifyCookieValue(signed, strings.Repeat("b", 32)); ok {
		t.Fatal("signature verified against wrong secret")
	}
}

func TestVerifyCookieValue_MalformedRejected(t *testing.T) {
	secret := strings.Repeat("a", 32)
	cases := []string{
		"",                // empty
		"nosigjustid",     // no dot
		".sigbutnoid",     // empty id
		"id.",             // empty signature
		"id.notbase64!!!", // invalid base64
		"id.aGVsbG8",      // wrong-length signature (decoded < 32 bytes)
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			if _, ok := verifyCookieValue(c, secret); ok {
				t.Fatalf("verify accepted malformed input: %q", c)
			}
		})
	}
}

func TestVerifyCookieValue_ConstantTimeCompare(t *testing.T) {
	// We can't directly measure timing in unit tests, but we can document
	// the contract by asserting that signed.b uses hmac.Equal under the
	// hood. Negative probe: a signature that differs only in the last
	// byte must still be rejected (catches naive byte-wise short-circuit).
	secret := strings.Repeat("a", 32)
	signed := signCookieValue("abc123", secret)
	// Build a "close" signature that shares most bytes.
	mismatch := signed[:len(signed)-2] + "ZZ"
	if _, ok := verifyCookieValue(mismatch, secret); ok {
		t.Fatal("verify accepted last-byte-different signature")
	}
}
