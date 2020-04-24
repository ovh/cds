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

	if ok {
		p, err := LoadModelByName(db, p.Name)
		require.NoError(t, err)
		// Eventually we have to clean all project_integration linked
		_, err = db.Exec("delete from project_integration where integration_model_id = $1", p.ID)
		require.NoError(t, err)
		require.NoError(t, DeleteModel(db, p.ID))
	}

	err = InsertModel(db, p)
	require.NoError(t, err)

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
