package vsphere

import (
	"context"
	"testing"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func TestIPFromProvisionName(t *testing.T) {
	encoded := provisionName(&ipResult{ip: "10.0.0.5"})
	assert.Contains(t, encoded, "provision-v2-ip-10-0-0-5-")

	ip, ok := ipFromProvisionName(encoded)
	assert.True(t, ok)
	assert.Equal(t, "10.0.0.5", ip)

	// A plain provision name (DHCP mode / old style) has no IP.
	_, ok = ipFromProvisionName("provision-v2-nervous-but-tender-pascal")
	assert.False(t, ok)

	// A claimed worker name has no IP.
	_, ok = ipFromProvisionName("worker-abc")
	assert.False(t, ok)
}

// getUsedIPs must see an in-flight provision's IP from its NAME (visible during
// the clone, before the annotation is populated), and an old-style/claimed VM's
// IP from its annotation or guest network.
func TestHatcheryVSphere_getUsedIPs(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	h := HatcheryVSphere{}

	srvs := []mo.VirtualMachine{
		{ // cloning provision: only the IP-encoded name, no annotation yet
			ManagedEntity: mo.ManagedEntity{Name: "provision-v2-ip-10-0-0-1-foo"},
		},
		{ // old-style provision: IP only in the annotation
			ManagedEntity: mo.ManagedEntity{Name: "provision-v2-bar"},
			Config:        &types.VirtualMachineConfigInfo{Annotation: `{"ip_address": "10.0.0.2"}`},
		},
		{ // running worker: IP on the guest network
			ManagedEntity: mo.ManagedEntity{Name: "worker-baz"},
			Guest:         &types.GuestInfo{Net: []types.GuestNicInfo{{IpAddress: []string{"10.0.0.3"}}}},
		},
	}

	used := h.getUsedIPs(context.Background(), srvs)
	for _, ip := range []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"} {
		_, ok := used[ip]
		assert.Truef(t, ok, "expected %s to be counted as used", ip)
	}
}

// pickFreeIP returns distinct IPs as the caller marks each one used.
func TestHatcheryVSphere_pickFreeIP_distinct(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	h := HatcheryVSphere{}
	h.availableNetworks = []availableNetwork{{
		config:      NetworkConfig{Gateway: "10.0.0.254", SubnetMask: "255.255.255.0"},
		ipAddresses: []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"},
	}}

	used := map[string]struct{}{}
	seen := map[string]bool{}
	for i := 0; i < 3; i++ {
		res, ok := h.pickFreeIP(used)
		assert.True(t, ok)
		assert.Falsef(t, seen[res.ip], "IP %s handed out twice", res.ip)
		seen[res.ip] = true
		used[res.ip] = struct{}{}
	}
	_, ok := h.pickFreeIP(used)
	assert.False(t, ok, "no IP should remain after all are used")
}
