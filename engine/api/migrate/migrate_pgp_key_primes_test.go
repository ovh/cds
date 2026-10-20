package migrate

import (
	"context"
	"crypto/rsa"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/openpgp/packet"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

// misorderPGPPrivateKey returns the key with p > q for the primary key, and for the subkey too unless
// subkeyValid, like the keys generated before the primes were ordered: a parse then serialization swaps
// the primes of every key that is not ordered beforehand.
func misorderPGPPrivateKey(t *testing.T, private string, subkeyValid bool) string {
	entity, err := keys.GetOpenPGPEntity(strings.NewReader(private))
	require.NoError(t, err)
	if subkeyValid {
		keys.OrderRSAPrimes(entity.Subkeys[0].PrivateKey)
	}
	misordered, err := armoredPGPPrivateKey(entity)
	require.NoError(t, err)
	return misordered
}

func rsaPrimes(t *testing.T, private string) [][]*big.Int {
	entity, err := keys.GetOpenPGPEntity(strings.NewReader(private))
	require.NoError(t, err)
	var primes [][]*big.Int
	for _, pk := range []*packet.PrivateKey{entity.PrivateKey, entity.Subkeys[0].PrivateKey} {
		primes = append(primes, pk.PrivateKey.(*rsa.PrivateKey).Primes)
	}
	return primes
}

func TestFixPGPPrivateKeyPrimes(t *testing.T) {
	k, err := keys.GeneratePGPKeyPair("mykey", "this is a test key", "email@localhost.local")
	require.NoError(t, err)

	t.Run("valid key is left unchanged", func(t *testing.T) {
		fixed, changed, err := fixPGPPrivateKeyPrimes(k.Private)
		require.NoError(t, err)
		require.False(t, changed)
		require.Equal(t, k.Private, fixed)
	})

	for _, subkeyValid := range []bool{false, true} {
		t.Run(fmt.Sprintf("misordered key is fixed, subkey valid: %v", subkeyValid), func(t *testing.T) {
			misordered := misorderPGPPrivateKey(t, k.Private, subkeyValid)
			primes := rsaPrimes(t, misordered)
			require.Equal(t, 1, primes[0][0].Cmp(primes[0][1]), "primary key must start with p > q")
			require.Equal(t, subkeyValid, primes[1][0].Cmp(primes[1][1]) < 0)

			fixed, changed, err := fixPGPPrivateKeyPrimes(misordered)
			require.NoError(t, err)
			require.True(t, changed)
			for _, p := range rsaPrimes(t, fixed) {
				require.Equal(t, -1, p[0].Cmp(p[1]), "p must be lower than q")
			}

			entity, err := keys.GetOpenPGPEntity(strings.NewReader(fixed))
			require.NoError(t, err)
			require.Equal(t, k.LongKeyID, entity.PrimaryKey.KeyIdString())
			public, err := armoredPGPPublicKey(entity)
			require.NoError(t, err)
			require.Equal(t, k.Public, public)

			_, changed, err = fixPGPPrivateKeyPrimes(fixed)
			require.NoError(t, err)
			require.False(t, changed, "a fixed key needs no further change")
		})
	}
}

func insertTestProjectPGPKey(t *testing.T, db gorpmapper.SqlExecutorWithTx, projectID int64, k sdk.Key) sdk.ProjectKey {
	pk := sdk.ProjectKey{Name: k.Name, Type: sdk.KeyTypePGP, ProjectID: projectID,
		Public: k.Public, Private: k.Private, KeyID: k.KeyID, LongKeyID: k.LongKeyID}
	require.NoError(t, project.InsertKey(db, &pk))
	return pk
}

func TestMigratePGPKeys_FixesMisorderedKeysOnly(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	ctx := context.TODO()
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	misordered, err := keys.GeneratePGPKeyPair("proj-misordered", "proj-misordered", "proj-misordered@cds")
	require.NoError(t, err)
	misordered.Private = misorderPGPPrivateKey(t, misordered.Private, false)
	misorderedKey := insertTestProjectPGPKey(t, db, proj.ID, misordered)
	valid, err := keys.GeneratePGPKeyPair("proj-valid", "proj-valid", "proj-valid@cds")
	require.NoError(t, err)
	validKey := insertTestProjectPGPKey(t, db, proj.ID, valid)

	// Builtin keys are only used by the API to encrypt secrets, never by jobs
	builtinID, err := db.SelectInt("SELECT id FROM project_key WHERE project_id = $1 AND builtin = true", proj.ID)
	require.NoError(t, err)
	refs, err := project.LoadPGPKeyRefs(ctx, db)
	require.NoError(t, err)
	misorderedRef := project.PGPKeyRef{ID: misorderedKey.ID, ProjectKey: proj.Key}
	validRef := project.PGPKeyRef{ID: validKey.ID, ProjectKey: proj.Key}
	require.Contains(t, refs, misorderedRef)
	require.Contains(t, refs, validRef)
	require.NotContains(t, refs, project.PGPKeyRef{ID: builtinID, ProjectKey: proj.Key})

	counts := migratePGPKeys(ctx, db.DbMap, []project.PGPKeyRef{misorderedRef, validRef, {ID: -1, ProjectKey: proj.Key}})
	require.Equal(t, pgpKeyCounts{fixed: 1, skipped: 2}, counts)

	k, err := project.LoadAndLockKeyByID(ctx, db, misorderedKey.ID)
	require.NoError(t, err)
	_, stillMisordered, err := fixPGPPrivateKeyPrimes(k.Private)
	require.NoError(t, err)
	require.False(t, stillMisordered, "p must be lower than q")
	require.Equal(t, misordered.Public, k.Public)
	require.Equal(t, misordered.LongKeyID, k.LongKeyID)
	e, err := keys.GetOpenPGPEntity(strings.NewReader(k.Private))
	require.NoError(t, err)
	require.Equal(t, misordered.LongKeyID, e.PrimaryKey.KeyIdString())

	// A valid key is not rewritten
	k, err = project.LoadAndLockKeyByID(ctx, db, validKey.ID)
	require.NoError(t, err)
	require.Equal(t, valid.Private, k.Private)

	// A second run finds nothing left to fix
	require.Equal(t, pgpKeyCounts{skipped: 1}, migratePGPKeys(ctx, db.DbMap, []project.PGPKeyRef{misorderedRef}))
}
