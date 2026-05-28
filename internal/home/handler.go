package home

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/OC15141355/shouyu/internal/auth"
	"github.com/OC15141355/shouyu/internal/config"
	"github.com/OC15141355/shouyu/internal/greeting"
	"github.com/OC15141355/shouyu/internal/notes"
	"github.com/OC15141355/shouyu/web/templates"
)

type ctxKey int

const authCtxOverride ctxKey = 1 // test-only seam

type Handler struct {
	cfg  *config.Config
	repo *notes.Repo
	loc  *time.Location // greeting/date rendered in this tz; tests pass time.UTC
}

func NewHandler(cfg *config.Config, repo *notes.Repo, loc *time.Location) *Handler {
	return &Handler{cfg: cfg, repo: repo, loc: loc}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sess := getSession(r.Context())
	if sess.Username == "" {
		http.Redirect(w, r, "/oauth/login", http.StatusFound)
		return
	}
	now := time.Now().In(h.loc)
	tiles := h.cfg.FilterByGroups(sess.Groups)
	ns, err := h.repo.ListActive(r.Context(), 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d := templates.HomeData{
		BrandName: h.cfg.Brand.Name,
		Greeting:  greeting.Greet(sess.Username, now),
		Date:      strings.ToLower(now.Format("Mon 2 Jan")),
		Tiles:     tiles,
		Notes:     ns,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.Home(d).Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getSession(ctx context.Context) auth.Session {
	if v, ok := ctx.Value(authCtxOverride).(auth.Session); ok {
		return v
	}
	s, _ := auth.FromContext(ctx)
	return s
}
