package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Embed the IANA zone db in the binary so distroless/static works
	// without a tzdata layer. ~500KB binary cost. W4.4 pre-survey finding.
	_ "time/tzdata"

	"github.com/OC15141355/shouyu/internal/auth"
	"github.com/OC15141355/shouyu/internal/notes"
	"github.com/OC15141355/shouyu/internal/server"
)

func main() {
	cfg := auth.Config{
		IssuerURL:     mustEnv("OPENID_ISSUER_URL"),
		ClientID:      mustEnv("OPENID_CLIENT_ID"),
		ClientSecret:  mustEnv("OPENID_CLIENT_SECRET"),
		RedirectURL:   mustEnv("OPENID_CALLBACK_URL"),
		RequiredRole:  os.Getenv("OPENID_REQUIRED_ROLE"),
		SessionSecret: mustEnv("SESSION_SECRET"),
	}
	loc, err := time.LoadLocation(envOr("PORTAL_TZ", "Australia/Sydney"))
	if err != nil {
		log.Fatalf("time.LoadLocation: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	provider, err := auth.NewProvider(ctx, cfg)
	if err != nil {
		log.Fatalf("auth.NewProvider: %v", err)
	}
	sessions := auth.NewSessionStore(24 * time.Hour)

	dbPath := envOr("DB_PATH", "/data/shouyu.db")
	repo, err := notes.NewRepo(dbPath)
	if err != nil {
		log.Fatalf("notes.NewRepo: %v", err)
	}
	defer func() { _ = repo.Close() }()
	if err := repo.Migrate(ctx); err != nil {
		log.Fatalf("notes.Migrate: %v", err)
	}

	srv, err := server.New(server.Deps{
		Provider:  provider,
		Sessions:  sessions,
		TilesPath: envOr("TILES_PATH", "/etc/shouyu/tiles.yaml"),
		NotesRepo: repo,
		StaticDir: envOr("STATIC_DIR", "web/static"),
		Loc:       loc,
	})
	if err != nil {
		log.Fatalf("server.New: %v", err)
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	go func() { <-sigc; cancel() }()

	addr := ":" + envOr("PORT", "8080")
	log.Printf("shouyu listening on %s", addr)
	if err := srv.ListenAndServe(ctx, addr); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("required env %s not set", k)
	}
	return v
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
