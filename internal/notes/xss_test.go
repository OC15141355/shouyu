package notes

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRenderListEscapesHTML(t *testing.T) {
	r := newTestRepo(t)
	_, _ = r.Add(context.Background(), "<script>alert(1)</script>", "x")
	h := NewHandlers(r)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	h.renderList(w, req)
	body := w.Body.String()
	if strings.Contains(body, "<script>") {
		t.Fatalf("UNESCAPED <script> in output: %s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Fatalf("expected escaped <script>; got: %s", body)
	}
}
