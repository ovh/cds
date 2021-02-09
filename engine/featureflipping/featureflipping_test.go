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

	featureName := sdk.FeatureName(sdk.RandomString(10))

	assert.False(t, featureflipping.Exists(context.TODO(), m, db, featureName))
	exists, enabled := featureflipping.IsEnabled(context.TODO(), m, db, featureName, map[string]string{
		"my_var": "true",
	})
	assert.False(t, exists)
	assert.False(t, enabled)

	require.NoError(t, featureflipping.Insert(m, db, &sdk.Feature{
		Name: featureName,
		Rule: `return my_var == "true"`,
	}))

	assert.True(t, featureflipping.Exists(context.TODO(), m, db, featureName))
	exists, enabled = featureflipping.IsEnabled(context.TODO(), m, db, featureName, map[string]string{
		"my_var": "true",
	})
	assert.True(t, exists)
	assert.True(t, enabled)

	assert.True(t, featureflipping.Exists(context.TODO(), m, db, featureName))
	exists, enabled = featureflipping.IsEnabled(context.TODO(), m, db, featureName, map[string]string{
		"my_var": "false",
	})
	assert.True(t, exists)
	assert.False(t, enabled)
}
