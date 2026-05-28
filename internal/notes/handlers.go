package notes

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/OC15141355/shouyu/internal/auth"
	"github.com/go-chi/chi/v5"
)

// authSessionCtxKey is a synthetic context key used only by tests to inject a
// Session without going through the OIDC login flow. Production requests carry
// the Session via auth.RequireAuth's own (unexported) ctx key; authorFromCtx
// reads that via auth.FromContext.
type ctxKeyForTest int

const authSessionCtxKey ctxKeyForTest = 99

type Handlers struct {
	repo *Repo
}

func NewHandlers(r *Repo) *Handlers { return &Handlers{repo: r} }

// Post handles POST /notes (htmx form). Returns the updated list partial.
func (h *Handlers) Post(w http.ResponseWriter, r *http.Request) {
	body := strings.TrimSpace(r.FormValue("body"))
	if body == "" {
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}
	if len(body) > 500 {
		http.Error(w, "body too long (max 500)", http.StatusBadRequest)
		return
	}
	author := authorFromCtx(r.Context())
	if author == "" {
		http.Error(w, "no session", http.StatusUnauthorized)
		return
	}
	if _, err := h.repo.Add(r.Context(), body, author); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.renderList(w, r)
}

// Delete handles DELETE /notes/{id}. Returns the updated list partial.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.renderList(w, r)
}

func (h *Handlers) renderList(w http.ResponseWriter, r *http.Request) {
	notes, err := h.repo.ListActive(r.Context(), 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Minimal HTML for v1 — templ rendering wired in W4.
	fmt.Fprint(w, `<div id="notes-list">`)
	for _, n := range notes {
		fmt.Fprintf(w, `<div class="note">%s<div class="who">%s · %s</div></div>`,
			escape(n.Body), escape(n.Author), n.CreatedAt.Format("Jan 2"))
	}
	fmt.Fprint(w, `</div>`)
}

func authorFromCtx(ctx context.Context) string {
	// Test seam: tests inject Session directly under authSessionCtxKey to
	// avoid the OIDC login dance. Production code path falls through to
	// auth.FromContext, which reads the key set by auth.RequireAuth.
	if v, ok := ctx.Value(authSessionCtxKey).(auth.Session); ok {
		return v.Username
	}
	if s, ok := auth.FromContext(ctx); ok {
		return s.Username
	}
	return ""
}

func escape(s string) string {
	return strings.NewReplacer("<", "&lt;", ">", "&gt;", "&", "&amp;", `"`, "&quot;").Replace(s)
}
