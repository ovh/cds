package user

import (
	"context"
	"github.com/ovh/cds/sdk/telemetry"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func getGPGKeys(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]sdk.UserGPGKey, error) {
	var keys []dbGpgKey
	if err := gorpmapping.GetAll(ctx, db, q, &keys); err != nil {
		return nil, sdk.WrapError(err, "cannot get user gpg keys")
	}
	gpgKeys := make([]sdk.UserGPGKey, 0, len(keys))
	for i := range keys {
		isValid, err := gorpmapping.CheckSignature(keys[i], keys[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "authentified user gpg key %d data corrupted", keys[i].ID)
			continue
		}
		gpgKeys = append(gpgKeys, keys[i].UserGPGKey)
	}
	return gpgKeys, nil
}

func getGPGKey(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) (*sdk.UserGPGKey, error) {
	var key dbGpgKey
	found, err := gorpmapping.Get(ctx, db, q, &key)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get authentified user gpg key")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := gorpmapping.CheckSignature(key, key.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "authentified user gpg key %d data corrupted", key.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	return &key.UserGPGKey, nil
}

func LoadGPGKeysByUserID(ctx context.Context, db gorp.SqlExecutor, userID string) ([]sdk.UserGPGKey, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM user_gpg_key
    WHERE authentified_user_id = $1
  `).Args(userID)
	return getGPGKeys(ctx, db, query)
}

func LoadGPGKeyByKeyID(ctx context.Context, db gorp.SqlExecutor, keyID string) (*sdk.UserGPGKey, error) {
	ctx, next := telemetry.Span(ctx, "user.LoadGPGKeyByKeyID")
	defer next()
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM user_gpg_key
    WHERE key_id = $1
  `).Args(keyID)
	return getGPGKey(ctx, db, query)
}

func InsertGPGKey(ctx context.Context, db gorpmapper.SqlExecutorWithTx, gpgKey *sdk.UserGPGKey) error {
	gpgKey.ID = sdk.UUID()
	gpgKey.Created = time.Now()
	dbKey := dbGpgKey{UserGPGKey: *gpgKey}
	return sdk.WrapError(gorpmapping.InsertAndSign(ctx, db, &dbKey), "unable to insert authentified user gpg key")
}

func DeleteGPGKey(db gorpmapper.SqlExecutorWithTx, gpgKey sdk.UserGPGKey) error {
	dbKey := dbGpgKey{UserGPGKey: gpgKey}
	return sdk.WrapError(gorpmapping.Delete(db, &dbKey), "unable to delete key %s", gpgKey.KeyID)
}
