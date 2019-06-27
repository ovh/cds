package builtin

import (
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var _ sdk.AuthDriver = new(AuthDriver)

// NewDriver returns a new initialized driver for builtin authentication.
func NewDriver() sdk.AuthDriver {
	return &AuthDriver{}
}

// AuthDriver for builtin authentication.
type AuthDriver struct{}

func (d AuthDriver) GetManifest() sdk.AuthDriverManifest {
	return sdk.AuthDriverManifest{
		Type:           sdk.ConsumerBuiltin,
		SignupDisabled: true,
	}
}

func (d AuthDriver) GetSigninURI(state string) string {
	return "/"
}

func (d AuthDriver) GetSessionDuration() time.Duration {
	return time.Hour // 1 hour session
}

// CheckSigninRequest checks that given driver request is valid for a signin with auth builtin.
func (d AuthDriver) CheckSigninRequest(req sdk.AuthConsumerSigninRequest) error {
	token, has := req["token"]
	if !has {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid authentication token")
	}

	var builtinConsumerAuthenticationToken builtinConsumerAuthenticationToken
	if err := authentication.VerifyJWS(token, &builtinConsumerAuthenticationToken); err != nil {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid authentication token: %v", err)
	}

	return nil
}

func (d AuthDriver) GetUserInfo(req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	token, has := req["token"]
	if !has {
		return sdk.AuthDriverUserInfo{}, sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid authentication token")
	}

	var builtinConsumerAuthenticationToken builtinConsumerAuthenticationToken
	if err := authentication.VerifyJWS(token, &builtinConsumerAuthenticationToken); err != nil {
		return sdk.AuthDriverUserInfo{}, sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid authentication token: %v", err)
	}

	log.Debug("builtin.GetUserInfo> %s", builtinConsumerAuthenticationToken.ConsumerID)

	return sdk.AuthDriverUserInfo{
		ExternalID: builtinConsumerAuthenticationToken.ConsumerID,
	}, nil
}

type builtinConsumerAuthenticationToken struct {
	ConsumerID string
	Nonce      int64
}

func GetAuthenticationToken(c *sdk.AuthConsumer) (string, error) {
	var builtinConsumerAuthenticationToken = builtinConsumerAuthenticationToken{
		ConsumerID: c.ID,
		Nonce:      time.Now().Unix(),
	}
	return authentication.SignJWS(builtinConsumerAuthenticationToken)
}

// NewConsumer returns a new builtin consumer for given data.
func NewConsumer(db gorp.SqlExecutor, name, description, userID string, groupIDs []int64, scopes []string) (*sdk.AuthConsumer, string, error) {
	c := sdk.AuthConsumer{
		Name:               name,
		Description:        description,
		AuthentifiedUserID: userID,
		Type:               sdk.ConsumerBuiltin,
		Data:               map[string]string{},
		GroupIDs:           groupIDs,
		Scopes:             scopes,
	}

	if err := authentication.InsertConsumer(db, &c); err != nil {
		return nil, "", err
	}

	jws, err := GetAuthenticationToken(&c)
	if err != nil {
		return nil, "", err
	}

	return &c, jws, nil
}
