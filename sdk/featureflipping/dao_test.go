package featureflipping

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestDAO(t *testing.T) {
	db, _ := test.SetupPG(t)
	all, err := LoadAll(context.TODO(), db)
	require.NoError(t, err)
	for _, f := range all {
		require.NoError(t, Delete(context.TODO(), db, f.ID))
	}

	var f = sdk.Feature{
		Name: sdk.RandomString(10),
		Rule: sdk.RandomString(10),
	}

	require.NoError(t, Insert(context.TODO(), db, &f))
	require.NoError(t, Update(context.TODO(), db, &f))

	_, err = LoadByName(context.TODO(), db, f.Name)
	require.NoError(t, err)

	require.NoError(t, Delete(context.TODO(), db, f.ID))
}
