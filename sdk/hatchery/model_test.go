package hatchery_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ovh/cds/engine/hatchery/swarm"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
)

func TestModelInterpolateSecrets(t *testing.T) {
	h := &swarm.HatcherySwarm{}

	ctrl := gomock.NewController(t)
	t.Cleanup(func() {
		ctrl.Finish()
	})

	mockCDSClient := mock_cdsclient.NewMockInterface(ctrl)
	h.Client = mockCDSClient

	mockCDSClient.EXPECT().WorkerModelSecretList("mygroup", "mymodel").DoAndReturn(
		func(groupName, modelName string) (sdk.WorkerModelSecrets, error) {
			return sdk.WorkerModelSecrets{
				{Name: "secrets.registry_password", Value: "mysecret"},
			}, nil
		},
	)

	m := sdk.Model{
		ID:   1,
		Type: sdk.Docker,
		Name: "mymodel",
		Group: &sdk.Group{
			ID:   1,
			Name: "mygroup",
		},
		ModelDocker: sdk.ModelDocker{
			Image:    "model:9",
			Private:  true,
			Registry: "lolcat.registry",
			Username: "myusername",
			Password: "{{.secrets.registry_password}}",
		},
	}

	require.NoError(t, hatchery.ModelInterpolateSecrets(h, &m))
	assert.Equal(t, "mysecret", m.ModelDocker.Password)
}
