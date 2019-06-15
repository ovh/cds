package user

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type LoadOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.AuthentifiedUser) error

var LoadOptions = struct {
	WithContacts       LoadOptionFunc
	WithDeprecatedUser LoadOptionFunc
}{
	WithContacts:       loadContacts,
	WithDeprecatedUser: loadDeprecatedUser, // TODO: will be removed
}

func loadContacts(ctx context.Context, db gorp.SqlExecutor, aus ...*sdk.AuthentifiedUser) error {
	userIDs := sdk.AuthentifiedUsersToIDs(aus)

	query := gorpmapping.NewQuery(`
    SELECT *
    FROM user_contact
    WHERE user_id = ANY(string_to_array($1, ',')::text[])
    ORDER BY id ASC
  `).Args(gorpmapping.IDStringsToQueryString(userIDs))

	contacts, err := getContacts(ctx, db, query)
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

func loadDeprecatedUser(ctx context.Context, db gorp.SqlExecutor, aus ...*sdk.AuthentifiedUser) error {
	authentifiedUserIDs := sdk.AuthentifiedUsersToIDs(aus)

	// Get all authentified user migration entries.
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM authentified_user_migration
    WHERE authentified_user_id = ANY(string_to_array($1, ',')::text[])
  `).Args(gorpmapping.IDStringsToQueryString(authentifiedUserIDs))
	var userMigrations []authentifiedUserMigration
	if err := gorpmapping.GetAll(ctx, db, query, &userMigrations); err != nil {
		return err
	}
	mUserMigrations := make(map[string]authentifiedUserMigration, len(userMigrations))
	for i := range userMigrations {
		mUserMigrations[userMigrations[i].AuthentifiedUserID] = userMigrations[i]
	}

	log.Debug("loadDeprecatedUser> userMigrations: %v", userMigrations)
	log.Debug("loadDeprecatedUser> mUserMigrations: %v", mUserMigrations)

	userIDs := make([]int64, len(userMigrations))
	for i := range userMigrations {
		userIDs[i] = userMigrations[i].UserID
	}
	us, err := LoadDeprecatedUsersWithoutAuthByIDs(ctx, db, userIDs)
	if err != nil {
		return err
	}
	mUsers := make(map[int64]sdk.User, len(us))
	for i := range us {
		mUsers[us[i].ID] = us[i]
	}

	log.Debug("loadDeprecatedUser> users: %v", us)
	log.Debug("loadDeprecatedUser> mUsers: %v", mUsers)

	for _, au := range aus {
		if userMigration, okMigration := mUserMigrations[au.ID]; okMigration {
			if oldUser, okUser := mUsers[userMigration.UserID]; okUser {
				au.OldUserStruct = &oldUser
			}
		}
	}

	return nil
}
