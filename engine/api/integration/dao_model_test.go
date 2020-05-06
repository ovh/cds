package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestCRUDModel(t *testing.T) {
	db, _, end := test.SetupPG(t)
	defer end()

	var p = &sdk.KafkaIntegration

	ok, err := ModelExists(db, p.Name)
	require.NoError(t, err)

	if !ok {
		err = InsertModel(db, p)
		require.NoError(t, err)
	} else {
		p1, err := LoadModelByName(db, p.Name)
		require.NoError(t, err)
		p = &p1
	}

	model, err := LoadModelByNameWithClearPassword(db, p.Name)
	require.NoError(t, err)

	model.PublicConfigurations = sdk.IntegrationConfigMap{
		"A": sdk.IntegrationConfig{},
		"B": sdk.IntegrationConfig{},
	}
	err = UpdateModel(db, p)
	require.NoError(t, err)

	models, err := LoadModels(db)
	require.NoError(t, err)

	assert.True(t, len(models) > 1)

	filter := sdk.IntegrationTypeEvent
	_, err = LoadPublicModelsByTypeWithDecryption(db, &filter)
	require.NoError(t, err)
}
