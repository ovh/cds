package user

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/sdk"
)

// LoadOptionFunc loads data on given authentified users.
type LoadOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.AuthentifiedUser) error

// LoadOptions for authentified users.
var LoadOptions = struct {
	WithContacts     LoadOptionFunc
	WithOrganization LoadOptionFunc
}{
	WithContacts:     loadContacts,
	WithOrganization: loadOrganization,
}

func loadContacts(ctx context.Context, db gorp.SqlExecutor, aus ...*sdk.AuthentifiedUser) error {
	userIDs := sdk.AuthentifiedUsersToIDs(aus)

	contacts, err := LoadContactsByUserIDs(ctx, db, userIDs)
	if err != nil {
		return err
	}

	mapUsers := make(map[string][]sdk.UserContact, len(contacts))
	for i := range contacts {
		if _, ok := mapUsers[contacts[i].UserID]; !ok {
			mapUsers[contacts[i].UserID] = make([]sdk.UserContact, 0, len(contacts))
		}
		mapUsers[contacts[i].UserID] = append(mapUsers[contacts[i].UserID], contacts[i])
	}

	for i := range aus {
		if _, ok := mapUsers[aus[i].ID]; ok {
			aus[i].Contacts = mapUsers[aus[i].ID]
		}
	}

	return nil
}

func loadOrganization(ctx context.Context, db gorp.SqlExecutor, aus ...*sdk.AuthentifiedUser) error {
	userIDs := sdk.AuthentifiedUsersToIDs(aus)

	// Get all organizations for user ids
	userOrgs, err := LoadAllUserOrganizationsByUserIDs(ctx, db, userIDs)
	if err != nil {
		return err
	}

	organizationIDs := make(sdk.StringSlice, len(userOrgs))
	for i := range userOrgs {
		organizationIDs[i] = (userOrgs)[i].OrganizationID
	}
	organizationIDs.Unique()

	organizations, err := organization.LoadOrganizationByIDs(ctx, db, organizationIDs)
	if err != nil {
		return err
	}

	mapOrgsName := make(map[string]string)
	for i := range organizations {
		mapOrgsName[organizations[i].ID] = organizations[i].Name
	}

	mapUserOrgs := make(map[string]string)
	for i := range userOrgs {
		mapUserOrgs[userOrgs[i].AuthentifiedUserID] = mapOrgsName[userOrgs[i].OrganizationID]
	}

	// Set organization on each users
	for i := range aus {
		if org, ok := mapUserOrgs[aus[i].ID]; ok {
			aus[i].Organization = org
		}
	}

	return nil
}
