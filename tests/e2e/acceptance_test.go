//go:build e2e

package e2e

import (
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
)

// TestAcceptance_LoginRenderAddDeleteLogout exercises the whole flow
// using a stub OIDC provider. Requires running with -tags=e2e and
// the same test helpers from internal/auth.
func TestAcceptance_LoginRenderAddDeleteLogout(t *testing.T) {
	t.Skip("scaffold; fully wire after deployment manifests land — see W9 followups")
	_ = cookiejar.New
	_ = url.QueryEscape
	_ = strings.NewReader
}
