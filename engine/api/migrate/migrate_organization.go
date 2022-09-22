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

func GetOrganizationUsersToMigrate(ctx context.Context, dbFunc func() *gorp.DbMap) ([]UserOrganizationMigrate, error) {
	db := dbFunc()
	var usersToMigrate []UserOrganizationMigrate

	// Load all new organization
	allOrgas, err := organization.LoadAllOrganizations(ctx, db)
	if err != nil {
		return nil, err
	}
	mapNewOrgas := make(map[string]string)
	for _, o := range allOrgas {
		mapNewOrgas[o.Name] = o.ID
	}

	// Check if users without organization exist
	userWithNoOrg, err := user.LoadUsersWithoutOrganization(ctx, db)
	if err != nil {
		return nil, err
	}
	if len(userWithNoOrg) > 0 {
		return nil, sdk.WrapError(sdk.ErrInvalidData, "You must assign organization to all users before upgrading CDS to the new version.")
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
