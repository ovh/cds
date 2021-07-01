package group

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

// Create insert a new group in database and set user for given id as group admin.
func Create(ctx context.Context, db gorpmapper.SqlExecutorWithTx, grp *sdk.Group, userID string) error {
	if err := Insert(ctx, db, grp); err != nil {
		return err
	}

	if err := InsertLinkGroupUser(ctx, db, &LinkGroupUser{
		GroupID:            grp.ID,
		AuthentifiedUserID: userID,
		Admin:              true,
	}); err != nil {
		return err
	}

	return nil
}

// Delete deletes group and dependencies.
func Delete(_ context.Context, db gorp.SqlExecutor, g *sdk.Group) error {
	// Remove the group from database, this will also delete cascade group_user links
	return deleteDB(db, g)
}
