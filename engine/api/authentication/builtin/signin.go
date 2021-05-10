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
	latestValidityPeriod := c.ValidityPeriods.Latest()
	payload := signinBuiltinConsumerToken{
		ConsumerID: c.ID,
		IAT:        latestValidityPeriod.IssuedAt.Unix(),
	}
	return authentication.SignJWS(payload, latestValidityPeriod.Duration)
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

func CheckSigninConsumerTokenIssuedAt(signature string, c *sdk.AuthConsumer) (string, error) {
	payload, err := parseSigninConsumerToken(signature)
	if err != nil {
		return "", err
	}
	for _, period := range c.ValidityPeriods {
		s, err := checkSigninConsumerTokenIssuedAt(payload, period)
		if err == nil {
			return s, nil
		}
	}
	return "", sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid signin token")
}

func checkSigninConsumerTokenIssuedAt(payload signinBuiltinConsumerToken, v sdk.AuthConsumerValidityPeriod) (string, error) {
	var eqIAT = time.Unix(payload.IAT, 0).Equal(v.IssuedAt)
	var afterIAT = time.Unix(payload.IAT, 0).After(v.IssuedAt)
	var hasRevoke = v.Duration > 0
	var beforeRevoke = time.Unix(payload.IAT, 0).Before(v.IssuedAt.Add(v.Duration))
	var eqRevoke = time.Unix(payload.IAT, 0).Equal(v.IssuedAt.Add(v.Duration))

	if v.Revoked {
		return "", sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid signin token")
	}
	if !eqIAT && !afterIAT {
		return "", sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid signin token")
	}
	if hasRevoke && !beforeRevoke && !eqRevoke {
		return "", sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid signin token")
	}
	return payload.ConsumerID, nil
}
