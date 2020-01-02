package group

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// LoadLinksGroupUserForGroupIDs returns data from group_user table for given group ids.
func LoadLinksGroupUserForGroupIDs(ctx context.Context, db gorp.SqlExecutor, groupIDs []int64) (LinksGroupUser, error) {
	ls := []LinkGroupUser{}

	query := gorpmapping.NewQuery(`
		SELECT *
		FROM group_user
		WHERE group_id = ANY(string_to_array($1, ',')::int[])
	`).Args(gorpmapping.IDsToQueryString(groupIDs))

	if err := gorpmapping.GetAll(ctx, db, query, &ls); err != nil {
		return nil, sdk.WrapError(err, "cannot get links between group and user")
	}

	return ls, nil
}

// LoadLinksGroupUserForUserIDs returns data from group_user table for given user ids.
func LoadLinksGroupUserForUserIDs(ctx context.Context, db gorp.SqlExecutor, userIDs []int64) (LinksGroupUser, error) {
	ls := []LinkGroupUser{}

	query := gorpmapping.NewQuery(`
		SELECT *
		FROM group_user
		WHERE user_id = ANY(string_to_array($1, ',')::int[])
	`).Args(gorpmapping.IDsToQueryString(userIDs))

	if err := gorpmapping.GetAll(ctx, db, query, &ls); err != nil {
		return nil, sdk.WrapError(err, "cannot get links between group and user")
	}

	return ls, nil
}

// LoadLinkGroupUserForGroupIDAndUserID returns a link from group_user if exists for given group and user ids.
func LoadLinkGroupUserForGroupIDAndUserID(ctx context.Context, db gorp.SqlExecutor, groupID, userID int64) (*LinkGroupUser, error) {
	var l LinkGroupUser

	query := gorpmapping.NewQuery(`
    SELECT *
    FROM group_user
    WHERE group_id = $1 AND user_id = $2
  `).Args(groupID, userID)

	found, err := gorpmapping.Get(ctx, db, query, &l)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get link between group and user")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	return &l, nil
}

// InsertLinkGroupUser inserts given link group-user into database.
func InsertLinkGroupUser(db gorp.SqlExecutor, l *LinkGroupUser) error {
	return sdk.WrapError(gorpmapping.Insert(db, l), "unable to insert link between group and user")
}

// UpdateLinkGroupUser updates given link group-user into database.
func UpdateLinkGroupUser(db gorp.SqlExecutor, l *LinkGroupUser) error {
	return sdk.WrapError(gorpmapping.Update(db, l), "unable to update link between group and user with id: %d", l.ID)
}

// DeleteLinkGroupUser removes given link group-user into database.
func DeleteLinkGroupUser(db gorp.SqlExecutor, l *LinkGroupUser) error {
	return sdk.WrapError(gorpmapping.Delete(db, l), "unable to delete link between group and user with id: %d", l.ID)
}
