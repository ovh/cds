package authentication

import (
	"time"
)

const signinConsumerTokenDuration time.Duration = time.Minute * 5

type signinConsumerToken struct {
	Origin string `json:"origin"`
}

// NewSigninStateToken returns a jws used for signin request.
func NewSigninStateToken(origin string) (string, error) {
	payload := signinConsumerToken{Origin: origin}
	return SignJWS(payload, signinConsumerTokenDuration)
}

// CheckSigninStateToken checks if a given signature is a valid signin state.
func CheckSigninStateToken(signature string) error {
	var payload signinConsumerToken
	return VerifyJWS(signature, &payload)
}
