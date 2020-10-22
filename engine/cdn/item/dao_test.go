package item_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/mitchellh/hashstructure"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/cdn/item"
	cdntest "github.com/ovh/cds/engine/cdn/test"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

func TestLoadItem(t *testing.T) {
	m := gorpmapper.New()
	item.InitDBMapping(m)

	db, _ := test.SetupPGWithMapper(t, m, sdk.TypeCDN)
	cdntest.ClearItem(t, context.TODO(), m, db)

	apiRef := sdk.CDNLogAPIRef{
		ProjectKey: sdk.RandomString(10),
	}
	hashRefU, err := hashstructure.Hash(apiRef, nil)
	require.NoError(t, err)
	hashRef := strconv.FormatUint(hashRefU, 10)

	i := sdk.CDNItem{
		APIRef:     apiRef,
		APIRefHash: hashRef,
		Type:       sdk.CDNTypeItemStepLog,
	}
	require.NoError(t, item.Insert(context.TODO(), m, db, &i))
	t.Cleanup(func() { _ = item.DeleteByID(db, i.ID) })

	res, err := item.LoadByID(context.TODO(), m, db, i.ID)
	require.NoError(t, err)
	require.Equal(t, i.ID, res.ID)
	require.Equal(t, i.Type, res.Type)
}
