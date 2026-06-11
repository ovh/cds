package vsphere

import (
	"context"
	"testing"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/mock/gomock"
)

func TestHatcheryVSphere_findAvailableIP_MultipleNetworks(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}

	// Configure two networks with different gateways and subnets
	h.availableIPAddresses = []string{
		"10.0.1.1", "10.0.1.2", "10.0.1.3",
		"10.0.2.1", "10.0.2.2", "10.0.2.3",
	}
	h.availableNetworks = []availableNetwork{
		{
			config: NetworkConfig{
				IPRange:    "10.0.1.0/29",
				Gateway:    "10.0.1.254",
				SubnetMask: "255.255.255.0",
			},
			ipAddresses: []string{"10.0.1.1", "10.0.1.2", "10.0.1.3"},
		},
		{
			config: NetworkConfig{
				IPRange:    "10.0.2.0/29",
				Gateway:    "10.0.2.254",
				SubnetMask: "255.255.248.0",
			},
			ipAddresses: []string{"10.0.2.1", "10.0.2.2", "10.0.2.3"},
		},
	}

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(
		func(ctx context.Context) ([]mo.VirtualMachine, error) {
			return []mo.VirtualMachine{}, nil
		},
	).AnyTimes()

	ctx := context.Background()

	// First available IP should come from the first network
	result, err := h.findAvailableIP(ctx)
	require.NoError(t, err)
	assert.Equal(t, "10.0.1.1", result.ip)
	assert.Equal(t, "10.0.1.254", result.gateway)
	assert.Equal(t, "255.255.255.0", result.subnetMask)
}

func TestHatcheryVSphere_findAvailableIP_FallsToSecondNetwork(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}

	// First network is fully reserved, second has availability
	h.availableIPAddresses = []string{
		"10.0.1.1", "10.0.1.2",
		"10.0.2.1", "10.0.2.2",
	}
	h.availableNetworks = []availableNetwork{
		{
			config: NetworkConfig{
				IPRange:    "10.0.1.0/30",
				Gateway:    "10.0.1.254",
				SubnetMask: "255.255.255.0",
			},
			ipAddresses: []string{"10.0.1.1", "10.0.1.2"},
		},
		{
			config: NetworkConfig{
				IPRange:    "10.0.2.0/30",
				Gateway:    "10.0.2.254",
				SubnetMask: "255.255.248.0",
			},
			ipAddresses: []string{"10.0.2.1", "10.0.2.2"},
		},
	}
	// Reserve all IPs from first network
	h.reservedIPAddresses = []string{"10.0.1.1", "10.0.1.2"}

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(
		func(ctx context.Context) ([]mo.VirtualMachine, error) {
			return []mo.VirtualMachine{}, nil
		},
	).AnyTimes()

	ctx := context.Background()

	// Should fall to second network
	result, err := h.findAvailableIP(ctx)
	require.NoError(t, err)
	assert.Equal(t, "10.0.2.1", result.ip)
	assert.Equal(t, "10.0.2.254", result.gateway)
	assert.Equal(t, "255.255.248.0", result.subnetMask)
}

func TestHatcheryVSphere_prepareCloneSpec_MultiNetwork(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}
	h.Config.VSphereNetworkString = "vbox-net"
	h.Config.VSphereCardName = "ethernet-card"
	h.Config.VSphereDatastoreString = "datastore"
	h.Config.DNS = "8.8.8.8"

	// First network exhausted (IPs used by VMs), second network has availability
	h.availableIPAddresses = []string{
		"10.0.1.1",
		"10.0.2.1", "10.0.2.2",
	}
	h.availableNetworks = []availableNetwork{
		{
			config: NetworkConfig{
				IPRange:    "10.0.1.0/30",
				Gateway:    "10.0.1.254",
				SubnetMask: "255.255.255.0",
			},
			ipAddresses: []string{"10.0.1.1"},
		},
		{
			config: NetworkConfig{
				IPRange:    "10.0.2.0/30",
				Gateway:    "10.0.2.254",
				SubnetMask: "255.255.248.0",
			},
			ipAddresses: []string{"10.0.2.1", "10.0.2.2"},
		},
	}

	c.EXPECT().LoadVirtualMachineDevices(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) (object.VirtualDeviceList, error) {
			card := types.VirtualEthernetCard{}
			return object.VirtualDeviceList{&card}, nil
		},
	)

	c.EXPECT().LoadNetwork(gomock.Any(), "vbox-net").DoAndReturn(
		func(ctx context.Context, s string) (object.NetworkReference, error) {
			return &object.Network{}, nil
		},
	)

	c.EXPECT().SetupEthernetCard(gomock.Any(), gomock.Any(), "ethernet-card", gomock.Any()).DoAndReturn(
		func(ctx context.Context, card *types.VirtualEthernetCard, ethernetCardName string, network object.NetworkReference) error {
			return nil
		},
	)

	c.EXPECT().LoadResourcePool(gomock.Any()).DoAndReturn(
		func(ctx context.Context) (*object.ResourcePool, error) {
			return &object.ResourcePool{}, nil
		},
	)

	c.EXPECT().LoadDatastore(gomock.Any(), "datastore").DoAndReturn(
		func(ctx context.Context, name string) (*object.Datastore, error) {
			return &object.Datastore{}, nil
		},
	)

	// Simulate 10.0.1.1 already in use by a VM
	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(
		func(ctx context.Context) ([]mo.VirtualMachine, error) {
			return []mo.VirtualMachine{
				{
					Summary: types.VirtualMachineSummary{
						Config: types.VirtualMachineConfigSummary{Template: false},
					},
					Guest: &types.GuestInfo{
						Net: []types.GuestNicInfo{
							{IpAddress: []string{"10.0.1.1"}},
						},
					},
				},
			}, nil
		},
	).AnyTimes()

	ctx := context.Background()
	cloneSpec, err := h.prepareCloneSpec(ctx, &object.VirtualMachine{}, &annotation{})
	require.NoError(t, err)
	require.NotNil(t, cloneSpec)

	// Should use IP from second network since first is exhausted
	assert.Equal(t, "10.0.2.1", cloneSpec.Customization.NicSettingMap[0].Adapter.Ip.(*types.CustomizationFixedIp).IpAddress)
	// Gateway and subnet from second network
	assert.Equal(t, "10.0.2.254", cloneSpec.Customization.NicSettingMap[0].Adapter.Gateway[0])
	assert.Equal(t, "255.255.248.0", cloneSpec.Customization.NicSettingMap[0].Adapter.SubnetMask)
	// DNS still global
	assert.Equal(t, "8.8.8.8", cloneSpec.Customization.GlobalIPSettings.DnsServerList[0])
}

func TestHatcheryVSphere_initNetworks_LegacyConfig(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	h := HatcheryVSphere{}
	h.Config.IPRange = "192.168.1.0/30"
	h.Config.Gateway = "192.168.1.254"
	h.Config.SubnetMask = "255.255.255.0"

	ctx := context.Background()
	err := h.initNetworks(ctx)
	require.NoError(t, err)

	require.Len(t, h.availableNetworks, 1)
	assert.Equal(t, "192.168.1.254", h.availableNetworks[0].config.Gateway)
	assert.Equal(t, "255.255.255.0", h.availableNetworks[0].config.SubnetMask)
	assert.True(t, len(h.availableNetworks[0].ipAddresses) > 0)
	assert.Equal(t, h.availableNetworks[0].ipAddresses, h.availableIPAddresses)
}

func TestHatcheryVSphere_initNetworks_NewConfig(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	h := HatcheryVSphere{}
	h.Config.Networks = []NetworkConfig{
		{
			IPRange:    "10.0.1.0/30",
			Gateway:    "10.0.1.1",
			SubnetMask: "255.255.255.0",
		},
		{
			IPRange:    "10.0.2.0/30",
			Gateway:    "10.0.2.1",
			SubnetMask: "255.255.248.0",
		},
	}

	ctx := context.Background()
	err := h.initNetworks(ctx)
	require.NoError(t, err)

	require.Len(t, h.availableNetworks, 2)
	assert.Equal(t, "10.0.1.1", h.availableNetworks[0].config.Gateway)
	assert.Equal(t, "255.255.255.0", h.availableNetworks[0].config.SubnetMask)
	assert.Equal(t, "10.0.2.1", h.availableNetworks[1].config.Gateway)
	assert.Equal(t, "255.255.248.0", h.availableNetworks[1].config.SubnetMask)

	// availableIPAddresses should contain IPs from both networks
	totalIPs := len(h.availableNetworks[0].ipAddresses) + len(h.availableNetworks[1].ipAddresses)
	assert.Equal(t, totalIPs, len(h.availableIPAddresses))
}

func TestHatcheryVSphere_initNetworks_NewConfigOverridesLegacy(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	h := HatcheryVSphere{}
	// Set both legacy and new config - new should take precedence
	h.Config.IPRange = "192.168.1.0/30"
	h.Config.Gateway = "192.168.1.254"
	h.Config.SubnetMask = "255.255.255.0"
	h.Config.Networks = []NetworkConfig{
		{
			IPRange:    "10.0.1.0/30",
			Gateway:    "10.0.1.1",
			SubnetMask: "255.255.248.0",
		},
	}

	ctx := context.Background()
	err := h.initNetworks(ctx)
	require.NoError(t, err)

	// Networks config takes precedence over legacy
	require.Len(t, h.availableNetworks, 1)
	assert.Equal(t, "10.0.1.1", h.availableNetworks[0].config.Gateway)
	assert.Equal(t, "255.255.248.0", h.availableNetworks[0].config.SubnetMask)
}
