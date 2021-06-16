package authentication

import (
	"time"

	"github.com/ovh/cds/sdk"
)

// NewDefaultSigninStateToken returns a jws used for signin request.
func NewDefaultSigninStateToken(origin, redirectURI string, isFirstConnection bool) (string, error) {
	var now = time.Now()
	payload := sdk.AuthSigninConsumerToken{
		Origin:            origin,
		RedirectURI:       redirectURI,
		IssuedAt:          now.Unix(),
		IsFirstConnection: isFirstConnection,
	}
	return SignJWS(payload, now, sdk.AuthSigninConsumerTokenDuration)
}

// CheckDefaultSigninStateToken checks if a given signature is a valid signin state.
func CheckDefaultSigninStateToken(signature string) error {
	var payload sdk.AuthSigninConsumerToken
	return VerifyJWS(signature, &payload)
}
