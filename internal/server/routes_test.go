package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/OC15141355/shouyu/internal/auth"
	"github.com/OC15141355/shouyu/internal/notes"
)

// TestUnauthedRoutesRedirectToLogin enforces the spec §12 auth-bypass-surface
// invariant: any content route returns 302 to /oauth/login without a session.
// Only /healthz, /readyz, /oauth/* and /static/* are unprotected.
//
// Per-method probing (W4.5 finding): chi v5 returns 405 for method/path
// mismatches BEFORE running group middleware, so `GET /notes` (registered
// only as POST) short-circuits the RequireAuth → 302 path. Per the plan's
// pre-warned fallback, the notes probe uses DELETE /notes/1 — that route
// IS registered, so RequireAuth runs and redirects unauthenticated requests.
func TestUnauthedRoutesRedirectToLogin(t *testing.T) {
	srv := newTestStack(t)
	cases := []struct {
		method     string
		path       string
		wantStatus int
	}{
		{"GET", "/", http.StatusFound},
		{"DELETE", "/notes/1", http.StatusFound},
		{"GET", "/healthz", http.StatusOK},
		{"GET", "/readyz", http.StatusOK},
		{"GET", "/static/style.css", http.StatusOK},
	}
	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.path, nil)
			w := httptest.NewRecorder()
			srv.r.ServeHTTP(w, req)
			if w.Code != c.wantStatus {
				t.Fatalf("%s %s: status = %d, want %d", c.method, c.path, w.Code, c.wantStatus)
			}
		})
	}
}

func TestSecurityHeaders_Present(t *testing.T) {
	srv := newTestStack(t)
	// Probe an unauthed path (responds 200) so we see the response headers,
	// not the redirect.
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	srv.r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	wantHeaders := map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"Referrer-Policy":           "no-referrer",
	}
	for k, want := range wantHeaders {
		if got := w.Header().Get(k); got != want {
			t.Fatalf("%s = %q, want %q", k, got, want)
		}
	}
	csp := w.Header().Get("Content-Security-Policy")
	for _, directive := range []string{"default-src 'self'", "script-src 'self'", "frame-ancestors 'none'"} {
		if !strings.Contains(csp, directive) {
			t.Fatalf("CSP missing directive %q: %q", directive, csp)
		}
	}
}

func newTestStack(t *testing.T) *Server {
	t.Helper()
	// Use minimal Deps with a no-op provider, a fresh store, and a temp tiles file.
	tilesFile := t.TempDir() + "/tiles.yaml"
	if err := writeTilesFixture(tilesFile); err != nil {
		t.Fatal(err)
	}
	staticDir := t.TempDir()
	// Touch a static file so /static/* returns 200
	if err := touchFile(staticDir + "/style.css"); err != nil {
		t.Fatal(err)
	}
	r, _ := newMemNotesRepo(t)
	srv, err := New(Deps{
		Provider:      nil, // unused for unauthed routes
		Sessions:      newEmptySessions(),
		SessionSecret: strings.Repeat("t", 32),
		TilesPath:     tilesFile,
		NotesRepo:     r,
		StaticDir:     staticDir,
		Loc:           time.UTC, // tests use UTC for deterministic greeting bands
	})
	if err != nil {
		t.Fatal(err)
	}
	return srv
}

func writeTilesFixture(path string) error {
	return os.WriteFile(path, []byte("brand:\n  name: Fran\ntiles: []\n"), 0o644)
}

func touchFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	return f.Close()
}

func newMemNotesRepo(t *testing.T) (*notes.Repo, error) {
	t.Helper()
	r, err := notes.NewRepo(":memory:")
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = r.Close() })
	return r, r.Migrate(context.Background())
}

func newEmptySessions() *auth.SessionStore { return auth.NewSessionStore(time.Hour) }
