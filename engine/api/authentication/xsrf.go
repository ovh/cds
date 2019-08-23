package authentication

import (
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

// XSRFTokenDuration is set to 10 minutes.
var XSRFTokenDuration = 60 * 10

// NewSessionXSRFToken generate and store a XSRF token for a given session id.
func NewSessionXSRFToken(store cache.Store, sessionID string) string {
	var XSRFToken = sdk.UUID()
	var k = cache.Key("token", "xsrf", sessionID)
	store.SetWithTTL(k, &XSRFToken, XSRFTokenDuration)
	return XSRFToken
}

// GetSessionXSRFToken returns a XSRF token from cache if exists for given session.
func GetSessionXSRFToken(store cache.Store, sessionID string) (string, bool) {
	var XSRFToken string
	var k = cache.Key("token", "xsrf", sessionID)
	if store.Get(k, &XSRFToken) {
		return XSRFToken, true
	}
	return "", false
}
