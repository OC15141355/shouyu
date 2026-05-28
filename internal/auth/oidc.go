package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Config holds the OIDC client configuration. All fields are required.
type Config struct {
	IssuerURL    string // e.g. https://keycloak.yagura.dev/realms/homelab
	ClientID     string
	ClientSecret string
	RedirectURL  string // full URL, e.g. https://home.yagura.dev/oauth/callback
	// PostLogoutURL is the RP-initiated logout target (default
	// https://home.yagura.dev/). Empty disables hint inclusion in the
	// end-session redirect; validation does not require it.
	PostLogoutURL string
	// RequiredRole, if non-empty, must appear in
	// id_token.resource_access.<ClientID>.roles for the user to be allowed in.
	RequiredRole string
	// SessionSecret is used to sign session cookies. Must be 32+ bytes.
	SessionSecret string
}

func (c Config) validate() error {
	if c.IssuerURL == "" {
		return errors.New("auth: IssuerURL required")
	}
	if c.ClientID == "" {
		return errors.New("auth: ClientID required")
	}
	if c.ClientSecret == "" {
		return errors.New("auth: ClientSecret required")
	}
	if c.RedirectURL == "" {
		return errors.New("auth: RedirectURL required")
	}
	return nil
}

// Provider bundles an OIDC provider, OAuth2 config, and ID token verifier.
type Provider struct {
	oidc     *oidc.Provider
	oauth    *oauth2.Config
	verifier *oidc.IDTokenVerifier
	cfg      Config
}

// NewProvider discovers the issuer and returns a Provider.
// Returns error if config is invalid or the issuer is unreachable.
func NewProvider(ctx context.Context, cfg Config) (*Provider, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	op, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("auth: oidc.NewProvider: %w", err)
	}
	verifier := op.Verifier(&oidc.Config{ClientID: cfg.ClientID})
	oc := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     op.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "groups"},
	}
	return &Provider{oidc: op, oauth: oc, verifier: verifier, cfg: cfg}, nil
}

// Config returns the (read-only) Config the provider was built with.
func (p *Provider) Config() Config { return p.cfg }
