package migrate

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gpg"
)

// legacyGPGKeyRow writes user GPG keys signed the way versions without sub_key_ids did.
type legacyGPGKeyRow struct {
	sdk.UserGPGKey
	gorpmapper.SignedEntity
}

func (g legacyGPGKeyRow) Canonical() gorpmapper.CanonicalForms {
	return []gorpmapper.CanonicalForm{
		"{{printf .ID}}{{.AuthentifiedUserID}}{{.KeyID}}{{.PublicKey}}",
	}
}

func init() {
	gorpmapping.Register(gorpmapping.New(legacyGPGKeyRow{}, "user_gpg_key", false, "id"))
}

func insertLegacyGPGKey(t *testing.T, db gorpmapper.SqlExecutorWithTx, userID, publicKey string) sdk.UserGPGKey {
	pub, err := gpg.NewPublicKeyFromPem(publicKey)
	require.NoError(t, err)
	k := sdk.UserGPGKey{ID: sdk.UUID(), AuthentifiedUserID: userID, KeyID: pub.KeyShortID(), SubKeyIDs: sdk.StringSlice{}, PublicKey: publicKey}
	require.NoError(t, gorpmapping.InsertAndSign(context.TODO(), db, &legacyGPGKeyRow{UserGPGKey: k}))
	return k
}

func TestMigrateUserGPGKeySubKeyIDs(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)
	ctx := context.TODO()
	u, _ := assets.InsertLambdaUser(t, db)
	other, _ := assets.InsertLambdaUser(t, db)

	pair, err := keys.GeneratePGPKeyPair(sdk.RandomString(10), "", "test@cds")
	require.NoError(t, err)
	k := insertLegacyGPGKey(t, db, u.ID, pair.Public)
	pub, err := gpg.NewPublicKeyFromPem(pair.Public)
	require.NoError(t, err)
	subKeyID := pub.SubKeyIDs()[0]

	// A tampered row is neither migrated nor made loadable
	pair2, err := keys.GeneratePGPKeyPair(sdk.RandomString(10), "", "test2@cds")
	require.NoError(t, err)
	tampered := insertLegacyGPGKey(t, db, u.ID, pair2.Public)
	pub2, err := gpg.NewPublicKeyFromPem(pair2.Public)
	require.NoError(t, err)
	_, err = db.Exec(`UPDATE user_gpg_key SET authentified_user_id = $1 WHERE id = $2`, other.ID, tampered.ID)
	require.NoError(t, err)

	// Signed with the former canonical form, the row still loads but its subkey is unknown
	loaded, err := user.LoadGPGKeyByKeyID(ctx, db, k.KeyID)
	require.NoError(t, err)
	require.Empty(t, loaded.SubKeyIDs)
	_, err = user.LoadGPGKeyByKeyID(ctx, db, subKeyID)
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound), "got %v", err)

	require.NoError(t, MigrateUserGPGKeySubKeyIDs(ctx, db.DbMap))

	for _, id := range []string{k.KeyID, subKeyID} {
		loaded, err = user.LoadGPGKeyByKeyID(ctx, db, id)
		require.NoError(t, err)
		require.Equal(t, k.ID, loaded.ID)
	}
	require.Equal(t, []string{subKeyID}, []string(loaded.SubKeyIDs))

	_, err = user.LoadGPGKeyByKeyID(ctx, db, pub2.SubKeyIDs()[0])
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound), "got %v", err)
	var subKeyIDs sdk.StringSlice
	require.NoError(t, db.QueryRow(`SELECT sub_key_ids FROM user_gpg_key WHERE id = $1`, tampered.ID).Scan(&subKeyIDs))
	require.Empty(t, subKeyIDs)

	// A second run changes nothing
	require.NoError(t, MigrateUserGPGKeySubKeyIDs(ctx, db.DbMap))
	loaded, err = user.LoadGPGKeyByKeyID(ctx, db, subKeyID)
	require.NoError(t, err)
	require.Equal(t, []string{subKeyID}, []string(loaded.SubKeyIDs))
}
