package vsphere

import (
	"context"
	"testing"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/mock/gomock"
)

func TestHatcheryVSphere_pickFreeIP_MultipleNetworks(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	h := HatcheryVSphere{}
	h.availableNetworks = []availableNetwork{
		{
			config:      NetworkConfig{Gateway: "10.0.1.254", SubnetMask: "255.255.255.0"},
			ipAddresses: []string{"10.0.1.1", "10.0.1.2", "10.0.1.3"},
		},
		{
			config:      NetworkConfig{Gateway: "10.0.2.254", SubnetMask: "255.255.248.0"},
			ipAddresses: []string{"10.0.2.1", "10.0.2.2", "10.0.2.3"},
		},
	}

	// First free IP comes from the first network.
	result, ok := h.pickFreeIP(map[string]struct{}{})
	require.True(t, ok)
	assert.Equal(t, "10.0.1.1", result.ip)
	assert.Equal(t, "10.0.1.254", result.gateway)
	assert.Equal(t, "255.255.255.0", result.subnetMask)
}

func TestHatcheryVSphere_pickFreeIP_FallsToSecondNetwork(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	h := HatcheryVSphere{}
	h.availableNetworks = []availableNetwork{
		{
			config:      NetworkConfig{Gateway: "10.0.1.254", SubnetMask: "255.255.255.0"},
			ipAddresses: []string{"10.0.1.1", "10.0.1.2"},
		},
		{
			config:      NetworkConfig{Gateway: "10.0.2.254", SubnetMask: "255.255.248.0"},
			ipAddresses: []string{"10.0.2.1", "10.0.2.2"},
		},
	}

	// All IPs of the first network are taken → fall to the second.
	used := map[string]struct{}{"10.0.1.1": {}, "10.0.1.2": {}}
	result, ok := h.pickFreeIP(used)
	require.True(t, ok)
	assert.Equal(t, "10.0.2.1", result.ip)
	assert.Equal(t, "10.0.2.254", result.gateway)
	assert.Equal(t, "255.255.248.0", result.subnetMask)

	// Exhaust everything → no free IP.
	for _, n := range h.availableNetworks {
		for _, ip := range n.ipAddresses {
			used[ip] = struct{}{}
		}
	}
	_, ok = h.pickFreeIP(used)
	assert.False(t, ok)
}

func TestHatcheryVSphere_prepareCloneSpec_UsesGivenIP(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
	}
	h.Config.VSphereNetworkString = "vbox-net"
	h.Config.VSphereCardName = "ethernet-card"
	h.Config.VSphereDatastoreString = "datastore"
	h.Config.DNS = "8.8.8.8"

	c.EXPECT().LoadVirtualMachineDevices(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine) (object.VirtualDeviceList, error) {
			card := types.VirtualEthernetCard{}
			return object.VirtualDeviceList{&card}, nil
		},
	)
	c.EXPECT().LoadNetwork(gomock.Any(), "vbox-net").Return(&object.Network{}, nil)
	c.EXPECT().SetupEthernetCard(gomock.Any(), gomock.Any(), "ethernet-card", gomock.Any()).Return(nil)
	c.EXPECT().LoadResourcePool(gomock.Any()).Return(&object.ResourcePool{}, nil)
	c.EXPECT().LoadDatastore(gomock.Any(), "datastore").Return(&object.Datastore{}, nil)

	ctx := context.Background()
	annot := annotation{}
	ip := &ipResult{ip: "10.0.2.1", gateway: "10.0.2.254", subnetMask: "255.255.248.0"}
	cloneSpec, err := h.prepareCloneSpec(ctx, &object.VirtualMachine{}, &annot, ip)
	require.NoError(t, err)
	require.NotNil(t, cloneSpec)

	// The clone spec uses the IP chosen by the caller, and the IP is recorded in
	// the annotation (the compatibility anchor).
	assert.Equal(t, "10.0.2.1", cloneSpec.Customization.NicSettingMap[0].Adapter.Ip.(*types.CustomizationFixedIp).IpAddress)
	assert.Equal(t, "10.0.2.254", cloneSpec.Customization.NicSettingMap[0].Adapter.Gateway[0])
	assert.Equal(t, "255.255.248.0", cloneSpec.Customization.NicSettingMap[0].Adapter.SubnetMask)
	assert.Equal(t, "8.8.8.8", cloneSpec.Customization.GlobalIPSettings.DnsServerList[0])
	assert.Equal(t, "10.0.2.1", annot.IPAddress)
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
