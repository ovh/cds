package builtin

import (
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/sdk"
)

type signinBuiltinConsumerToken struct {
	ConsumerID string
	IAT        int64
}

// NewSigninConsumerToken returns a token to signin with built in consumer.
func NewSigninConsumerToken(c *sdk.AuthConsumer) (string, error) {
	payload := signinBuiltinConsumerToken{
		ConsumerID: c.ID,
		IAT:        c.IssuedAt.Unix(),
	}
	return authentication.SignJWS(payload, 0) // 0 means no expiration time
}

func CheckSigninConsumerToken(signature string) (string, error) {
	return CheckSigninConsumerTokenIssuedAt(signature, time.Time{})
}

func CheckSigninConsumerTokenIssuedAt(signature string, iat time.Time) (string, error) {
	var payload signinBuiltinConsumerToken
	if err := authentication.VerifyJWS(signature, &payload); err != nil {
		return "", sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid signin token"))
	}
	if !iat.IsZero() {
		iatUnix := iat.Unix()
		if payload.IAT != iatUnix {
			return "", sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid signin token")
		}
	}
	return payload.ConsumerID, nil
}
