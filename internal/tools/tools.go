//go:build tools

// Package tools pins build dependencies that are not yet imported by production
// code. Remove this file once all deps are in use (W4.5+).
package tools

import (
	_ "github.com/a-h/templ"
	_ "github.com/coreos/go-oidc/v3/oidc"
	_ "github.com/go-chi/chi/v5"
	_ "golang.org/x/oauth2"
	_ "gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)
