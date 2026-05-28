package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
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
		Provider:  nil, // unused for unauthed routes
		Sessions:  newEmptySessions(),
		TilesPath: tilesFile,
		NotesRepo: r,
		StaticDir: staticDir,
		Loc:       time.UTC, // tests use UTC for deterministic greeting bands
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
