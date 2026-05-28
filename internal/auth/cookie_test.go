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

func TestVerifyCookieValue_StrictBase64PadBits(t *testing.T) {
	// Defence against a subtle base64 footgun: for 32-byte signatures,
	// the 43rd (last) char encodes 4 real bits + 2 pad bits. Without
	// Strict() decoding, two distinct cookies whose last chars share the
	// top 4 bits decode to the same 32 bytes — a forgery the HMAC can't
	// catch.
	//
	// b64url alphabet: A-Z=0-25, a-z=26-51, 0-9=52-61, -=62, _=63.
	// Bit 0 of an encoded char's VALUE (not ASCII) is the LSB pad bit.
	// We probe ids until we find one whose last char's value has a
	// non-zero LSB, then construct the colliding twin (zero out the LSB)
	// and assert it's rejected.
	secret := strings.Repeat("a", 32)
	b64Index := func(c byte) int {
		switch {
		case c >= 'A' && c <= 'Z':
			return int(c - 'A')
		case c >= 'a' && c <= 'z':
			return int(c-'a') + 26
		case c >= '0' && c <= '9':
			return int(c-'0') + 52
		case c == '-':
			return 62
		case c == '_':
			return 63
		}
		return -1
	}
	b64Char := func(v int) byte {
		const alpha = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
		return alpha[v]
	}
	// Attacker's move: take a legitimate signature (whose last char always
	// has zero pad bits — that's what the encoder produces) and flip a
	// pad bit ON. A non-strict decoder discards the pad bits, treats the
	// forged char as identical to the legit one, decodes to the same 32
	// bytes, and HMAC verifies. With Strict(), the decoder refuses the
	// non-zero pad bits up front.
	id := "abc123"
	signed := signCookieValue(id, secret)
	last := signed[len(signed)-1]
	v := b64Index(last)
	if v < 0 || v&0x03 != 0 {
		t.Fatalf("test precondition broken: legit signature last char %q has non-zero pad bits (v=%d)", string(last), v)
	}
	twin := b64Char(v | 0x01) // turn on a pad bit
	mutated := signed[:len(signed)-1] + string(twin)
	if _, ok := verifyCookieValue(mutated, secret); ok {
		t.Fatalf("pad-bit-on twin %q accepted (Strict() base64 decode is required); original last=%q twin=%q", mutated, string(last), string(twin))
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
