package auth

import (
	"context"
	"net/http"
)

type ctxKey int

const sessionKey ctxKey = 1

// RequireAuth is HTTP middleware that ensures a valid HMAC-signed session
// cookie is present. Missing/invalid/tampered cookie → 302 to /oauth/login.
// Valid → request continues with the Session attached to ctx (retrieve via
// FromContext).
//
// secret is the HMAC-SHA256 key the cookie was signed with (Config.SessionSecret).
func RequireAuth(store *SessionStore, secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(sessionCookieName)
			if err != nil {
				http.Redirect(w, r, "/oauth/login", http.StatusFound)
				return
			}
			id, ok := verifyCookieValue(c.Value, secret)
			if !ok {
				http.Redirect(w, r, "/oauth/login", http.StatusFound)
				return
			}
			sess, ok := store.Get(id)
			if !ok {
				http.Redirect(w, r, "/oauth/login", http.StatusFound)
				return
			}
			ctx := context.WithValue(r.Context(), sessionKey, sess)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// FromContext extracts the Session attached by RequireAuth.
func FromContext(ctx context.Context) (Session, bool) {
	s, ok := ctx.Value(sessionKey).(Session)
	return s, ok
}
