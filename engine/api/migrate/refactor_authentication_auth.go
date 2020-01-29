package migrate

import (
	"context"
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/authentication/local"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/mail"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// RefactorAuthenticationAuth .
func RefactorAuthenticationAuth(ctx context.Context, db *gorp.DbMap, store cache.Store, apiURL, uiURL string) error {
	// get all authentified user without local consumer
	us, err := user.LoadAll(ctx, db, user.LoadOptions.WithContacts)
	if err != nil {
		return err
	}

	for _, u := range us {
		if err := refactorAuthenticationAuth(ctx, db, store, apiURL, uiURL, u); err != nil {
			log.Error(ctx, "migrate.RefactorAuthenticationAuth> %+v", err)
		}
	}

	return nil
}

func refactorAuthenticationAuth(ctx context.Context, db *gorp.DbMap, store cache.Store, apiURL, uiURL string, u sdk.AuthentifiedUser) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	// Lock the user if it has not been migrated
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM "authentified_user"
		WHERE id = $1
		AND id NOT IN (
			SELECT user_id
			FROM auth_consumer
			WHERE type = 'local'
		)
		FOR UPDATE SKIP LOCKED
	`).Args(u.ID)

	if _, err := user.Get(ctx, tx, query); err != nil {
		if sdk.ErrorIs(err, sdk.ErrUserNotFound) {
			log.Info(ctx, "migrate.RefactorAuthenticationAuth> local auth consumer already exists for %s(%s)", u.Username, u.ID)
			return nil
		}
		return sdk.WithStack(err)
	}

	log.Info(ctx, "migrate.RefactorAuthenticationAuth> starting auth migration %s(%s)", u.Username, u.ID)

	localConsumer, err := local.NewConsumer(ctx, tx, u.ID)
	if err != nil {
		return err
	}

	resetToken, err := local.NewResetConsumerToken(store, localConsumer.ID)
	if err != nil {
		return err
	}

	email := u.GetEmail()
	if err := mail.SendMailAskResetToken(ctx, email, u.Username, resetToken,
		uiURL+"/auth/reset?token=%s",
		apiURL,
	); err != nil {
		return sdk.WrapError(err, "cannot send reset token email at %s", email)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	log.Info(ctx, "migrate.RefactorAuthenticationAuth> ending user auth migration %s(%s)", u.Username, u.ID)
	return nil
}

// RefactorGroupMembership .
func RefactorGroupMembership(ctx context.Context, db *gorp.DbMap) error {
	log.Debug("migrate.RefactorGroupMembership> begin")
	defer func() {
		log.Debug("migrate.RefactorGroupMembership> end")
	}()

	// First step
	// Migrate "group" entities to sign it
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	rows, err := tx.Query(`SELECT id, name FROM "group" WHERE sig IS NULL FOR UPDATE SKIP LOCKED`)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return sdk.WithStack(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return sdk.WithStack(err)
		}

		g := sdk.Group{
			ID:   id,
			Name: name,
		}

		if err := group.Update(ctx, tx, &g); err != nil {
			return sdk.WithStack(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	// Second step
	// Migrate data from table user_°group
	tx, err = db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	rows, err = tx.Query(`
	SELECT group_user.group_id, authentified_user_migration.authentified_user_id, group_user.group_admin 
	FROM group_user
	JOIN authentified_user_migration ON authentified_user_migration.user_id = group_user.user_id
	WHERE authentified_user_migration.authentified_user_id NOT IN (
		SELECT DISTINCT authentified_user_id 
		FROM group_authentified_user
	)
	FOR UPDATE SKIP LOCKED
	`)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return sdk.WithStack(err)
	}
	defer rows.Close()

	for rows.Next() {
		var groupID int64
		var authentifiedUserID string
		var groupAdmin bool
		if err := rows.Scan(&groupID, &authentifiedUserID, &groupAdmin); err != nil {
			return sdk.WithStack(err)
		}

		var l = group.LinkGroupUser{
			GroupID:            groupID,
			AuthentifiedUserID: authentifiedUserID,
			Admin:              groupAdmin,
		}

		if err := group.InsertLinkGroupUser(ctx, tx, &l); err != nil {
			return sdk.WithStack(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	return nil
}
