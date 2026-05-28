package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// stubKC builds a tiny issuer that serves a discovery doc + jwks + token endpoint.
// Enough to exercise login + callback + verify + RBAC reject.
func stubKC(t *testing.T, withRole bool) (*httptest.Server, *rsa.PrivateKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	var srvURL string

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 srvURL,
			"authorization_endpoint": srvURL + "/auth",
			"token_endpoint":         srvURL + "/token",
			"jwks_uri":               srvURL + "/jwks",
			"end_session_endpoint":   srvURL + "/logout",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		writeJWKS(w, &priv.PublicKey)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		idToken := mintIDToken(t, priv, srvURL, "shouyu", "declan", withRole)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "stub",
			"id_token":     idToken,
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	})
	s := httptest.NewServer(mux)
	srvURL = s.URL
	return s, priv
}

func TestLogin_RedirectsToAuthEndpoint(t *testing.T) {
	kc, _ := stubKC(t, true)
	defer kc.Close()
	p, err := NewProvider(context.Background(), Config{
		IssuerURL: kc.URL, ClientID: "shouyu", ClientSecret: "x",
		RedirectURL:   "https://home.yagura.dev/oauth/callback",
		SessionSecret: strings.Repeat("x", 32),
	})
	if err != nil {
		t.Fatal(err)
	}
	store := NewSessionStore(time.Hour)
	h := NewHandlers(p, store)

	req := httptest.NewRequest("GET", "/oauth/login", nil)
	w := httptest.NewRecorder()
	h.Login(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", w.Code)
	}
	loc := w.Header().Get("Location")
	if !strings.HasPrefix(loc, kc.URL+"/auth?") {
		t.Fatalf("Location = %q", loc)
	}
	if !strings.Contains(loc, "redirect_uri=https%3A%2F%2Fhome.yagura.dev%2Foauth%2Fcallback") {
		t.Fatalf("redirect_uri not in Location: %q", loc)
	}
	// state cookie set
	if len(w.Result().Cookies()) == 0 {
		t.Fatal("expected state cookie")
	}
}

func TestCallback_AcceptsValidUser(t *testing.T) {
	kc, _ := stubKC(t, true)
	defer kc.Close()
	secret := strings.Repeat("x", 32)
	p, _ := NewProvider(context.Background(), Config{
		IssuerURL: kc.URL, ClientID: "shouyu", ClientSecret: "x",
		RedirectURL:   kc.URL + "/cb",
		RequiredRole:  "shouyu-user",
		SessionSecret: secret,
	})
	store := NewSessionStore(time.Hour)
	h := NewHandlers(p, store)
	// simulate login flow: first hit /login to get state cookie, then /callback
	loginReq := httptest.NewRequest("GET", "/oauth/login", nil)
	loginW := httptest.NewRecorder()
	h.Login(loginW, loginReq)
	stateCookie := loginW.Result().Cookies()[0]

	cbReq := httptest.NewRequest("GET", "/oauth/callback?state="+stateCookie.Value+"&code=fake", nil)
	cbReq.AddCookie(stateCookie)
	cbW := httptest.NewRecorder()
	h.Callback(cbW, cbReq)

	if cbW.Code != http.StatusFound {
		t.Fatalf("status = %d body=%s", cbW.Code, cbW.Body.String())
	}
	// session cookie set
	var sessCookie *http.Cookie
	for _, c := range cbW.Result().Cookies() {
		if c.Name == sessionCookieName {
			sessCookie = c
		}
	}
	if sessCookie == nil {
		t.Fatal("session cookie not set")
	}
	id, ok := verifyCookieValue(sessCookie.Value, secret)
	if !ok {
		t.Fatalf("session cookie value %q failed HMAC verification", sessCookie.Value)
	}
	if _, ok := store.Get(id); !ok {
		t.Fatal("session not in store")
	}
}

func TestCallback_RejectsMissingRole(t *testing.T) {
	kc, _ := stubKC(t, false) // user has NO role
	defer kc.Close()
	p, _ := NewProvider(context.Background(), Config{
		IssuerURL: kc.URL, ClientID: "shouyu", ClientSecret: "x",
		RedirectURL:   kc.URL + "/cb",
		RequiredRole:  "shouyu-user",
		SessionSecret: strings.Repeat("x", 32),
	})
	store := NewSessionStore(time.Hour)
	h := NewHandlers(p, store)

	loginW := httptest.NewRecorder()
	h.Login(loginW, httptest.NewRequest("GET", "/oauth/login", nil))
	stateCookie := loginW.Result().Cookies()[0]
	cbReq := httptest.NewRequest("GET", "/oauth/callback?state="+stateCookie.Value+"&code=fake", nil)
	cbReq.AddCookie(stateCookie)
	cbW := httptest.NewRecorder()
	h.Callback(cbW, cbReq)

	if cbW.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", cbW.Code, cbW.Body.String())
	}
}

func TestLogout_RedirectsToEndSessionWithIDTokenHint(t *testing.T) {
	kc, _ := stubKC(t, true)
	defer kc.Close()
	secret := strings.Repeat("x", 32)
	p, _ := NewProvider(context.Background(), Config{
		IssuerURL: kc.URL, ClientID: "shouyu", ClientSecret: "x",
		RedirectURL:   "https://home.yagura.dev/oauth/callback",
		PostLogoutURL: "https://home.yagura.dev/",
		SessionSecret: secret,
		RequiredRole:  "shouyu-user",
	})
	store := NewSessionStore(time.Hour)
	h := NewHandlers(p, store)

	// Simulate completed login: stash a session with a raw ID token.
	id := store.NewID()
	store.Put(id, Session{Username: "declan", RawIDToken: "fake-id-token"})

	req := httptest.NewRequest("GET", "/oauth/logout", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: signCookieValue(id, secret)})
	w := httptest.NewRecorder()
	h.Logout(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", w.Code)
	}
	loc := w.Header().Get("Location")
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("Location not a parseable URL: %v (%q)", err, loc)
	}
	// Base URL must be the end_session_endpoint from discovery, NOT the
	// AuthURL fallback. Stub exposes end_session_endpoint at <kc>/logout.
	if !strings.HasPrefix(loc, kc.URL+"/logout?") {
		t.Fatalf("Location base = %q, want prefix %q", loc, kc.URL+"/logout?")
	}
	q := u.Query()
	if got := q.Get("id_token_hint"); got != "fake-id-token" {
		t.Fatalf("id_token_hint = %q, want %q", got, "fake-id-token")
	}
	if got := q.Get("post_logout_redirect_uri"); got != "https://home.yagura.dev/" {
		t.Fatalf("post_logout_redirect_uri = %q, want portal root", got)
	}
	// Verify session is deleted.
	if _, ok := store.Get(id); ok {
		t.Fatal("session not deleted on logout")
	}
}

func TestLogout_NoSessionStillRedirects(t *testing.T) {
	// If the cookie is missing or stale, Logout should still redirect to the
	// end-session endpoint (just without id_token_hint). UX: a stale tab
	// clicking "logout" shouldn't 500.
	kc, _ := stubKC(t, true)
	defer kc.Close()
	p, _ := NewProvider(context.Background(), Config{
		IssuerURL: kc.URL, ClientID: "shouyu", ClientSecret: "x",
		RedirectURL:   "https://home.yagura.dev/oauth/callback",
		PostLogoutURL: "https://home.yagura.dev/",
		SessionSecret: strings.Repeat("x", 32),
	})
	store := NewSessionStore(time.Hour)
	h := NewHandlers(p, store)

	req := httptest.NewRequest("GET", "/oauth/logout", nil)
	w := httptest.NewRecorder()
	h.Logout(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", w.Code)
	}
	loc := w.Header().Get("Location")
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("Location not parseable: %v", err)
	}
	// id_token_hint MUST be absent when there's no session; we don't want
	// to leak a stale or empty hint that Keycloak then rejects.
	if got := u.Query().Get("id_token_hint"); got != "" {
		t.Fatalf("id_token_hint = %q on no-session logout, want empty", got)
	}
	// post_logout_redirect_uri still threaded through.
	if got := u.Query().Get("post_logout_redirect_uri"); got != "https://home.yagura.dev/" {
		t.Fatalf("post_logout_redirect_uri = %q, want portal root", got)
	}
}

func TestCallback_StoresRawIDToken(t *testing.T) {
	kc, _ := stubKC(t, true)
	defer kc.Close()
	secret := strings.Repeat("x", 32)
	p, _ := NewProvider(context.Background(), Config{
		IssuerURL: kc.URL, ClientID: "shouyu", ClientSecret: "x",
		RedirectURL:   kc.URL + "/cb",
		RequiredRole:  "shouyu-user",
		SessionSecret: secret,
	})
	store := NewSessionStore(time.Hour)
	h := NewHandlers(p, store)

	// Drive a successful login flow (mirrors TestCallback_AcceptsValidUser).
	loginW := httptest.NewRecorder()
	h.Login(loginW, httptest.NewRequest("GET", "/oauth/login", nil))
	stateCookie := loginW.Result().Cookies()[0]
	cbReq := httptest.NewRequest("GET", "/oauth/callback?state="+stateCookie.Value+"&code=fake", nil)
	cbReq.AddCookie(stateCookie)
	cbW := httptest.NewRecorder()
	h.Callback(cbW, cbReq)

	// Extract the session and check RawIDToken is populated.
	var sessCookie *http.Cookie
	for _, c := range cbW.Result().Cookies() {
		if c.Name == sessionCookieName {
			sessCookie = c
		}
	}
	if sessCookie == nil {
		t.Fatal("session cookie not set")
	}
	id, ok := verifyCookieValue(sessCookie.Value, secret)
	if !ok {
		t.Fatalf("session cookie value %q failed HMAC verification", sessCookie.Value)
	}
	sess, ok := store.Get(id)
	if !ok {
		t.Fatal("session not in store")
	}
	if sess.RawIDToken == "" {
		t.Fatal("RawIDToken not stored on session")
	}
}

// helpers below

func mintIDToken(t *testing.T, priv *rsa.PrivateKey, iss, aud, sub string, withRole bool) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss":                iss,
		"aud":                aud,
		"sub":                sub,
		"preferred_username": "declan",
		"name":               "Declan Lee",
		"email":              "declan@example.com",
		"groups":             []string{"family"},
		"exp":                time.Now().Add(time.Hour).Unix(),
		"iat":                time.Now().Unix(),
	}
	if withRole {
		claims["resource_access"] = map[string]any{
			"shouyu": map[string]any{"roles": []string{"shouyu-user"}},
		}
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := tok.SignedString(priv)
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func writeJWKS(w http.ResponseWriter, pub *rsa.PublicKey) {
	// Minimal RSA JWKS encoding; relies on go-oidc's tolerance.
	// (full impl in jwks_test_helpers.go to keep this file readable)
	writeRSAJWKS(w, pub)
}
