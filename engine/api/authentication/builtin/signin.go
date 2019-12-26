package builtin

import (
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/sdk"
)

type signinBuiltinConsumerToken struct {
	ConsumerID string `json:"consumer_id"`
	IAT        int64  `json:"iat"`
}

// NewSigninConsumerToken returns a token to signin with built in consumer.
func NewSigninConsumerToken(c *sdk.AuthConsumer) (string, error) {
	payload := signinBuiltinConsumerToken{
		ConsumerID: c.ID,
		IAT:        c.IssuedAt.Unix(),
	}
	return authentication.SignJWS(payload, 0) // 0 means no expiration time
}

func CheckSigninConsumerToken(signature string) (string, int64, error) {
	payload, err := parseSigninConsumerToken(signature)
	if err != nil {
		return "", 0, err
	}
	return payload.ConsumerID, payload.IAT, nil
}

func parseSigninConsumerToken(signature string) (signinBuiltinConsumerToken, error) {
	var payload signinBuiltinConsumerToken
	if err := authentication.VerifyJWS(signature, &payload); err != nil {
		return payload, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid signin token"))
	}
	return payload, nil
}

func CheckSigninConsumerTokenIssuedAt(signature string, iat time.Time) (string, error) {
	payload, err := parseSigninConsumerToken(signature)
	if err != nil {
		return "", err
	}
	iatUnix := iat.Unix()
	if payload.IAT != iatUnix {
		return "", sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid signin token")
	}
	return payload.ConsumerID, nil
}
