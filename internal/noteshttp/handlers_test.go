package noteshttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/OC15141355/shouyu/internal/auth"
	"github.com/OC15141355/shouyu/internal/notes"
	"github.com/go-chi/chi/v5"
)

func newTestRepo(t *testing.T) *notes.Repo {
	t.Helper()
	r, err := notes.NewRepo(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return r
}

func newTestServer(t *testing.T) (*Handlers, *notes.Repo) {
	t.Helper()
	r := newTestRepo(t)
	return NewHandlers(r), r
}

func TestPostNote_AddsAndReturnsList(t *testing.T) {
	h, repo := newTestServer(t)
	form := url.Values{"body": []string{"Milk please"}}
	req := httptest.NewRequest("POST", "/notes", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = withSession(req, "cam")
	w := httptest.NewRecorder()
	h.Post(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Milk please") {
		t.Fatalf("body missing note: %s", w.Body.String())
	}
	ns, _ := repo.ListActive(context.Background(), 10)
	if len(ns) != 1 {
		t.Fatalf("repo count = %d", len(ns))
	}
}

func TestPostNote_RejectsEmpty(t *testing.T) {
	h, _ := newTestServer(t)
	form := url.Values{"body": []string{"   "}}
	req := httptest.NewRequest("POST", "/notes", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = withSession(req, "cam")
	w := httptest.NewRecorder()
	h.Post(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d", w.Code)
	}
}

func TestDeleteNote_RemovesAndReturnsEmpty(t *testing.T) {
	h, repo := newTestServer(t)
	// Author and session match — the only path now allowed past the P0-4 gate.
	id, _ := repo.Add(context.Background(), "x", "y")
	req := httptest.NewRequest("DELETE", "/notes/"+strconv.FormatInt(id, 10), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", strconv.FormatInt(id, 10))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = withSession(req, "y")
	w := httptest.NewRecorder()
	h.Delete(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	ns, _ := repo.ListActive(context.Background(), 10)
	if len(ns) != 0 {
		t.Fatalf("not deleted: %+v", ns)
	}
}

func TestDeleteNote_RejectsNonAuthor(t *testing.T) {
	h, repo := newTestServer(t)
	// Alice creates a note.
	id, err := repo.Add(context.Background(), "alice's note", "alice")
	if err != nil {
		t.Fatal(err)
	}
	// Bob tries to delete it.
	req := httptest.NewRequest("DELETE", "/notes/"+strconv.FormatInt(id, 10), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", strconv.FormatInt(id, 10))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = withSession(req, "bob")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	// Alice's note is still there.
	ns, _ := repo.ListActive(context.Background(), 10)
	if len(ns) != 1 {
		t.Fatalf("alice's note was deleted — bob bypassed the gate: %+v", ns)
	}
}

func TestDeleteNote_AcceptsAuthor(t *testing.T) {
	h, repo := newTestServer(t)
	id, err := repo.Add(context.Background(), "alice's note", "alice")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("DELETE", "/notes/"+strconv.FormatInt(id, 10), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", strconv.FormatInt(id, 10))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = withSession(req, "alice")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	ns, _ := repo.ListActive(context.Background(), 10)
	if len(ns) != 0 {
		t.Fatalf("alice's note not deleted: %+v", ns)
	}
}

func TestDeleteNote_NotFoundReturns404(t *testing.T) {
	h, _ := newTestServer(t)
	req := httptest.NewRequest("DELETE", "/notes/9999", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "9999")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = withSession(req, "alice")
	w := httptest.NewRecorder()
	h.Delete(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func withSession(r *http.Request, username string) *http.Request {
	ctx := context.WithValue(r.Context(), authSessionCtxKey, auth.Session{Username: username})
	return r.WithContext(ctx)
}
