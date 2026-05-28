# shouyu

OIDC-aware family portal. Single Go binary, htmx + templ for UI, SQLite for state.

Deployed under brand label "Fran" at <https://home.yagura.dev>. Sibling brand labels are supported via the `PORTAL_BRAND` env var; same codebase.

## Stack

- Go 1.23+, `net/http` + `chi` router
- `github.com/coreos/go-oidc/v3` for Keycloak OIDC
- `templ` for typed HTML templates, htmx for inline interactivity
- `modernc.org/sqlite` (pure-Go) for the notes store
- `gcr.io/distroless/static` base image (~25 MB total)

## Status

v1 in progress (Phase 2c of the Yagura homelab initiative).
