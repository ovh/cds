package local

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func getRegistration(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) (*sdk.UserRegistration, error) {
	var reg userRegistration

	found, err := gorpmapping.Get(ctx, db, q, &reg)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get user registration")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := gorpmapping.CheckSignature(reg, reg.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "local.getRegistration> user registration %s data corrupted", reg.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	return &reg.UserRegistration, nil
}

// LoadRegistrationByID returns an user registration from database.
func LoadRegistrationByID(ctx context.Context, db gorp.SqlExecutor, id string) (*sdk.UserRegistration, error) {
	query := gorpmapping.NewQuery("SELECT * FROM user_registration WHERE id = $1").Args(id)
	return getRegistration(ctx, db, query)
}

// InsertRegistration in database.
func InsertRegistration(ctx context.Context, db gorpmapper.SqlExecutorWithTx, ur *sdk.UserRegistration) error {
	if err := sdk.IsValidUsername(ur.Username); err != nil {
		return sdk.WithStack(sdk.ErrInvalidUsername)
	}

	if ur.ID == "" {
		ur.ID = sdk.UUID()
	}
	ur.Created = time.Now()
	r := userRegistration{UserRegistration: *ur}
	if err := gorpmapping.InsertAndSign(ctx, db, &r); err != nil {
		return sdk.WrapError(err, "unable to insert user registration")
	}
	*ur = r.UserRegistration
	return nil
}

// DeleteRegistrationByID removes a user registration in database for given id.
func DeleteRegistrationByID(db gorp.SqlExecutor, id string) error {
	_, err := db.Exec("DELETE FROM user_registration WHERE id = $1", id)
	return sdk.WrapError(err, "unable to delete user registration with id %s", id)
}
