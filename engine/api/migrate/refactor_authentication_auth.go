package migrate

import (
	"context"
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/authentication/local"
	"github.com/ovh/cds/engine/api/cache"
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
			log.Error(ctx, "migrate.RefactorAuthenticationAuth> %v", err)
		}
	}

	return nil
}

func refactorAuthenticationAuth(ctx context.Context, db *gorp.DbMap, store cache.Store, apiURL, uiURL string, u sdk.AuthentifiedUser) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	var res interface{}

	// Lock the user if it has not been migrated
	if err := tx.SelectOne(&res, `
		SELECT *
		FROM "authentified_user"
		WHERE id = $1
		AND id NOT IN (
			SELECT user_id
			FROM auth_consumer
			WHERE type = 'local'
		)
		FOR UPDATE SKIP LOCKED
	`, u.ID); err != nil {
		if err == sql.ErrNoRows {
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
