package authentication

import (
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

// NewSessionXSRFToken generate and store a XSRF token for a given session id.
func NewSessionXSRFToken(store cache.Store, sessionID string, sessionExpirationDelaySecond int) (string, error) {
	var XSRFToken = sdk.UUID()
	var k = cache.Key("token", "xsrf", sessionID)
	if err := store.SetWithTTL(k, &XSRFToken, sessionExpirationDelaySecond); err != nil {
		return "", err
	}
	return XSRFToken, nil
}

// GetSessionXSRFToken returns a XSRF token from cache if exists for given session.
func GetSessionXSRFToken(store cache.Store, sessionID string) (string, bool) {
	var XSRFToken string
	var k = cache.Key("token", "xsrf", sessionID)
	if has, _ := store.Get(k, &XSRFToken); has {
		return XSRFToken, true
	}
	return "", false
}
