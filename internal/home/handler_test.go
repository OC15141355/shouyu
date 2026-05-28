package home

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/OC15141355/shouyu/internal/auth"
	"github.com/OC15141355/shouyu/internal/config"
	"github.com/OC15141355/shouyu/internal/notes"
)

func TestHomeRenders(t *testing.T) {
	repo, _ := notes.NewRepo(":memory:")
	t.Cleanup(func() { _ = repo.Close() })
	_ = repo.Migrate(context.Background())
	_, _ = repo.Add(context.Background(), "Milk please", "cam")

	cfg := &config.Config{
		Brand: config.Brand{Name: "Fran"},
		Tiles: []config.Tile{
			{ID: "fran", Name: "Fran", Href: "https://fran.yagura.dev", VisibleToGroups: []string{"family"}},
			{ID: "jelly", Name: "Jellyfin", Href: "https://jellyfin.yagura.dev", VisibleToGroups: []string{"family"}},
			{ID: "admin", Name: "Admin", Href: "https://admin.yagura.dev", VisibleToGroups: []string{"admin"}},
		},
	}
	// Tests pass time.UTC for deterministic greeting bands.
	h := NewHandler(cfg, repo, time.UTC)
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), authCtxOverride, auth.Session{Username: "declan", Groups: []string{"family"}})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"declan", "Fran", "Jellyfin", "Milk please", "cam"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in body", want)
		}
	}
	if strings.Contains(body, "Admin") {
		t.Fatal("admin tile leaked to family user")
	}
}

func TestHomeRenders_UsesBrandFromConfig(t *testing.T) {
	repo, _ := notes.NewRepo(":memory:")
	t.Cleanup(func() { _ = repo.Close() })
	_ = repo.Migrate(context.Background())

	cfg := &config.Config{
		Brand: config.Brand{Name: "Mochi"},
		Tiles: []config.Tile{},
	}
	h := NewHandler(cfg, repo, time.UTC)
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), authCtxOverride, auth.Session{Username: "declan", Groups: []string{"family"}})
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "<title>Mochi</title>") {
		t.Fatalf("title not 'Mochi': %s", body)
	}
	if strings.Contains(body, "<title>Fran</title>") {
		t.Fatal("hard-coded 'Fran' leaked into title")
	}
}

// TestHomeRedirectsWithoutSession proves the if-no-session-then-redirect branch
// is genuinely live — mirrors W1.4's TestRequireAuth_NoCookie_RedirectsToLogin
// discipline. The last line of defense if RequireAuth ever gets bypassed
// (refactor mistake, middleware misorder).
func TestHomeRedirectsWithoutSession(t *testing.T) {
	repo, _ := notes.NewRepo(":memory:")
	t.Cleanup(func() { _ = repo.Close() })
	_ = repo.Migrate(context.Background())

	h := NewHandler(&config.Config{}, repo, time.UTC)
	req := httptest.NewRequest("GET", "/", nil)
	// NO session in context.
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", w.Code)
	}
	if got := w.Header().Get("Location"); got != "/oauth/login" {
		t.Fatalf("Location = %q, want /oauth/login", got)
	}
}
