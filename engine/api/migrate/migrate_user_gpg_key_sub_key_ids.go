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
// A subkey ID already used by another key is left out, so that a commit never matches two users.
func MigrateUserGPGKeySubKeyIDs(ctx context.Context, db *gorp.DbMap) error {
	keys, err := user.LoadAllGPGKeysUnverified(ctx, db)
	if err != nil {
		return err
	}

	// owners maps every key ID in use to the primary key ID of its key.
	owners := make(map[string]string)
	for _, k := range keys {
		owners[k.KeyID] = k.KeyID
		for _, id := range k.SubKeyIDs {
			owners[id] = k.KeyID
		}
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
		if other, collides := firstOwnedByAnother(owners, ids, k.KeyID); collides {
			log.Warn(ctx, "MigrateUserGPGKeySubKeyIDs: subkey IDs of key %s collide with key %s, not stored", k.KeyID, other)
			continue
		}
		k.SubKeyIDs = ids
		if err := updateUserGPGKey(ctx, db, k); err != nil {
			return err
		}
		for _, id := range ids {
			owners[id] = k.KeyID
		}
	}
	return nil
}

func firstOwnedByAnother(owners map[string]string, ids []string, keyID string) (string, bool) {
	for _, id := range ids {
		if owner, ok := owners[id]; ok && owner != keyID {
			return owner, true
		}
		if id == keyID {
			return keyID, true
		}
	}
	return "", false
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
