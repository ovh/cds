package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestCRUDModel(t *testing.T) {
	db, _ := test.SetupPG(t)

	var p = &sdk.KafkaIntegration

	ok, err := ModelExists(db, p.Name)
	require.NoError(t, err)

	if !ok {
		err = InsertModel(db, p)
		require.NoError(t, err)
	} else {
		p1, err := LoadModelByName(context.TODO(), db, p.Name)
		require.NoError(t, err)
		p = &p1
	}

	model, err := LoadModelByNameWithClearPassword(context.TODO(), db, p.Name)
	require.NoError(t, err)

	model.PublicConfigurations = sdk.IntegrationConfigMap{
		"A": sdk.IntegrationConfig{},
		"B": sdk.IntegrationConfig{},
	}
	err = UpdateModel(context.TODO(), db, p)
	require.NoError(t, err)

	models, err := LoadModels(db)
	require.NoError(t, err)

	assert.True(t, len(models) > 1)
}
