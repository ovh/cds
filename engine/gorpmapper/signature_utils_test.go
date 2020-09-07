package gorpmapper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

type testAuthentifiedUser struct {
	sdk.AuthentifiedUser
	gorpmapper.SignedEntity
}

func (u testAuthentifiedUser) Canonical() gorpmapper.CanonicalForms {
	return []gorpmapper.CanonicalForm{
		"{{.ID}}{{.Username}}{{.Fullname}}{{.Ring}}{{printDate .Created}}",
	}
}

func Test_GetSignedEntities(t *testing.T) {
	m := gorpmapper.New()
	m.Register(m.NewTableMapping(testAuthentifiedUser{}, "authentified_user", false, "id"))

	entities := m.ListSignedEntities()
	assert.Len(t, entities, 1)
	t.Logf("%v", entities)
}

func Test_ListCanonicalFormsByEntity(t *testing.T) {
	m := gorpmapper.New()
	m.Register(m.NewTableMapping(testAuthentifiedUser{}, "authentified_user", false, "id"))

	db, _ := test.SetupPGWithMapper(t, m, sdk.TypeAPI)

	res, err := m.ListCanonicalFormsByEntity(db, "gorpmapper_test.testAuthentifiedUser")
	require.NoError(t, err)
	t.Logf("%+v", res)
}

func Test_ListTupleByCanonicalForm(t *testing.T) {
	m := gorpmapper.New()
	m.Register(m.NewTableMapping(testAuthentifiedUser{}, "authentified_user", false, "id"))

	db, _ := test.SetupPGWithMapper(t, m, sdk.TypeAPI)

	res, err := m.ListCanonicalFormsByEntity(db, "gorpmapper_test.testAuthentifiedUser")
	require.NoError(t, err)

	if len(res) == 0 {
		t.SkipNow()
	}

	ids, err := m.ListTupleByCanonicalForm(db, "gorpmapper_test.testAuthentifiedUser", res[0].Signer)
	require.NoError(t, err)
	t.Logf("%+v", ids)

	require.Equal(t, int(res[0].Number), len(ids))
}

func Test_LoadTupleByPrimaryKey(t *testing.T) {
	m := gorpmapper.New()
	m.Register(m.NewTableMapping(testAuthentifiedUser{}, "authentified_user", false, "id"))

	db, _ := test.SetupPGWithMapper(t, m, sdk.TypeAPI)

	res, err := m.ListCanonicalFormsByEntity(db, "gorpmapper_test.testAuthentifiedUser")
	require.NoError(t, err)

	if len(res) == 0 {
		t.SkipNow()
	}

	ids, err := m.ListTupleByCanonicalForm(db, "gorpmapper_test.testAuthentifiedUser", res[0].Signer)
	require.NoError(t, err)

	u, err := m.LoadTupleByPrimaryKey(db, "gorpmapper_test.testAuthentifiedUser", ids[0])
	require.NoError(t, err)

	t.Logf("loaded %T : %+v", u, u)
}

func Test_RollSignedTupleByPrimaryKey(t *testing.T) {
	m := gorpmapper.New()
	m.Register(m.NewTableMapping(testAuthentifiedUser{}, "authentified_user", false, "id"))

	db, _ := test.SetupPGWithMapper(t, m, sdk.TypeAPI)

	res, err := m.ListCanonicalFormsByEntity(db, "gorpmapper_test.testAuthentifiedUser")
	require.NoError(t, err)

	if len(res) == 0 {
		t.SkipNow()
	}

	ids, err := m.ListTupleByCanonicalForm(db, "gorpmapper_test.testAuthentifiedUser", res[0].Signer)
	require.NoError(t, err)

	err = m.RollSignedTupleByPrimaryKey(context.TODO(), db, "gorpmapper_test.testAuthentifiedUser", ids[0])
	require.NoError(t, err)
}
