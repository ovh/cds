package featureflipping

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsEnabled(t *testing.T) {
	db, _ := test.SetupPG(t)
	var f = sdk.Feature{
		Name: sdk.RandomString(10),
		Rule: `return my_var == "true"`,
	}
	require.NoError(t, Insert(context.TODO(), db, &f))

	assert.True(t, Exists(context.TODO(), db, f.Name))

	vars := map[string]string{
		"my_var": "true",
	}
	assert.True(t, IsEnabled(context.TODO(), db, f.Name, vars))
	assert.True(t, IsEnabled(context.TODO(), db, f.Name, vars)) // this should display a log "featureflipping.IsEnabled> feature_flipping '2qhp3jesa0' loaded from cache"
}
