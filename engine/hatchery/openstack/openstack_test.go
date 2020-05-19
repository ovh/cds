package openstack

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/ovh/cds/sdk/log"
)

func TestHatcheryOpenstack_CanSpawn(t *testing.T) {
	h := &HatcheryOpenstack{}

	// no model, no requirement, canSpawn must be true
	canSpawn := h.CanSpawn(context.TODO(), nil, 1, nil)
	require.True(t, canSpawn)

	// no model, service requirement, canSpawn must be false: service can't be managed by openstack hatchery
	canSpawn = h.CanSpawn(context.TODO(), nil, 1, []sdk.Requirement{{Name: "pg", Type: sdk.ServiceRequirement, Value: "postgres:9.5.4"}})
	require.False(t, canSpawn)

	// no model, memory prerequisite, canSpawn must be false: memory prerequisite can't be managed by openstack hatchery
	canSpawn = h.CanSpawn(context.TODO(), nil, 1, []sdk.Requirement{{Name: "mem", Type: sdk.MemoryRequirement, Value: "4096"}})
	require.False(t, canSpawn)

	// no model, hostname prerequisite, canSpawn must be false: hostname can't be managed by openstack hatchery
	canSpawn = h.CanSpawn(context.TODO(), nil, 1, []sdk.Requirement{{Type: sdk.HostnameRequirement, Value: "localhost"}})
	require.False(t, canSpawn)
}

func TestHatcheryOpenstack_WorkerModelsEnabled(t *testing.T) {
	log.SetLogger(t)

	h := &HatcheryOpenstack{}

	ctrl := gomock.NewController(t)
	mockClient := mock_cdsclient.NewMockInterface(ctrl)
	h.Client = mockClient
	t.Cleanup(func() { ctrl.Finish() })

	mockClient.EXPECT().WorkerModelsEnabled().DoAndReturn(func() ([]sdk.Model, error) {
		return []sdk.Model{
			{
				ID:    1,
				Type:  sdk.Docker,
				Name:  "my-model-1",
				Group: &sdk.Group{ID: 1, Name: "mygroup"},
			},
			{
				ID:                  2,
				Type:                sdk.Openstack,
				Name:                "my-model-2",
				Group:               &sdk.Group{ID: 1, Name: "mygroup"},
				ModelVirtualMachine: sdk.ModelVirtualMachine{Flavor: "b2-120"},
			},
			{
				ID:                  3,
				Type:                sdk.Openstack,
				Name:                "my-model-3",
				Group:               &sdk.Group{ID: 1, Name: "mygroup"},
				ModelVirtualMachine: sdk.ModelVirtualMachine{Flavor: "b2-7"},
			},
			{
				ID:                  4,
				Type:                sdk.Openstack,
				Name:                "my-model-4",
				Group:               &sdk.Group{ID: 1, Name: "mygroup"},
				ModelVirtualMachine: sdk.ModelVirtualMachine{Flavor: "unknown"},
			},
		}, nil
	})

	h.flavors = []flavors.Flavor{
		{Name: "b2-7", VCPUs: 2},
		{Name: "b2-30", VCPUs: 16},
		{Name: "b2-120", VCPUs: 32},
	}

	// Only model that match a known flavor should be returned and sorted by CPUs asc
	ms, err := h.WorkerModelsEnabled()
	require.NoError(t, err)
	require.Len(t, ms, 2)
	assert.Equal(t, "my-model-3", ms[0].Name)
	assert.Equal(t, "my-model-2", ms[1].Name)
}
