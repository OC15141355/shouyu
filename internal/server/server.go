package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/OC15141355/shouyu/internal/auth"
	"github.com/OC15141355/shouyu/internal/config"
	"github.com/OC15141355/shouyu/internal/home"
	"github.com/OC15141355/shouyu/internal/notes"
	"github.com/OC15141355/shouyu/internal/noteshttp"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Deps struct {
	Provider *auth.Provider
	Sessions *auth.SessionStore
	// SessionSecret is the HMAC key used by RequireAuth to verify the
	// session cookie. Threaded explicitly (rather than read from
	// Provider.Config()) so tests can construct a stack without a live
	// OIDC Provider. Production wires it from auth.Config.SessionSecret.
	SessionSecret string
	TilesPath     string
	NotesRepo     *notes.Repo
	StaticDir     string         // path to web/static
	Loc           *time.Location // tz for greeting/date rendering (W4.4 pre-survey finding)
}

type Server struct {
	r    *chi.Mux
	deps Deps
}

// securityHeaders sets a baseline set of defense-in-depth response headers on
// every request. CSP is tight ('self'-only) — matches v1's static asset
// surface (one templ-rendered page, vendored htmx.min.js, vendored style.css,
// no third-party resources). Loosen only when a real cross-origin asset
// shows up.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h.Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; frame-ancestors 'none'")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func New(deps Deps) (*Server, error) {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(securityHeaders)

	authH := auth.NewHandlers(deps.Provider, deps.Sessions)
	notesH := noteshttp.NewHandlers(deps.NotesRepo)

	// Reload tiles per request (spec §12 simplicity choice). Cheap; YAML is tiny.
	tilesLoader := func() *config.Config {
		c, err := config.Load(deps.TilesPath)
		if err != nil {
			// Log + return empty so the page partially renders (greeting +
			// notes intact) rather than crashing. Operator sees the error in
			// Loki/stdout. Silent fallback would hide a real-world surface
			// (tiles ConfigMap broken in production → "where did my tiles go").
			// W4.5 pre-survey finding — do NOT strip this log as noise.
			log.Printf("server: config.Load(%s): %v", deps.TilesPath, err)
			return &config.Config{}
		}
		return c
	}

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })

	r.Get("/oauth/login", authH.Login)
	r.Get("/oauth/callback", authH.Callback)
	r.Get("/oauth/logout", authH.Logout)

	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(deps.StaticDir))))

	r.Group(func(r chi.Router) {
		r.Use(auth.RequireAuth(deps.Sessions, deps.SessionSecret))
		r.Get("/", func(w http.ResponseWriter, req *http.Request) {
			home.NewHandler(tilesLoader(), deps.NotesRepo, deps.Loc).ServeHTTP(w, req)
		})
		r.Post("/notes", notesH.Post)
		r.Delete("/notes/{id}", notesH.Delete)
	})

	return &Server{r: r, deps: deps}, nil
}

func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.r,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
