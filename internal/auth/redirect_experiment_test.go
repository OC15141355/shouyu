package auth

import (
	"net/url"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

// TestRedirectURIShape documents (with assertions) exactly what
// oauth2.Config.AuthCodeURL puts in the redirect_uri param.
//
// This is the spec §15 Q3 experiment. If the assertions below
// ever fail, the doubling-redirect-uri bug from 2b W3.9 is back.
func TestRedirectURIShape(t *testing.T) {
	cases := []struct {
		name       string
		configured string
		want       string
	}{
		{"full URL passes through", "https://home.yagura.dev/oauth/callback", "https://home.yagura.dev/oauth/callback"},
		{"path-only stays path-only (no auto-prepend)", "/oauth/callback", "/oauth/callback"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &oauth2.Config{
				ClientID:    "shouyu",
				RedirectURL: tc.configured,
				Endpoint: oauth2.Endpoint{
					AuthURL:  "https://keycloak.yagura.dev/realms/homelab/protocol/openid-connect/auth",
					TokenURL: "https://keycloak.yagura.dev/realms/homelab/protocol/openid-connect/token",
				},
				Scopes: []string{"openid", "profile", "groups"},
			}
			u := cfg.AuthCodeURL("state-xyz")
			parsed, err := url.Parse(u)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			got := parsed.Query().Get("redirect_uri")
			if got != tc.want {
				t.Fatalf("redirect_uri = %q, want %q", got, tc.want)
			}
			// Verify there's exactly one redirect_uri (not the 2b doubled trap).
			if strings.Count(u, "redirect_uri=") != 1 {
				t.Fatalf("redirect_uri appears %d times in %q; want exactly 1", strings.Count(u, "redirect_uri="), u)
			}
		})
	}
}
