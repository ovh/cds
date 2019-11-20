package migrate

import (
	"context"
	"database/sql"

	"github.com/ovh/cds/engine/api/authentication/local"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	"github.com/go-gorp/gorp"
)

// RefactorAuthentication migrates the old user table to the new user tables
func RefactorAuthentication(ctx context.Context, db *gorp.DbMap, store cache.Store, apiURL, uiURL string) error {
	query := gorpmapping.NewQuery(`select * from "user" where id not in (select user_id from authentified_user_migration) order by id`)

	usrs, err := user.GetDeprecatedUsers(ctx, db, query)
	if err != nil {
		return err
	}

	for _, u := range usrs {
		tx, err := db.Begin()
		if err != nil {
			log.Error(ctx, "migrate.RefactorAuthentication> %v", err)
			continue
		}

		if err := refactorAuthenticationUser(ctx, tx, store, u, apiURL, uiURL); err != nil {
			log.Error(ctx, "migrate.RefactorAuthentication> %v", err)
			tx.Rollback() // nolint
			continue
		}

		if err := tx.Commit(); err != nil {
			log.Error(ctx, "migrate.RefactorAuthentication> %v", err)
			tx.Rollback() // nolint
			continue
		}
	}

	return nil
}

func refactorAuthenticationUser(ctx context.Context, db gorp.SqlExecutor, store cache.Store, u sdk.User, apiURL, uiURL string) error {
	// Lock the user if it has not been migrated
	_, err := db.Exec(`
	SELECT * 
	FROM "user" 
	WHERE id = $1 
	AND id NOT IN (
		SELECT user_id 
		FROM authentified_user_migration
	)
	FOR UPDATE SKIP LOCKED`, u.ID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return sdk.WithStack(err)
	}

	log.Info("migrate.RefactorAuthentication> starting user migration %s - %s", u.Username, u.Fullname)

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

	consumer, err := local.NewConsumer(ctx, db, newUser.ID)
	if err != nil {
		return err
	}

	resetToken, err := local.NewResetConsumerToken(store, consumer.ID)
	if err != nil {
		return err
	}

	// Insert the authentication
	if err := mail.SendMailAskResetToken(u.Email, newUser.Username, resetToken,
		apiURL+"/auth/reset?token=%s",
		uiURL,
	); err != nil {
		return sdk.WrapError(err, "cannot send reset token email at %s", contact.Value)
	}

	log.Info("migrate.RefactorAuthentication> ending user migration %s - %s", newUser.ID, contact.Value)

	return nil
}
