package openstack

import (
	"context"
	"testing"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
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

	flavors := []flavors.Flavor{
		{Name: "b2-7"},
	}
	h.flavors = flavors

	m := &sdk.Model{
		ID:   1,
		Name: "my-model",
		Group: &sdk.Group{
			ID:   1,
			Name: "mygroup",
		},
		ModelVirtualMachine: sdk.ModelVirtualMachine{
			Flavor: "vps-ssd-3",
		},
	}

	// model with a unknowned flavor
	canSpawn = h.CanSpawn(context.TODO(), m, 1, nil)
	require.False(t, canSpawn)

	m = &sdk.Model{
		ID:   1,
		Name: "my-model",
		Group: &sdk.Group{
			ID:   1,
			Name: "mygroup",
		},
		ModelVirtualMachine: sdk.ModelVirtualMachine{
			Flavor: "b2-7",
		},
	}

	// model with a knowned flavor
	canSpawn = h.CanSpawn(context.TODO(), m, 1, nil)
	require.True(t, canSpawn)
}
