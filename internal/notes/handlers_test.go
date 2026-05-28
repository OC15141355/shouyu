package notes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/OC15141355/shouyu/internal/auth"
	"github.com/go-chi/chi/v5"
)

func newTestServer(t *testing.T) (*Handlers, *Repo) {
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
	notes, _ := repo.ListActive(context.Background(), 10)
	if len(notes) != 1 {
		t.Fatalf("repo count = %d", len(notes))
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
	id, _ := repo.Add(context.Background(), "x", "y")
	req := httptest.NewRequest("DELETE", "/notes/"+strconv.FormatInt(id, 10), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", strconv.FormatInt(id, 10))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = withSession(req, "z")
	w := httptest.NewRecorder()
	h.Delete(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	notes, _ := repo.ListActive(context.Background(), 10)
	if len(notes) != 0 {
		t.Fatalf("not deleted: %+v", notes)
	}
}

func withSession(r *http.Request, username string) *http.Request {
	ctx := context.WithValue(r.Context(), authSessionCtxKey, auth.Session{Username: username})
	return r.WithContext(ctx)
}
