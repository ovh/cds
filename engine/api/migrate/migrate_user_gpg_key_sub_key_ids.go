package migrate

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gpg"
)

// MigrateUserGPGKeySubKeyIDs stores the subkey IDs of the user GPG keys imported before they were recorded.
// Rows failing their signature are left untouched, so that re-signing never validates a tampered key.
// A subkey ID already used by another key is left out, so that a commit never matches two users.
func MigrateUserGPGKeySubKeyIDs(ctx context.Context, db *gorp.DbMap) error {
	keys, err := user.LoadAllGPGKeys(ctx, db)
	if err != nil {
		return err
	}

	for i := range keys {
		k := &keys[i]
		if len(k.SubKeyIDs) > 0 {
			continue
		}
		pub, err := gpg.NewPublicKeyFromPem(k.PublicKey)
		if err != nil {
			log.Error(ctx, "MigrateUserGPGKeySubKeyIDs: key %s: cannot parse public key: %v", k.KeyID, err)
			continue
		}
		ids := pub.SubKeyIDs()
		if len(ids) == 0 {
			continue
		}
		exists, err := user.GPGKeyIDsExist(db, ids)
		if err != nil {
			return err
		}
		if exists {
			log.Warn(ctx, "MigrateUserGPGKeySubKeyIDs: subkey IDs of key %s already used by a stored key, not stored", k.KeyID)
			continue
		}
		k.SubKeyIDs = ids
		if err := updateUserGPGKey(ctx, db, k); err != nil {
			return err
		}
	}
	return nil
}

func updateUserGPGKey(ctx context.Context, db *gorp.DbMap, k *sdk.UserGPGKey) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint
	if err := user.UpdateGPGKeySubKeyIDs(ctx, tx, k); err != nil {
		return err
	}
	return sdk.WithStack(tx.Commit())
}
