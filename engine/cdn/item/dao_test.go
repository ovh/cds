package item_test

import (
	"context"
	"github.com/ovh/cds/sdk/cdn"
	"testing"

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

	apiRef := sdk.NewCDNLogApiRef(cdn.Signature{
		ProjectKey: sdk.RandomString(10),
	})
	hashRef, err := apiRef.ToHash()
	require.NoError(t, err)

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
	_, is := res.APIRef.(*sdk.CDNLogAPIRef)
	require.True(t, is)

	_, no := res.APIRef.(*sdk.CDNArtifactAPIRef)
	require.False(t, no)
}
