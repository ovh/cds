package test

import (
	"context"
	"testing"
	"time"

	"github.com/ovh/cds/sdk"
)

var _ sdk.AuthDriver = new(authDriver)

type authDriver struct {
	t *testing.T
}

// NewDriver returns a new ldap auth driver.
func NewDriver(t *testing.T) sdk.AuthDriver {
	var d = authDriver{t}
	return d
}

func (d authDriver) GetManifest() sdk.AuthDriverManifest {
	return sdk.AuthDriverManifest{
		Type:           sdk.ConsumerTest,
		SignupDisabled: false,
	}
}

func (d authDriver) GetSessionDuration(__ sdk.AuthDriverUserInfo, _ sdk.AuthConsumer) time.Duration {
	return time.Hour * 24 * 30 // 1 month session
}

func (d authDriver) CheckSigninRequest(req sdk.AuthConsumerSigninRequest) error {
	if bind, ok := req["username"]; !ok || bind == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid bind term for ldap signin")
	}
	return nil
}

func (d authDriver) GetUserInfo(ctx context.Context, req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	var u sdk.AuthDriverUserInfo
	var username = req["username"]

	u.ExternalID = username
	u.Username = username
	u.Email = username + "@planet-express.futurama"

	switch username {
	case "fry":
		u.Fullname = "Philip J. Fry"
	case "philip.fry":
		u.Fullname = "Philip J. Fry"
		u.Email = "fry@planet-express.futurama"
	case "leela":
		u.Fullname = "Turanga Leela"
	case "bender":
		u.Fullname = "Bender Bending Rodriguez"
	case "farnsworth":
		u.Fullname = "Professor Hubert J. Farnsworth"
	case "amy":
		u.Fullname = "Amy Wong"
	case "zoidberg":
		u.Fullname = "Dr. John A. Zoidberg"
	default:
		return u, sdk.WithStack(sdk.ErrUnauthorized)
	}

	return u, nil
}
