package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRequireAuth_NoCookie_RedirectsToLogin(t *testing.T) {
	store := NewSessionStore(time.Hour)
	mw := RequireAuth(store, strings.Repeat("x", 32))
	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if called {
		t.Fatal("handler should NOT have been called")
	}
	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", w.Code)
	}
	if w.Header().Get("Location") != "/oauth/login" {
		t.Fatalf("Location = %q", w.Header().Get("Location"))
	}
}

func TestRequireAuth_ValidSession_PassesThrough(t *testing.T) {
	secret := strings.Repeat("x", 32)
	store := NewSessionStore(time.Hour)
	id := store.NewID()
	store.Put(id, Session{Username: "declan"})

	mw := RequireAuth(store, secret)
	var captured *Session
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s, ok := FromContext(r.Context())
		if ok {
			captured = &s
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: signCookieValue(id, secret)})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if captured == nil || captured.Username != "declan" {
		t.Fatalf("session not in context: %+v", captured)
	}
}

func TestRequireAuth_TamperedCookieRedirects(t *testing.T) {
	secret := strings.Repeat("x", 32)
	store := NewSessionStore(time.Hour)
	id := store.NewID()
	store.Put(id, Session{Username: "declan"})
	signed := signCookieValue(id, secret)
	// Flip the last byte of the signature.
	tampered := signed[:len(signed)-1] + string(signed[len(signed)-1]^0x01)

	mw := RequireAuth(store, secret)
	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: tampered})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if called {
		t.Fatal("handler called with tampered cookie")
	}
	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", w.Code)
	}
}
