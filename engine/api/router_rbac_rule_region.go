package api

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

func hasRoleOnRegion(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, regionIdentifier string, role string) error {
	if auth == nil {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	hasRole, err := hasRoleOnRegionAndUserID(ctx, db, role, auth.AuthConsumerUser.AuthentifiedUser, regionIdentifier)
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, cdslog.RbacRole, role)
	if !hasRole {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	return nil
}

func hasRoleOnRegionAndUserID(ctx context.Context, db gorp.SqlExecutor, role string, authentifiedUser *sdk.AuthentifiedUser, regionIdentifier string) (bool, error) {
	ctx, next := telemetry.Span(ctx, "rbac.HasRoleOnRegionAndUserID")
	defer next()

	// Get all region permissions with the given user
	rRegion, err := rbac.LoadRegionIDsByRoleAndUserID(ctx, db, role, authentifiedUser.ID)
	if err != nil {
		return false, err
	}

	// Load user organization to get its ID
	org, err := organization.LoadOrganizationByName(ctx, db, authentifiedUser.Organization)
	if err != nil {
		return false, err
	}

	// Load region ID if needed
	regionID := regionIdentifier
	if !sdk.IsValidUUID(regionID) {
		reg, err := region.LoadRegionByName(ctx, db, regionIdentifier)
		if err != nil {
			return false, err
		}
		regionID = reg.ID
	}

	// Check region and organization
	for _, rr := range rRegion {
		if rr.RegionID == regionID {
			if err := rbac.LoadRBACRegionOrganizations(ctx, db, &rr); err != nil {
				return false, err
			}
			for _, rbacOrga := range rr.RBACOrganizationIDs {
				if rbacOrga == org.ID {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// RegionRead return nil if the current AuthConsumer have the ProjectRoleRead on current project KEY
func (api *API) regionRead(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	regionIdentifier := vars["regionIdentifier"]
	return hasRoleOnRegion(ctx, auth, store, db, regionIdentifier, sdk.RegionRoleList)
}
