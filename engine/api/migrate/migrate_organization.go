package migrate

import (
	"context"
	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"

	"github.com/go-gorp/gorp"
)

type UserOrganizationMigrate struct {
	User             *sdk.AuthentifiedUser
	OrganizationName string
}

func GetOrganizationUsersToMigrate(ctx context.Context, db *gorp.DbMap) ([]UserOrganizationMigrate, error) {
	var usersToMigrate []UserOrganizationMigrate

	// Load all new organization
	allOrgas, err := organization.LoadOrganizations(ctx, db)
	if err != nil {
		return nil, err
	}
	mapNewOrgas := make(map[string]string)
	for _, o := range allOrgas {
		mapNewOrgas[o.Name] = o.ID
	}

	// Load all users
	allUsers, err := user.LoadAll(ctx, db)
	if err != nil {
		return nil, err
	}

	userIds := make([]string, len(allUsers))
	mapUsers := make(map[string]*sdk.AuthentifiedUser)
	for i := range allUsers {
		u := &allUsers[i]
		userIds = append(userIds, u.ID)
		mapUsers[u.ID] = u
	}

	// Load all old orga
	oldOrgas, err := user.LoadOldOrganizationsByUserIDs(ctx, db, userIds)
	if err != nil {
		return nil, err
	}

	if len(oldOrgas) != len(allUsers) {
		return nil, sdk.WrapError(sdk.ErrInvalidData, "You must assign organization to all users before upgrading CDS to the new version.")
	}

	// Check if all old orga exist on new organizations
	for _, o := range oldOrgas {
		userToMigrate := UserOrganizationMigrate{
			OrganizationName: o.Organization,
			User:             mapUsers[o.AuthentifiedUserID],
		}
		usersToMigrate = append(usersToMigrate, userToMigrate)
	}
	if len(usersToMigrate) != len(allUsers) {
		return nil, sdk.NewErrorFrom(sdk.ErrInvalidData, "missing some users. Nb of users: %d, nb of users to migrate %d", len(allUsers), len(usersToMigrate))
	}

	return usersToMigrate, nil
}
