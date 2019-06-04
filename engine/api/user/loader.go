package user

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type LoadOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.AuthentifiedUser) error

var LoadOptions = struct {
	WithContacts                 LoadOptionFunc
	WithDeprecatedUser           LoadOptionFunc
	WithDeprecatedUserWithGroups LoadOptionFunc
}{
	WithContacts:                 loadContacts,
	WithDeprecatedUser:           loadDeprecatedUser,           // TODO: will be removed
	WithDeprecatedUserWithGroups: loadDeprecatedUserWithGroups, // TODO: will be removed
}

func loadContacts(ctx context.Context, db gorp.SqlExecutor, aus ...*sdk.AuthentifiedUser) error {
	userIDs := sdk.AuthentifiedUsersToIDs(aus)

	query := gorpmapping.NewQuery(`
    SELECT *
    FROM user_contact
    WHERE user_id = ANY(string_to_array($1, ',')::text[])
    ORDER BY id ASC
  `).Args(gorpmapping.IDStringsToQueryString(userIDs))

	var dbContacts []userContact
	if err := gorpmapping.GetAll(ctx, db, query, &dbContacts); err != nil {
		return err
	}

	mapUsers := make(map[string][]sdk.UserContact, len(dbContacts))
	for i := range dbContacts {
		if _, ok := mapUsers[dbContacts[i].UserID]; !ok {
			mapUsers[dbContacts[i].UserID] = make([]sdk.UserContact, 0, len(dbContacts))
		}

		// TODO do not return if any error
		ok, err := gorpmapping.CheckSignature(db, dbContacts[i])
		if err != nil {
			return err
		}
		if !ok {
			return sdk.WithStack(sdk.ErrCorruptedData)
		}

		mapUsers[dbContacts[i].UserID] = append(mapUsers[dbContacts[i].UserID], sdk.UserContact(dbContacts[i]))
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

func loadDeprecatedUserWithGroups(ctx context.Context, db gorp.SqlExecutor, aus ...*sdk.AuthentifiedUser) error {
	if err := loadDeprecatedUser(ctx, db, aus...); err != nil {
		return err
	}

	// Prepare list of old user ids for all given authentified users
	oldUserIDs := make([]int64, 0, len(aus))
	for i := range aus {
		if aus[i].OldUserStruct != nil {
			oldUserIDs = append(oldUserIDs, aus[i].OldUserStruct.ID)
		}
	}

	log.Debug("loadDeprecatedUserWithGroups> oldUserIDs: %v", oldUserIDs)

	// Get link between old users and groups.
	links, err := group.GetLinksGroupUserForUserIDs(ctx, db, oldUserIDs)
	if err != nil {
		return err
	}
	mLinks := make(map[int64][]group.LinkGroupUser)
	for i := range links {
		if _, ok := mLinks[links[i].UserID]; !ok {
			mLinks[links[i].UserID] = []group.LinkGroupUser{links[i]}
		} else {
			mLinks[links[i].UserID] = append(mLinks[links[i].UserID], links[i])
		}
	}

	log.Debug("loadDeprecatedUserWithGroups> links: %v", links)
	log.Debug("loadDeprecatedUserWithGroups> mLinks: %v", mLinks)

	// Get all groups that are referenced in links
	groupIDs := make([]int64, 0, len(links))
	for i := range links {
		groupIDs = append(groupIDs, links[i].GroupID)
	}
	groups, err := group.LoadAllByIDs(ctx, db, groupIDs)
	if err != nil {
		return err
	}
	mGroups := make(map[int64]sdk.Group, len(groups))
	for i := range groups {
		mGroups[groups[i].ID] = groups[i]
	}

	log.Debug("loadDeprecatedUserWithGroups> mGroups: %v", mGroups)

	// For each authentified user, add groups on old user struct
	for _, au := range aus {
		if au.OldUserStruct != nil {
			if _, okLinks := mLinks[au.OldUserStruct.ID]; okLinks {
				for _, link := range mLinks[au.OldUserStruct.ID] {
					if grp, okGroup := mGroups[link.GroupID]; okGroup {
						au.OldUserStruct.Groups = append(au.OldUserStruct.Groups, grp)
					}
				}
			}
		}
	}

	return nil
}
