package vsphere

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/mock/gomock"

	"github.com/ovh/cds/engine/hatchery/vsphere/mock_vsphere"
)

func TestPrepareCloneSpecWithFlavor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_vsphere.NewMockVSphereClient(ctrl)
	h := &HatcheryVSphere{
		vSphereClient:        mockClient,
		availableIPAddresses: []string{},
		Config: HatcheryConfiguration{
			VSphereNetworkString:   "VM Network",
			VSphereDatastoreString: "datastore1",
			VSphereCardName:        "e1000",
		},
	}

	// Mock expectations
	mockClient.EXPECT().LoadVirtualMachineDevices(gomock.Any(), gomock.Any()).Return(
		[]types.BaseVirtualDevice{
			&types.VirtualE1000{
				VirtualEthernetCard: types.VirtualEthernetCard{
					VirtualDevice: types.VirtualDevice{},
				},
			},
		}, nil)
	mockClient.EXPECT().LoadNetwork(gomock.Any(), "VM Network").Return(&object.Network{}, nil)
	mockClient.EXPECT().SetupEthernetCard(gomock.Any(), gomock.Any(), "e1000", gomock.Any()).Return(nil)
	mockClient.EXPECT().LoadResourcePool(gomock.Any()).Return(&object.ResourcePool{}, nil)
	mockClient.EXPECT().LoadDatastore(gomock.Any(), "datastore1").Return(&object.Datastore{}, nil)

	flavor := &VSphereFlavorConfig{
		CPUs:     8,
		MemoryMB: 16384,
	}

	ctx := context.Background()
	cloneSpec, err := h.prepareCloneSpec(ctx, &object.VirtualMachine{}, &annotation{}, flavor)
	require.NoError(t, err)
	require.NotNil(t, cloneSpec)
	require.NotNil(t, cloneSpec.Config)

	// Assert flavor was applied
	assert.Equal(t, int32(8), cloneSpec.Config.NumCPUs)
	assert.Equal(t, int64(16384), cloneSpec.Config.MemoryMB)
}

func TestPrepareCloneSpecWithoutFlavor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_vsphere.NewMockVSphereClient(ctrl)
	h := &HatcheryVSphere{
		vSphereClient:        mockClient,
		availableIPAddresses: []string{},
		Config: HatcheryConfiguration{
			VSphereNetworkString:   "VM Network",
			VSphereDatastoreString: "datastore1",
			VSphereCardName:        "e1000",
		},
	}

	// Mock expectations
	mockClient.EXPECT().LoadVirtualMachineDevices(gomock.Any(), gomock.Any()).Return(
		[]types.BaseVirtualDevice{
			&types.VirtualE1000{
				VirtualEthernetCard: types.VirtualEthernetCard{
					VirtualDevice: types.VirtualDevice{},
				},
			},
		}, nil)
	mockClient.EXPECT().LoadNetwork(gomock.Any(), "VM Network").Return(&object.Network{}, nil)
	mockClient.EXPECT().SetupEthernetCard(gomock.Any(), gomock.Any(), "e1000", gomock.Any()).Return(nil)
	mockClient.EXPECT().LoadResourcePool(gomock.Any()).Return(&object.ResourcePool{}, nil)
	mockClient.EXPECT().LoadDatastore(gomock.Any(), "datastore1").Return(&object.Datastore{}, nil)

	ctx := context.Background()
	cloneSpec, err := h.prepareCloneSpec(ctx, &object.VirtualMachine{}, &annotation{}, nil)
	require.NoError(t, err)
	require.NotNil(t, cloneSpec)
	require.NotNil(t, cloneSpec.Config)

	// Assert no flavor was applied (NumCPUs/MemoryMB should be zero/unset)
	assert.Equal(t, int32(0), cloneSpec.Config.NumCPUs)
	assert.Equal(t, int64(0), cloneSpec.Config.MemoryMB)
}

func TestGetSmallerFlavorCPUs(t *testing.T) {
	h := &HatcheryVSphere{
		Config: HatcheryConfiguration{
			Flavors: []VSphereFlavorConfig{
				{Name: "small", CPUs: 2, MemoryMB: 4096},
				{Name: "medium", CPUs: 4, MemoryMB: 8192},
				{Name: "large", CPUs: 8, MemoryMB: 16384},
				{Name: "xlarge", CPUs: 16, MemoryMB: 32768},
			},
		},
	}

	// Test smallest flavor for large (should return 2, not 4)
	smallestCPUs := h.getSmallerFlavorCPUs("large")
	assert.Equal(t, 2, smallestCPUs, "should return smallest flavor (small=2), not largest smaller (medium=4)")

	// Test smallest flavor for xlarge (should return 2)
	smallestCPUs = h.getSmallerFlavorCPUs("xlarge")
	assert.Equal(t, 2, smallestCPUs)

	// Test smallest flavor for medium (should return 2)
	smallestCPUs = h.getSmallerFlavorCPUs("medium")
	assert.Equal(t, 2, smallestCPUs)

	// Test smallest flavor for small (no smaller flavor exists)
	smallestCPUs = h.getSmallerFlavorCPUs("small")
	assert.Equal(t, 0, smallestCPUs, "should return 0 when no smaller flavor exists")

	// Test unknown flavor
	smallestCPUs = h.getSmallerFlavorCPUs("unknown")
	assert.Equal(t, 0, smallestCPUs)
}
