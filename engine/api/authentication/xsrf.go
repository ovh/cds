package authentication

import (
	"context"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

// NewSessionXSRFToken generate and store a XSRF token for a given session id.
func NewSessionXSRFToken(ctx context.Context, store cache.Store, sessionID string, sessionExpirationDelaySecond int) (string, error) {
	var XSRFToken = sdk.UUID()
	var k = cache.Key("token", "xsrf", sessionID)
	if err := store.SetWithTTL(ctx, k, &XSRFToken, sessionExpirationDelaySecond); err != nil {
		return "", err
	}
	return XSRFToken, nil
}

// GetSessionXSRFToken returns a XSRF token from cache if exists for given session.
func GetSessionXSRFToken(ctx context.Context, store cache.Store, sessionID string) (string, bool) {
	var XSRFToken string
	var k = cache.Key("token", "xsrf", sessionID)
	if has, _ := store.Get(ctx, k, &XSRFToken); has {
		return XSRFToken, true
	}
	return "", false
}
