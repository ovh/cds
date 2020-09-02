package featureflipping_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

func TestIsEnabled(t *testing.T) {
	m := gorpmapper.New()
	featureflipping.Init(m)

	db, _ := test.SetupPGWithMapper(t, m, sdk.TypeAPI)

	var f = sdk.Feature{
		Name: sdk.RandomString(10),
		Rule: `return my_var == "true"`,
	}
	require.NoError(t, featureflipping.Insert(m, db, &f))

	assert.True(t, featureflipping.Exists(context.TODO(), m, db, f.Name))

	vars := map[string]string{
		"my_var": "true",
	}
	assert.True(t, featureflipping.IsEnabled(context.TODO(), m, db, f.Name, vars))
	assert.True(t, featureflipping.IsEnabled(context.TODO(), m, db, f.Name, vars)) // this should display a log "featureflipping.IsEnabled> feature_flipping '2qhp3jesa0' loaded from cache"
}
