package group

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func getAllLinksGroupUser(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) (LinksGroupUser, error) {
	var ls []LinkGroupUser
	if err := gorpmapping.GetAll(ctx, db, q, &ls); err != nil {
		return nil, sdk.WrapError(err, "cannot get links between group and user")
	}

	var verifiedLinkGroupUser []LinkGroupUser
	for i := range ls {
		isValid, err := gorpmapping.CheckSignature(ls[i], ls[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "group.LoadLinksGroupUserForUserIDs> group_authentified_user %d data corrupted", ls[i].ID)
			continue
		}
		verifiedLinkGroupUser = append(verifiedLinkGroupUser, ls[i])
	}

	return verifiedLinkGroupUser, nil
}

// LoadLinksGroupUserForGroupIDs returns data from group_user table for given group ids.
func LoadLinksGroupUserForGroupIDs(ctx context.Context, db gorp.SqlExecutor, groupIDs []int64) (LinksGroupUser, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM group_authentified_user
		WHERE group_id = ANY(string_to_array($1, ',')::int[])
	`).Args(gorpmapping.IDsToQueryString(groupIDs))
	return getAllLinksGroupUser(ctx, db, query)
}

// LoadLinksGroupUserForUserIDs returns data from group_user table for given user ids.
func LoadLinksGroupUserForUserIDs(ctx context.Context, db gorp.SqlExecutor, userIDs []string) (LinksGroupUser, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM group_authentified_user
		WHERE authentified_user_id = ANY(string_to_array($1, ','))
	`).Args(gorpmapping.IDStringsToQueryString(userIDs))
	return getAllLinksGroupUser(ctx, db, query)
}

// LoadLinkGroupUserForGroupIDAndUserID returns a link from group_user if exists for given group and user ids.
func LoadLinkGroupUserForGroupIDAndUserID(ctx context.Context, db gorp.SqlExecutor, groupID int64, userID string) (*LinkGroupUser, error) {
	var l LinkGroupUser

	query := gorpmapping.NewQuery(`
    SELECT *
    FROM group_authentified_user
    WHERE group_id = $1 AND authentified_user_id = $2
  `).Args(groupID, userID)

	found, err := gorpmapping.Get(ctx, db, query, &l)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get link between group and user")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := gorpmapping.CheckSignature(l, l.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "group.LoadLinkGroupUserForGroupIDAndUserID> group_authentified_user %d data corrupted", l.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	return &l, nil
}

// InsertLinkGroupUser inserts given link group-user into database.
func InsertLinkGroupUser(ctx context.Context, db gorpmapper.SqlExecutorWithTx, l *LinkGroupUser) error {
	return sdk.WrapError(gorpmapping.InsertAndSign(ctx, db, l), "unable to insert link between group and user")
}

// UpdateLinkGroupUser updates given link group-user into database.
func UpdateLinkGroupUser(ctx context.Context, db gorpmapper.SqlExecutorWithTx, l *LinkGroupUser) error {
	return sdk.WrapError(gorpmapping.UpdateAndSign(ctx, db, l), "unable to update link between group and user with id: %d", l.ID)
}

// DeleteLinkGroupUser removes given link group-user into database.
func DeleteLinkGroupUser(db gorp.SqlExecutor, l *LinkGroupUser) error {
	return sdk.WrapError(gorpmapping.Delete(db, l), "unable to delete link between group and user with id: %d", l.ID)
}

// DeleteAllLinksGroupUser removes all links group-user into database for given group.
func DeleteLinkGroupUserForGroupIDAndUserID(db gorp.SqlExecutor, groupID int64, userID string) error {
	query := `DELETE FROM group_authentified_user WHERE group_id = $1 AND authentified_user_id = $2`
	if _, err := db.Exec(query, groupID, userID); err != nil {
		return sdk.WrapError(err, "unable to delete link between group and user for group %d and user id %s", groupID, userID)
	}
	return nil
}

// DeleteAllLinksGroupUser removes all links group-user into database for given group.
func DeleteAllLinksGroupUserForGroupID(db gorp.SqlExecutor, groupID int64) error {
	query := `DELETE FROM group_authentified_user WHERE group_id = $1`
	if _, err := db.Exec(query, groupID); err != nil {
		return sdk.WrapError(err, "unable to delete links between group and user for group with id: %d", groupID)
	}
	return nil
}
