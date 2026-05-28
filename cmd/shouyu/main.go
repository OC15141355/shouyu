package main

import (
	"log"
	"net/http"
	"os"

	// Blank imports pin core deps in go.mod until W1+ wires real imports.
	// Remove these as each dep gets a real consumer.
	_ "github.com/a-h/templ"
	_ "github.com/coreos/go-oidc/v3/oidc"
	_ "github.com/go-chi/chi/v5"
	_ "golang.org/x/oauth2"
	_ "gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	log.Printf("shouyu listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
