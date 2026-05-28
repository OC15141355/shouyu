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
	Provider  *auth.Provider
	Sessions  *auth.SessionStore
	TilesPath string
	NotesRepo *notes.Repo
	StaticDir string         // path to web/static
	Loc       *time.Location // tz for greeting/date rendering (W4.4 pre-survey finding)
}

type Server struct {
	r    *chi.Mux
	deps Deps
}

func New(deps Deps) (*Server, error) {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)

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
		r.Use(auth.RequireAuth(deps.Sessions))
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
		ReadHeaderTimeout: 10 * time.Second,
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
