package hatchery

import (
	"context"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/sdk"
)

var SessionDuration = 365 * 24 * time.Hour

// CheckSigninRequest checks that given driver request is valid for a signin with auth builtin.
func CheckSigninRequest(req sdk.AuthConsumerHatcherySigninRequest) (string, error) {
	if req.Token == "" {
		return "", sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid authentication token")
	}

	// check token like a builtin token
	payload, err := parseSigninConsumerToken(req.Token)
	if err != nil {
		return "", err
	}
	return payload.ConsumerID, err
}

func GetManifest() sdk.AuthDriverManifest {
	return sdk.AuthDriverManifest{
		Type:           sdk.ConsumerHatchery,
		SignupDisabled: true,
	}
}

type signinHatcheryConsumerToken struct {
	ConsumerID string `json:"consumer_id"`
	IAT        int64  `json:"iat"`
}

// NewSigninConsumerToken returns a token to signin with built in consumer.
func NewSigninConsumerToken(c *sdk.AuthHatcheryConsumer) (string, error) {
	latestValidityPeriod := c.ValidityPeriods.Latest()
	payload := signinHatcheryConsumerToken{
		ConsumerID: c.ID,
		IAT:        latestValidityPeriod.IssuedAt.Unix(),
	}
	return authentication.SignJWS(payload, latestValidityPeriod.IssuedAt, latestValidityPeriod.Duration)
}

func parseSigninConsumerToken(signature string) (signinHatcheryConsumerToken, error) {
	var payload signinHatcheryConsumerToken
	if err := authentication.VerifyJWS(signature, &payload); err != nil {
		return payload, sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid signin token"))
	}
	return payload, nil
}

func CheckSigninConsumerTokenIssuedAt(ctx context.Context, signature string, c *sdk.AuthHatcheryConsumer) (string, error) {
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

func checkSigninConsumerTokenIssuedAt(ctx context.Context, payload signinHatcheryConsumerToken, v sdk.AuthConsumerValidityPeriod) (string, error) {
	var eqIAT = payload.IAT == v.IssuedAt.Unix()
	var hasRevoke = v.Duration > 0
	var afterRevoke = time.Now().After(v.IssuedAt.Add(v.Duration))

	if !eqIAT {
		return "", sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid signin token")
	}
	if hasRevoke && afterRevoke {
		return "", sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid signin token")
	}
	return payload.ConsumerID, nil
}
