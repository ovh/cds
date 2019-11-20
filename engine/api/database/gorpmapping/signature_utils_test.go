package gorpmapping_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	_ "github.com/ovh/cds/engine/api/user"
	"github.com/stretchr/testify/assert"
)

func Test_GetSignedEntities(t *testing.T) {
	entities := gorpmapping.ListSignedEntities()
	assert.True(t, len(entities) > 5)
	t.Logf("%v", entities)
}

func Test_ListCanonicalFormsByEntity(t *testing.T) {
	db, _, end := test.SetupPG(t)
	defer end()
	res, err := gorpmapping.ListCanonicalFormsByEntity(db, "user.authentifiedUser")
	require.NoError(t, err)
	t.Logf("%+v", res)
}

func Test_ListTupleByCanonicalForm(t *testing.T) {
	db, _, end := test.SetupPG(t)
	defer end()

	res, err := gorpmapping.ListCanonicalFormsByEntity(db, "user.authentifiedUser")
	require.NoError(t, err)

	if len(res) == 0 {
		t.SkipNow()
	}

	ids, err := gorpmapping.ListTupleByCanonicalForm(db, "user.authentifiedUser", res[0].Signer)
	require.NoError(t, err)
	t.Logf("%+v", ids)

	require.Equal(t, int(res[0].Number), len(ids))
}

func Test_LoadTupleByPrimaryKey(t *testing.T) {
	db, _, end := test.SetupPG(t)
	defer end()

	res, err := gorpmapping.ListCanonicalFormsByEntity(db, "user.authentifiedUser")
	require.NoError(t, err)

	if len(res) == 0 {
		t.SkipNow()
	}

	ids, err := gorpmapping.ListTupleByCanonicalForm(db, "user.authentifiedUser", res[0].Signer)
	require.NoError(t, err)

	u, err := gorpmapping.LoadTupleByPrimaryKey(db, "user.authentifiedUser", ids[0])
	require.NoError(t, err)

	t.Logf("loaded %T : %+v", u, u)
}

func Test_RollSignedTupleByPrimaryKey(t *testing.T) {
	db, _, end := test.SetupPG(t)
	defer end()

	res, err := gorpmapping.ListCanonicalFormsByEntity(db, "user.authentifiedUser")
	require.NoError(t, err)

	if len(res) == 0 {
		t.SkipNow()
	}

	ids, err := gorpmapping.ListTupleByCanonicalForm(db, "user.authentifiedUser", res[0].Signer)
	require.NoError(t, err)

	err = gorpmapping.RollSignedTupleByPrimaryKey(context.TODO(), db, "user.authentifiedUser", ids[0])
	require.NoError(t, err)
}
