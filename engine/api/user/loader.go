package user

import (
	"context"

	"github.com/go-gorp/gorp"

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
	orgs, err := LoadOrganizationsByUserIDs(ctx, db, userIDs)
	if err != nil {
		return err
	}
	mOrgs := make(map[string]Organization)
	for i := range orgs {
		mOrgs[orgs[i].AuthentifiedUserID] = orgs[i]
	}

	// Set organization on each users
	for i := range aus {
		if org, ok := mOrgs[aus[i].ID]; ok {
			aus[i].Organization = org.Organization
		}
	}

	return nil
}
