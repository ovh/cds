package group

import (
	"context"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

// CheckUserInDefaultGroup insert user in default group
func CheckUserInDefaultGroup(ctx context.Context, db gorpmapper.SqlExecutorWithTx, userID string) error {
	if DefaultGroup == nil || DefaultGroup.ID == 0 || userID == "" {
		return nil
	}

	l, err := LoadLinkGroupUserForGroupIDAndUserID(ctx, db, DefaultGroup.ID, userID)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return err
	}

	// If user is not in default group at it
	if l == nil {
		return InsertLinkGroupUser(ctx, db, &LinkGroupUser{
			GroupID:            DefaultGroup.ID,
			AuthentifiedUserID: userID,
			Admin:              false,
		})
	}

	return nil
}
