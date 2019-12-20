package migrate

import (
	"context"
	"database/sql"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	"github.com/go-gorp/gorp"
)

// RefactorAuthenticationUser migrates the old user table to the new user tables.
func RefactorAuthenticationUser(ctx context.Context, db *gorp.DbMap, store cache.Store) error {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM "user"
		WHERE id NOT IN (
			SELECT user_id
			FROM authentified_user_migration
		) ORDER BY id
	`)

	usrs, err := user.GetDeprecatedUsers(ctx, db, query)
	if err != nil {
		return err
	}

	for _, u := range usrs {
		if err := refactorAuthenticationUser(ctx, db, store, u); err != nil {
			log.Error(ctx, "migrate.RefactorAuthentication> %v", err)
		}
	}

	return nil
}

func refactorAuthenticationUser(ctx context.Context, db *gorp.DbMap, store cache.Store, u sdk.User) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	// Lock the user if it has not been migrated
	query := `
		SELECT *
		FROM "user"
		WHERE id = $1
		AND id NOT IN (
			SELECT user_id
			FROM authentified_user_migration
		)
		FOR UPDATE SKIP LOCKED
	`

	if _, err := db.Exec(query, u.ID); err != nil {
		if err == sql.ErrNoRows {
			return sdk.WithStack(err)
		}
		return nil
	}

	log.Info(ctx, "migrate.RefactorAuthenticationUser> starting user migration %s - %s", u.Username, u.Fullname)

	var newUser = sdk.AuthentifiedUser{
		Username:      u.Username,
		Fullname:      u.Fullname,
		OldUserStruct: &u,
	}

	if u.Admin {
		newUser.Ring = sdk.UserRingAdmin
	} else {
		newUser.Ring = sdk.UserRingUser
	}

	if err := user.Insert(ctx, db, &newUser); err != nil {
		return sdk.WithStack(err)
	}

	var contact = sdk.UserContact{
		Type:     sdk.UserContactTypeEmail,
		Value:    u.Email,
		Verified: true,
		UserID:   newUser.ID,
		Primary:  true,
	}

	if err := user.InsertContact(ctx, db, &contact); err != nil {
		return sdk.WithStack(err)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	log.Info(ctx, "migrate.RefactorAuthenticationUser> ending user migration %s - %s", newUser.ID, contact.Value)
	return nil
}
