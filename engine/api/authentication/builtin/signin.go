package builtin

import (
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/sdk"
)

type signinBuiltinConsumerToken struct {
	ConsumerID string
	Nonce      int64
}

// newSigninConsumerToken returns a token to signin with built in consumer.
func newSigninConsumerToken(c *sdk.AuthConsumer) (string, error) {
	payload := signinBuiltinConsumerToken{
		ConsumerID: c.ID,
		Nonce:      time.Now().Unix(),
	}
	return authentication.SignJWS(payload, 0) // 0 means no expiration time
}

func checkSigninConsumerToken(signature string) (string, error) {
	var payload signinBuiltinConsumerToken
	if err := authentication.VerifyJWS(signature, &payload); err != nil {
		return "", sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid signin token"))
	}
	return payload.ConsumerID, nil
}
