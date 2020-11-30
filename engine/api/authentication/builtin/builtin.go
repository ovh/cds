package builtin

import (
	"context"
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/gorpmapper"
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

func (d AuthDriver) GetSessionDuration(_ sdk.AuthDriverUserInfo, _ sdk.AuthConsumer) time.Duration {
	return time.Hour // 1 hour session
}

func (d AuthDriver) GetUserInfo(ctx context.Context, req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	var userInfo sdk.AuthDriverUserInfo

	token, has := req["token"]
	if !has {
		return userInfo, sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid authentication token")
	}

	consumerID, _, err := CheckSigninConsumerToken(token)
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

	_, _, err := CheckSigninConsumerToken(token)
	return err
}

// NewConsumer returns a new builtin consumer for given data.
// The parent consumer should be given with all data loaded including the authentified user.
func NewConsumer(ctx context.Context, db gorpmapper.SqlExecutorWithTx, name, description string, parentConsumer *sdk.AuthConsumer,
	groupIDs []int64, scopes sdk.AuthConsumerScopeDetails) (*sdk.AuthConsumer, string, error) {
	if name == "" {
		return nil, "", sdk.NewErrorFrom(sdk.ErrWrongRequest, "name should be given to create a built in consumer")
	}

	// For each given group id check if it's in parent consumer group ids.
	// When the parent is a builtin consumer even if it was created by an admin we should check groups to prevent
	// creating child with more permission than parents.
	if parentConsumer.Type == sdk.ConsumerBuiltin || !parentConsumer.Admin() {
		// Only if parentGroupIDs aren't empty. Because empty means all groups access
		if len(parentConsumer.GroupIDs) > 0 {
			parentGroupIDs := parentConsumer.GetGroupIDs()
			for i := range groupIDs {
				if !sdk.IsInInt64Array(groupIDs[i], parentGroupIDs) {
					return nil, "", sdk.WrapError(sdk.ErrWrongRequest, "invalid given group id %d", groupIDs[i])
				}
			}
		}
	}

	// Check that given scopes are valid and if they match parent scopes
	if err := checkNewConsumerScopes(parentConsumer.ScopeDetails, scopes); err != nil {
		return nil, "", err
	}

	c := sdk.AuthConsumer{
		Name:               name,
		Description:        description,
		ParentID:           &parentConsumer.ID,
		AuthentifiedUserID: parentConsumer.AuthentifiedUserID,
		Type:               sdk.ConsumerBuiltin,
		Data:               map[string]string{},
		GroupIDs:           groupIDs,
		ScopeDetails:       scopes,
		IssuedAt:           time.Now(),
	}

	if err := authentication.InsertConsumer(ctx, db, &c); err != nil {
		return nil, "", err
	}

	jws, err := NewSigninConsumerToken(&c)
	if err != nil {
		return nil, "", err
	}

	return &c, jws, nil
}

func checkNewConsumerScopes(parentScopes, scopes sdk.AuthConsumerScopeDetails) error {
	// At least one scope should be given, for each given scope checks if its authorized and if it's in parent scopes
	if len(scopes) == 0 {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "built in consumer creation requires at least one scope to be set")
	}
	if err := scopes.IsValid(); err != nil {
		return err
	}
	// If parent scopes length equals 0 this means all scopes else checks that given scope is in parent scopes
	if len(parentScopes) == 0 {
		return nil
	}

	for i := range scopes {
		var validScope bool
		for j := range parentScopes {
			if scopes[i].Scope == parentScopes[j].Scope {
				// if no endpoint restrictions on parent scope is valid
				if len(parentScopes[j].Endpoints) == 0 {
					validScope = true
					break
				}

				// invalid if no endpoint given and parents endpoint list is not empty
				if len(scopes[i].Endpoints) == 0 {
					return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid endpoints for scope %s when creating built in consumer", scopes[i].Scope)
				}

				// if parent as scope restrictions, child should contains only all or a subset of those restrictions
				for _, e := range scopes[i].Endpoints {
					existsParentEndpoint, parentEndpoint := parentScopes[j].Endpoints.FindEndpoint(e.Route)
					if !existsParentEndpoint {
						return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given route %s for scope %s when creating built in consumer", e.Route, scopes[i].Scope)
					}
					if len(parentEndpoint.Methods) > 0 {
						if len(e.Methods) == 0 {
							return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given methods for route %s and scope %s when creating built in consumer", e.Route, scopes[i].Scope)
						}
						for _, m := range e.Methods {
							if !parentEndpoint.Methods.Contains(m) {
								return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given method %s for route %s and scope %s when creating built in consumer", m, e.Route, scopes[i].Scope)
							}
						}
					}
				}

				validScope = true
				break
			}
		}
		if !validScope {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given scope %s when creating built in consumer", scopes[i])
		}
	}

	return nil
}
