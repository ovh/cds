package index_test

import (
	"context"
	"github.com/mitchellh/hashstructure"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/cdn/index"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func TestLoadItem(t *testing.T) {
	m := gorpmapper.New()
	index.InitDBMapping(m)

	db, _ := test.SetupPGWithMapper(t, m, sdk.TypeCDN)

	apiRef := index.ApiRef{
		ProjectKey: sdk.RandomString(10),
	}
	hashRefU, err := hashstructure.Hash(apiRef, nil)
	require.NoError(t, err)
	hashRef := strconv.FormatUint(hashRefU, 10)

	i := index.Item{
		ApiRef:     apiRef,
		ApiRefHash: hashRef,
		Type:       index.TypeItemStepLog,
	}
	require.NoError(t, index.InsertItem(context.TODO(), m, db, &i))

	res, err := index.LoadItemByID(context.TODO(), m, db, i.ID)
	require.NoError(t, err)
	require.Equal(t, i.ID, res.ID)
	require.Equal(t, i.Type, res.Type)
}
