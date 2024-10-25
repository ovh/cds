package builtin

import (
	"context"
	"github.com/ovh/cds/engine/api/driver/builtin"
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

var _ sdk.AuthDriver = new(AuthDriver)

// NewDriver returns a new initialized driver for builtin authentication.
func NewDriver() sdk.AuthDriver {
	return &AuthDriver{
		driver: builtin.NewBuiltinDriver(),
	}
}

// AuthDriver for builtin authentication.
type AuthDriver struct {
	driver sdk.Driver
}

func (d AuthDriver) GetDriver() sdk.Driver {
	return d.driver
}

func (d AuthDriver) GetManifest() sdk.AuthDriverManifest {
	return sdk.AuthDriverManifest{
		Type:           sdk.ConsumerBuiltin,
		SignupDisabled: true,
	}
}

func (d AuthDriver) GetSessionDuration() time.Duration {
	return time.Hour // 1 hour session
}

func (d AuthDriver) GetUserInfo(ctx context.Context, req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	return d.driver.GetUserInfoFromDriver(ctx, req)
}

// NewConsumer returns a new builtin consumer for given data.
// The parent consumer should be given with all data loaded including the authentified user.
type NewConsumerOptions struct {
	Name                         string
	Description                  string
	Duration                     time.Duration
	GroupIDs                     []int64
	Scopes                       sdk.AuthConsumerScopeDetails
	ServiceName                  *string
	ServiceType                  *string
	ServiceRegion                *string
	ServiceIgnoreJobWithNoRegion *bool
}

func NewConsumer(ctx context.Context, db gorpmapper.SqlExecutorWithTx, opts NewConsumerOptions, parentConsumer *sdk.AuthUserConsumer) (*sdk.AuthUserConsumer, string, error) {
	if opts.Name == "" {
		return nil, "", sdk.NewErrorFrom(sdk.ErrWrongRequest, "name should be given to create a built in consumer")
	}

	// For each given group id check if it's in parent consumer group ids.
	// When the parent is a builtin consumer even if it was created by an admin we should check groups to prevent
	// creating child with more permission than parents.
	if parentConsumer.Type == sdk.ConsumerBuiltin || !parentConsumer.Admin() {
		// Only if parentGroupIDs aren't empty. Because empty means all groups access
		if len(parentConsumer.AuthConsumerUser.GroupIDs) > 0 {
			parentGroupIDs := parentConsumer.GetGroupIDs()
			for i := range opts.GroupIDs {
				if !sdk.IsInInt64Array(opts.GroupIDs[i], parentGroupIDs) {
					return nil, "", sdk.WrapError(sdk.ErrWrongRequest, "invalid given group id %d", opts.GroupIDs[i])
				}
			}
		}

	}

	// Check that given scopes are valid and if they match parent scopes

	parentScope := parentConsumer.AuthConsumerUser.ScopeDetails

	if err := checkNewConsumerScopes(parentScope, opts.Scopes); err != nil {
		return nil, "", err
	}

	c := sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            opts.Name,
			Description:     opts.Description,
			ParentID:        &parentConsumer.ID,
			Type:            sdk.ConsumerBuiltin,
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), opts.Duration),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			AuthentifiedUserID:           parentConsumer.AuthConsumerUser.AuthentifiedUserID,
			Data:                         map[string]string{},
			GroupIDs:                     opts.GroupIDs,
			ScopeDetails:                 opts.Scopes,
			ServiceName:                  opts.ServiceName,
			ServiceType:                  opts.ServiceType,
			ServiceRegion:                opts.ServiceRegion,
			ServiceIgnoreJobWithNoRegion: opts.ServiceIgnoreJobWithNoRegion,
		},
	}

	if err := authentication.InsertUserConsumer(ctx, db, &c); err != nil {
		return nil, "", err
	}

	jws, err := builtin.NewSigninConsumerToken(&c)
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
