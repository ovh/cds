package openstack

import (
	"context"
	"testing"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

func TestHatcheryOpenstack_checkSpawnLimits_MaxWorker(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	h := &HatcheryOpenstack{}
	h.Config.Provision.MaxWorker = 3
	h.flavors = []flavors.Flavor{
		{Name: "my-flavor", VCPUs: 2},
	}

	m := sdk.Model{
		ID:                  1,
		Name:                "my-model",
		Group:               &sdk.Group{ID: 1, Name: "my-group"},
		ModelVirtualMachine: sdk.ModelVirtualMachine{Flavor: "my-flavor"},
	}

	lservers.list = []servers.Server{
		{Metadata: map[string]string{"flavor": "b2-30"}},
		{Metadata: map[string]string{"flavor": "b2-60"}},
	}

	err := h.checkSpawnLimits(context.TODO(), sdk.WorkerStarterWorkerModel{ModelV1: &m})
	require.NoError(t, err)

	lservers.list = []servers.Server{
		{Metadata: map[string]string{"flavor": "b2-30"}},
		{Metadata: map[string]string{"flavor": "b2-60"}},
		{Metadata: map[string]string{"flavor": "b2-120"}},
	}

	err = h.checkSpawnLimits(context.TODO(), sdk.WorkerStarterWorkerModel{ModelV1: &m})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MaxWorker")
}

func TestHatcheryOpenstack_checkSpawnLimits_MaxCPUs(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	h := &HatcheryOpenstack{}
	h.Config.Provision.MaxWorker = 10
	h.Config.MaxCPUs = 6
	h.flavors = []flavors.Flavor{
		{Name: "b2-7", VCPUs: 2},
	}

	m := sdk.Model{
		ID:                  1,
		Name:                "my-model",
		Group:               &sdk.Group{ID: 1, Name: "my-group"},
		ModelVirtualMachine: sdk.ModelVirtualMachine{Flavor: "b2-7"},
	}

	lservers.list = []servers.Server{
		{Metadata: map[string]string{"flavor": "b2-7"}},
		{Metadata: map[string]string{"flavor": "b2-7"}},
	}

	err := h.checkSpawnLimits(context.TODO(), sdk.WorkerStarterWorkerModel{ModelV1: &m})
	require.NoError(t, err)

	lservers.list = []servers.Server{
		{Metadata: map[string]string{"flavor": "b2-7"}},
		{Metadata: map[string]string{"flavor": "b2-7"}},
		{Metadata: map[string]string{"flavor": "b2-7"}},
	}

	err = h.checkSpawnLimits(context.TODO(), sdk.WorkerStarterWorkerModel{ModelV1: &m})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MaxCPUs")
}

func TestHatcheryOpenstack_checkSpawnLimits_CountSmallerFlavorToKeep(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	h := &HatcheryOpenstack{}
	h.Config.Provision.MaxWorker = 10
	h.Config.MaxCPUs = 30
	h.Config.CountSmallerFlavorToKeep = 2
	h.flavors = []flavors.Flavor{
		{ID: "1", Name: "b2-7", VCPUs: 2},
		{ID: "3", Name: "b2-30", VCPUs: 8},
		{ID: "2", Name: "b2-15", VCPUs: 4},
	}

	m1 := sdk.Model{
		ID:                  1,
		Name:                "my-model-1",
		Group:               &sdk.Group{ID: 1, Name: "my-group"},
		ModelVirtualMachine: sdk.ModelVirtualMachine{Flavor: "b2-7"},
	}
	m2 := sdk.Model{
		ID:                  2,
		Name:                "my-model-2",
		Group:               &sdk.Group{ID: 1, Name: "my-group"},
		ModelVirtualMachine: sdk.ModelVirtualMachine{Flavor: "b2-15"},
	}
	m3 := sdk.Model{
		ID:                  3,
		Name:                "my-model-3",
		Group:               &sdk.Group{ID: 1, Name: "my-group"},
		ModelVirtualMachine: sdk.ModelVirtualMachine{Flavor: "b2-30"},
	}

	lservers.list = []servers.Server{
		{Metadata: map[string]string{"flavor": "b2-30"}},
	}

	err := h.checkSpawnLimits(context.TODO(), sdk.WorkerStarterWorkerModel{ModelV1: &m3})
	require.NoError(t, err, "22 CPUs left (30-8) should be enough to start 8 CPUs flavor (8+4*2=16)")

	lservers.list = []servers.Server{
		{Metadata: map[string]string{"flavor": "b2-30"}},
		{Metadata: map[string]string{"flavor": "b2-30"}},
	}

	err = h.checkSpawnLimits(context.TODO(), sdk.WorkerStarterWorkerModel{ModelV1: &m3})
	require.Error(t, err, "14 CPUs left (30-8*2) should be not be enough to start 8 CPUs flavor (8+4*2=16)")
	assert.Contains(t, err.Error(), "CountSmallerFlavorToKeep")

	err = h.checkSpawnLimits(context.TODO(), sdk.WorkerStarterWorkerModel{ModelV1: &m2})
	require.NoError(t, err, "14 CPUs left (30-8*2) should be enough to start 4 CPUs flavor (4+2*2=8)")

	lservers.list = []servers.Server{
		{Metadata: map[string]string{"flavor": "b2-30"}},
		{Metadata: map[string]string{"flavor": "b2-30"}},
		{Metadata: map[string]string{"flavor": "b2-15"}},
		{Metadata: map[string]string{"flavor": "b2-15"}},
		{Metadata: map[string]string{"flavor": "b2-7"}},
		{Metadata: map[string]string{"flavor": "b2-7"}},
	}

	err = h.checkSpawnLimits(context.TODO(), sdk.WorkerStarterWorkerModel{ModelV1: &m1})
	require.NoError(t, err, "2 CPUs left (30-8*2-4*2-2*2) should be enough to start the smallest flavor with 2 CPUs")

	lservers.list = []servers.Server{
		{Metadata: map[string]string{"flavor": "b2-30"}},
		{Metadata: map[string]string{"flavor": "b2-30"}},
		{Metadata: map[string]string{"flavor": "b2-15"}},
		{Metadata: map[string]string{"flavor": "b2-15"}},
		{Metadata: map[string]string{"flavor": "b2-7"}},
		{Metadata: map[string]string{"flavor": "b2-7"}},
		{Metadata: map[string]string{"flavor": "b2-7"}},
	}

	err = h.checkSpawnLimits(context.TODO(), sdk.WorkerStarterWorkerModel{ModelV1: &m1})
	require.Error(t, err, "0 CPUs left to start new flavor")
	assert.Contains(t, err.Error(), "MaxCPUs limit")
}
