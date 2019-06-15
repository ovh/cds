package authentication

import (
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var XSRFTokenDuration = 60 * 10 // 10 minutes

// NewSessionXSRFToken generate and store a XSRF token for a given session id.
func NewSessionXSRFToken(store cache.Store, sessionID string) string {
	log.Debug("authentication.StoreXSRFToken")
	var xsrfToken = sdk.UUID()
	var k = cache.Key("token", "xsrf", sessionID)
	store.SetWithTTL(k, &xsrfToken, XSRFTokenDuration)
	return xsrfToken
}

// CheckSessionXSRFToken checks a value "xsrfToken" against the session XSRF in cache for given session id.
func CheckSessionXSRFToken(store cache.Store, sessionID, xsrfToken string) bool {
	log.Debug("authentication.CheckXSRFToken")
	var expectedXSRFfToken string
	var k = cache.Key("token", "xsrf", sessionID)
	if store.Get(k, &expectedXSRFfToken) {
		return expectedXSRFfToken == xsrfToken
	}
	return false
}
