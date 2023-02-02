package authentication

import (
	"time"

	"github.com/ovh/cds/sdk"
)

// NewDefaultSigninStateToken returns a jws used for signin request.
func NewDefaultSigninStateToken(signinState sdk.AuthSigninConsumerToken) (string, error) {
	var now = time.Now()
	payload := sdk.AuthSigninConsumerToken{
		Origin:            signinState.Origin,
		RedirectURI:       signinState.RedirectURI,
		IssuedAt:          now.Unix(),
		IsFirstConnection: signinState.IsFirstConnection,
		LinkUser:          signinState.LinkUser,
	}
	return SignJWS(payload, now, sdk.AuthSigninConsumerTokenDuration)
}

// CheckDefaultSigninStateToken checks if a given signature is a valid signin state.
func CheckDefaultSigninStateToken(signature string) error {
	var payload sdk.AuthSigninConsumerToken
	return VerifyJWS(signature, &payload)
}
