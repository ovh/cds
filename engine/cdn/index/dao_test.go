package index_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func TestLoadItem(t *testing.T) {
	m := gorpmapper.New()
	index.Init(m)

	db, _ := test.SetupPGWithMapper(t, m, sdk.TypeCDN)

	i := index.Item{
		Name: sdk.RandomString(10),
	}
	require.NoError(t, index.InsertItem(context.TODO(), m, db, &i))

	res, err := index.LoadItemByID(context.TODO(), m, db, i.ID)
	require.NoError(t, err)
	require.Equal(t, i.ID, res.ID)
	require.Equal(t, i.Name, res.Name)
}
