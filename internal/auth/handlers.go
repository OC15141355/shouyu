package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"

	"github.com/coreos/go-oidc/v3/oidc"
)

const (
	sessionCookieName = "shouyu_session"
	stateCookieName   = "shouyu_oauth_state"
	stateCookieTTL    = 600 // seconds (10 min)
)

// Handlers wires OIDC flow handlers to a Provider and a SessionStore.
type Handlers struct {
	p     *Provider
	store *SessionStore
}

func NewHandlers(p *Provider, s *SessionStore) *Handlers {
	return &Handlers{p: p, store: s}
}

// Login starts the OIDC code-flow.
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	state := randHex(16)
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Path:     "/",
		MaxAge:   stateCookieTTL,
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, h.p.oauth.AuthCodeURL(state), http.StatusFound)
}

// Callback completes the OIDC dance and issues a session cookie.
func (h *Handlers) Callback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie(stateCookieName)
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	tok, err := h.p.oauth.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "exchange failed", http.StatusBadGateway)
		return
	}
	rawID, ok := tok.Extra("id_token").(string)
	if !ok {
		http.Error(w, "no id_token", http.StatusBadGateway)
		return
	}
	idtok, err := h.p.verifier.Verify(ctx, rawID)
	if err != nil {
		http.Error(w, "id_token verify failed", http.StatusForbidden)
		return
	}
	claims := struct {
		Username       string   `json:"preferred_username"`
		Name           string   `json:"name"`
		Email          string   `json:"email"`
		Groups         []string `json:"groups"`
		ResourceAccess map[string]struct {
			Roles []string `json:"roles"`
		} `json:"resource_access"`
	}{}
	if err := idtok.Claims(&claims); err != nil {
		http.Error(w, "claims parse failed", http.StatusBadGateway)
		return
	}
	if h.p.cfg.RequiredRole != "" {
		ra, ok := claims.ResourceAccess[h.p.cfg.ClientID]
		if !ok || !contains(ra.Roles, h.p.cfg.RequiredRole) {
			http.Error(w, fmt.Sprintf("missing required role %q", h.p.cfg.RequiredRole), http.StatusForbidden)
			return
		}
	}
	id := h.store.NewID()
	h.store.Put(id, Session{
		Username:   claims.Username,
		Name:       claims.Name,
		Email:      claims.Email,
		Groups:     claims.Groups,
		RawIDToken: rawID,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
		SameSite: http.SameSiteLaxMode,
	})
	// Clear state cookie
	http.SetCookie(w, &http.Cookie{Name: stateCookieName, MaxAge: -1, Path: "/"})
	http.Redirect(w, r, "/", http.StatusFound)
}

// Logout kills the local session and redirects to Keycloak end-session.
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	var idToken string
	if c, err := r.Cookie(sessionCookieName); err == nil {
		// Atomic read+delete: avoids a TOCTOU race between concurrent
		// logouts on the same session ID (both would otherwise observe
		// the session and both attempt deletion).
		if sess, ok := h.store.GetAndDelete(c.Value); ok {
			idToken = sess.RawIDToken
		}
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, MaxAge: -1, Path: "/"})

	// Discover the end-session endpoint from the OIDC provider's metadata.
	endSession := h.p.oidc.Endpoint().AuthURL // fallback if discovery doesn't expose end_session
	var disc struct {
		EndSession string `json:"end_session_endpoint"`
	}
	_ = h.p.oidc.Claims(&disc)
	if disc.EndSession != "" {
		endSession = disc.EndSession
	}

	// Build the query via net/url.Values so the redirect URI + id_token_hint
	// are properly percent-encoded.
	q := url.Values{}
	if h.p.cfg.PostLogoutURL != "" {
		q.Set("post_logout_redirect_uri", h.p.cfg.PostLogoutURL)
	}
	if idToken != "" {
		// Keycloak 26.x requires id_token_hint for silent RP-initiated logout;
		// without it Keycloak renders a confirmation prompt.
		q.Set("id_token_hint", idToken)
	}
	if enc := q.Encode(); enc != "" {
		endSession += "?" + enc
	}
	http.Redirect(w, r, endSession, http.StatusFound)
}

// silence unused
var _ = oidc.ScopeOpenID
var _ = context.Background

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
