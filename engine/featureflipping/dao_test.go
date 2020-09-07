package featureflipping_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

func TestDAO(t *testing.T) {
	m := gorpmapper.New()
	featureflipping.Init(m)

	db, _ := test.SetupPGWithMapper(t, m, sdk.TypeAPI)

	all, err := featureflipping.LoadAll(context.TODO(), m, db)
	require.NoError(t, err)
	for _, f := range all {
		require.NoError(t, featureflipping.Delete(db, f.ID))
	}

	var f = sdk.Feature{
		Name: sdk.RandomString(10),
		Rule: sdk.RandomString(10),
	}

	require.NoError(t, featureflipping.Insert(m, db, &f))
	require.NoError(t, featureflipping.Update(m, db, &f))

	_, err = featureflipping.LoadByName(context.TODO(), m, db, f.Name)
	require.NoError(t, err)

	require.NoError(t, featureflipping.Delete(db, f.ID))
}
