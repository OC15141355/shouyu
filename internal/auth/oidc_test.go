package auth

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewProvider_MissingFieldsRejected(t *testing.T) {
	// SessionSecret is 32+ bytes in every case so the per-field check is
	// the only one that can fail.
	goodSecret := strings.Repeat("a", 32)
	cases := []struct {
		name string
		cfg  Config
	}{
		{"no issuer", Config{ClientID: "shouyu", ClientSecret: "x", RedirectURL: "https://x/cb", SessionSecret: goodSecret}},
		{"no client id", Config{IssuerURL: "https://kc/realms/r", ClientSecret: "x", RedirectURL: "https://x/cb", SessionSecret: goodSecret}},
		{"no client secret", Config{IssuerURL: "https://kc/realms/r", ClientID: "shouyu", RedirectURL: "https://x/cb", SessionSecret: goodSecret}},
		{"no redirect", Config{IssuerURL: "https://kc/realms/r", ClientID: "shouyu", ClientSecret: "x", SessionSecret: goodSecret}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewProvider(context.Background(), tc.cfg)
			if err == nil {
				t.Fatalf("want error, got nil")
			}
		})
	}
}

func TestNewProvider_SessionSecretTooShortRejected(t *testing.T) {
	cfg := Config{
		IssuerURL:     "https://kc/realms/r",
		ClientID:      "shouyu",
		ClientSecret:  "x",
		RedirectURL:   "https://x/cb",
		SessionSecret: "tooshort", // < 32 bytes
	}
	_, err := NewProvider(context.Background(), cfg)
	if err == nil {
		t.Fatal("want error for short SessionSecret, got nil")
	}
}

// TestNewProvider_UnreachableIssuer ensures NewProvider returns
// (and doesn't hang forever) when Keycloak isn't reachable.
func TestNewProvider_UnreachableIssuer(t *testing.T) {
	srv := httptest.NewServer(nil)
	srv.Close() // server is dead; URL is unroutable
	cfg := Config{
		IssuerURL:     srv.URL + "/realms/x",
		ClientID:      "shouyu",
		ClientSecret:  "x",
		RedirectURL:   "https://x/cb",
		SessionSecret: strings.Repeat("a", 32),
	}
	_, err := NewProvider(context.Background(), cfg)
	if err == nil {
		t.Fatal("want unreachable error, got nil")
	}
}
