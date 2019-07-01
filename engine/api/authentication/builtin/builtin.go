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

func (d AuthDriver) GetUserInfo(req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	var userInfo sdk.AuthDriverUserInfo

	token, has := req["token"]
	if !has {
		return userInfo, sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid authentication token")
	}

	consumerID, err := CheckSigninConsumerToken(token)
	if err != nil {
		return userInfo, err
	}

	log.Debug("builtin.GetUserInfo> %s", consumerID)

	return sdk.AuthDriverUserInfo{
		ExternalID: consumerID,
	}, nil
}

// CheckSigninRequest checks that given driver request is valid for a signin with auth builtin.
func (d AuthDriver) CheckSigninRequest(req sdk.AuthConsumerSigninRequest) error {
	token, has := req["token"]
	if !has {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid authentication token")
	}

	_, err := CheckSigninConsumerToken(token)
	return err
}

// NewConsumer returns a new builtin consumer for given data.
func NewConsumer(db gorp.SqlExecutor, name, description string, parentConsumer *sdk.AuthConsumer,
	groupIDs []int64, scopes []sdk.AuthConsumerScope) (*sdk.AuthConsumer, string, error) {
	if name == "" {
		return nil, "", sdk.NewErrorFrom(sdk.ErrWrongRequest, "name should be given to create a built in consumer")
	}

	// For each given group id check if it's in parent consumer group ids
	if !parentConsumer.Admin() {
		parentGroupIDs := parentConsumer.GetGroupIDs()
		for i := range groupIDs {
			if !sdk.IsInInt64Array(groupIDs[i], parentGroupIDs) {
				return nil, "", sdk.WrapError(sdk.ErrWrongRequest, "invalid given group id %d", groupIDs[i])
			}
		}
	}

	// At least one scope should be given, for each given scope checks if its authorized and if it's in parent scopes
	if len(scopes) == 0 {
		return nil, "", sdk.NewErrorFrom(sdk.ErrWrongRequest, "built in consumer creation requires at least one scope to be set")
	}
	for i := range scopes {
		// If parent scopes length equals 0 this means all scopes else checks that given scope is in parent scopes.
		if len(parentConsumer.Scopes) > 0 {
			var found bool
			for j := range parentConsumer.Scopes {
				if scopes[i] == parentConsumer.Scopes[j] {
					found = true
					break
				}
			}
			if !found {
				return nil, "", sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given scope %s when creating built in consumer", scopes[i])
			}
		}
	}

	c := sdk.AuthConsumer{
		Name:               name,
		Description:        description,
		AuthentifiedUserID: parentConsumer.AuthentifiedUserID,
		Type:               sdk.ConsumerBuiltin,
		Data:               map[string]string{},
		GroupIDs:           groupIDs,
		Scopes:             scopes,
	}

	if err := authentication.InsertConsumer(db, &c); err != nil {
		return nil, "", err
	}

	jws, err := NewSigninConsumerToken(&c)
	if err != nil {
		return nil, "", err
	}

	return &c, jws, nil
}
