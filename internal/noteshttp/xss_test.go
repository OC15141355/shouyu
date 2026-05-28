package noteshttp

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/OC15141355/shouyu/web/templates"
)

func TestNotesListEscapesHTML(t *testing.T) {
	r := newTestRepo(t)
	_, _ = r.Add(context.Background(), "<script>alert(1)</script>", "x")
	ns, err := r.ListActive(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := templates.NotesList(ns).Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}
	body := buf.String()
	if strings.Contains(body, "<script>") {
		t.Fatalf("UNESCAPED <script> in output: %s", body)
	}
	// templ's EscapeString (stdlib html.EscapeString) escapes `<` → `&lt;`.
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Fatalf("expected escaped <script>; got: %s", body)
	}
}
