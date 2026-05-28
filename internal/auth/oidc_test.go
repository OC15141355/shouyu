package auth

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestNewProvider_MissingFieldsRejected(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
	}{
		{"no issuer", Config{ClientID: "shouyu", ClientSecret: "x", RedirectURL: "https://x/cb"}},
		{"no client id", Config{IssuerURL: "https://kc/realms/r", ClientSecret: "x", RedirectURL: "https://x/cb"}},
		{"no client secret", Config{IssuerURL: "https://kc/realms/r", ClientID: "shouyu", RedirectURL: "https://x/cb"}},
		{"no redirect", Config{IssuerURL: "https://kc/realms/r", ClientID: "shouyu", ClientSecret: "x"}},
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

// TestNewProvider_UnreachableIssuer ensures NewProvider returns
// (and doesn't hang forever) when Keycloak isn't reachable.
func TestNewProvider_UnreachableIssuer(t *testing.T) {
	srv := httptest.NewServer(nil)
	srv.Close() // server is dead; URL is unroutable
	cfg := Config{
		IssuerURL:    srv.URL + "/realms/x",
		ClientID:     "shouyu",
		ClientSecret: "x",
		RedirectURL:  "https://x/cb",
	}
	_, err := NewProvider(context.Background(), cfg)
	if err == nil {
		t.Fatal("want unreachable error, got nil")
	}
}
