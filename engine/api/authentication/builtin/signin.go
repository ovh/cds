package builtin

import (
	"context"
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
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

func CheckSigninConsumerTokenIssuedAt(ctx context.Context, signature string, c *sdk.AuthConsumer) (string, error) {
	payload, err := parseSigninConsumerToken(signature)
	if err != nil {
		return "", err
	}
	for _, period := range c.ValidityPeriods {
		s, err := checkSigninConsumerTokenIssuedAt(ctx, payload, period)
		if err == nil {
			return s, nil
		} else {
			log.Debug(ctx, "payload IAT %q is not valid in %+v: %v", payload.IAT, period, err)
		}
	}
	return "", sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid signin token")
}

func checkSigninConsumerTokenIssuedAt(ctx context.Context, payload signinBuiltinConsumerToken, v sdk.AuthConsumerValidityPeriod) (string, error) {
	var eqIAT = payload.IAT == v.IssuedAt.Unix()
	var hasRevoke = v.Duration > 0
	var beforeRevoke = time.Now().Before(v.IssuedAt.Add(v.Duration))
	var eqRevoke = time.Now().Equal(v.IssuedAt.Add(v.Duration))

	if !eqIAT {
		return "", sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid signin token")
	}
	if hasRevoke && !beforeRevoke && !eqRevoke {
		return "", sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid signin token")
	}
	return payload.ConsumerID, nil
}
