package vsphere

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/mock/gomock"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
)

// extraConfigMap flattens a VM ExtraConfig into key/value pairs.
func extraConfigMap(t *testing.T, opts []types.BaseOptionValue) map[string]string {
	t.Helper()
	res := make(map[string]string, len(opts))
	for _, o := range opts {
		ov, ok := o.(*types.OptionValue)
		require.True(t, ok, "unexpected option type %T", o)
		v, ok := ov.Value.(string)
		require.True(t, ok, "option %q has non-string value %T", ov.Key, ov.Value)
		res[ov.Key] = v
	}
	return res
}

// newGuestInfoCloneHatchery returns a hatchery whose "freebsd14" image is
// declared as a guestinfo model, with the clone-time vSphere calls mocked.
func newGuestInfoCloneHatchery(t *testing.T) *HatcheryVSphere {
	t.Helper()

	c := NewVSphereClientTest(t)
	h := &HatcheryVSphere{
		vSphereClient: c,
		Config: HatcheryConfiguration{
			Models: []ModelConfig{{ModelVMWare: "freebsd14", GuestInfo: true}},
		},
	}
	h.Config.VSphereNetworkString = "vbox-net"
	h.Config.VSphereCardName = "ethernet-card"
	h.Config.VSphereDatastoreString = "datastore"
	h.Config.DNS = "192.168.0.253"

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

	return h
}

// A guestinfo model gets its network through guestinfo keys and no customization
// spec at all: GOSC does not support its guest OS.
func TestHatcheryVSphere_prepareCloneSpec_guestInfo(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	h := newGuestInfoCloneHatchery(t)

	annot := annotation{VMwareModelPath: "freebsd14", WorkerName: "provision-v2-ip-192-168-0-3-foo"}
	ip := &ipResult{ip: "192.168.0.3", gateway: "192.168.0.254", subnetMask: "255.255.255.0"}
	cloneSpec, err := h.prepareCloneSpec(context.Background(), &object.VirtualMachine{}, &annot, ip)
	require.NoError(t, err)
	require.NotNil(t, cloneSpec)

	assert.Nil(t, cloneSpec.Customization, "a guestinfo clone must carry no customization spec")

	extra := extraConfigMap(t, cloneSpec.Config.ExtraConfig)
	assert.Equal(t, "192.168.0.3", extra["guestinfo.cds.net.ip"])
	assert.Equal(t, "255.255.255.0", extra["guestinfo.cds.net.mask"])
	assert.Equal(t, "192.168.0.254", extra["guestinfo.cds.net.gateway"])
	assert.Equal(t, "192.168.0.253", extra["guestinfo.cds.net.dns"])
	assert.Equal(t, "provision-v2-ip-192-168-0-3-foo", extra["guestinfo.cds.net.hostname"])

	// Debug access is opt-in: unconfigured means the worker has no SSH access.
	assert.NotContains(t, extra, "guestinfo.cds.access.allowed_cidrs")
	assert.NotContains(t, extra, "guestinfo.cds.access.authorized_keys")

	// The IP accounting anchor must survive: getUsedIPs reads it back.
	assert.Equal(t, "192.168.0.3", annot.IPAddress)
}

// Debug access is two independent layers, and both must reach the guest: the
// CIDRs drive its packet filter, the keys drive sshd. Emitting only one would
// leave the worker either unreachable or filtered open.
func TestHatcheryVSphere_prepareCloneSpec_guestInfoAccess(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	h := newGuestInfoCloneHatchery(t)
	h.Config.SSHAllowedCIDRs = []string{"10.0.0.0/24", "10.1.0.0/24"}
	h.Config.InjectSSHPublicKeys = []string{
		`from="10.0.0.0/24" ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA op@example`,
		`from="10.1.0.0/24" ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB ops@example`,
	}

	annot := annotation{VMwareModelPath: "freebsd14", WorkerName: "provision-v2-worker"}
	ip := &ipResult{ip: "192.168.0.3", gateway: "192.168.0.254", subnetMask: "255.255.255.0"}
	cloneSpec, err := h.prepareCloneSpec(context.Background(), &object.VirtualMachine{}, &annot, ip)
	require.NoError(t, err)

	extra := extraConfigMap(t, cloneSpec.Config.ExtraConfig)
	// Space-separated: the guest iterates the value word by word.
	assert.Equal(t, "10.0.0.0/24 10.1.0.0/24", extra["guestinfo.cds.access.allowed_cidrs"])
	// Newline-separated: the value is dropped in as an authorized_keys body.
	assert.Equal(t, strings.Join(h.Config.InjectSSHPublicKeys, "\n"), extra["guestinfo.cds.access.authorized_keys"])
}

// Injected keys must state which source addresses may use them: a key without a
// from= option would be usable from anywhere.
func TestHatcheryVSphere_CheckConfiguration_injectSSHPublicKeys(t *testing.T) {
	h := New()
	cfg := HatcheryConfiguration{
		VSphereUser:             "user",
		VSphereEndpoint:         "endpoint",
		VSpherePassword:         "password",
		VSphereDatacenterString: "datacenter",
	}
	cfg.Name = "hatchery"
	cfg.API.HTTP.URL = "http://localhost:8081"
	cfg.API.Token = "xxx"
	cfg.Provision.MaxWorker = 1

	key := `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA op@example`

	cfg.InjectSSHPublicKeys = []string{key}
	require.Error(t, h.CheckConfiguration(cfg), "a key without a from= option must be rejected")

	cfg.InjectSSHPublicKeys = []string{`from="10.0.0.0/24" ` + key}
	require.NoError(t, h.CheckConfiguration(cfg))

	// A malformed CIDR would produce a broken guest allowlist.
	cfg.SSHAllowedCIDRs = []string{"10.0.0.0/24", "not-a-cidr"}
	require.Error(t, h.CheckConfiguration(cfg))

	cfg.SSHAllowedCIDRs = []string{"10.0.0.0/24"}
	require.NoError(t, h.CheckConfiguration(cfg))
}

// A model that is not declared as guestinfo keeps the guest customization path,
// with no guestinfo key emitted.
func TestHatcheryVSphere_prepareCloneSpec_linuxUnchanged(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	h := newGuestInfoCloneHatchery(t)
	h.Config.InjectSSHPublicKeys = []string{`from="10.0.0.0/24" ssh-ed25519 AAAAC3Nza op@example`}

	annot := annotation{VMwareModelPath: "debian12", WorkerName: "provision-v2-worker"}
	ip := &ipResult{ip: "192.168.0.3", gateway: "192.168.0.254", subnetMask: "255.255.255.0"}
	cloneSpec, err := h.prepareCloneSpec(context.Background(), &object.VirtualMachine{}, &annot, ip)
	require.NoError(t, err)

	require.NotNil(t, cloneSpec.Customization)
	assert.IsType(t, &types.CustomizationLinuxPrep{}, cloneSpec.Customization.Identity)
	assert.Equal(t, "192.168.0.3", cloneSpec.Customization.NicSettingMap[0].Adapter.Ip.(*types.CustomizationFixedIp).IpAddress)
	assert.Equal(t, "192.168.0.254", cloneSpec.Customization.NicSettingMap[0].Adapter.Gateway[0])
	assert.Empty(t, cloneSpec.Config.ExtraConfig, "a Linux clone must emit no guestinfo key")
	assert.Equal(t, "192.168.0.3", annot.IPAddress)
}

// Spawning a guestinfo worker pushes the bootstrap through ReconfigureVirtualMachine
// before power-on and never touches the guest-operations channel. The mock
// controller fails the test on any unexpected ProcessManager/StartProgramInGuest.
func TestHatcheryVSphere_SpawnWorkerGuestInfo(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	c := NewVSphereClientTest(t)
	h := HatcheryVSphere{
		vSphereClient: c,
		Config: HatcheryConfiguration{
			// No credentials anywhere: a guestinfo model must need none.
			Models: []ModelConfig{{ModelVMWare: "freebsd14", GuestInfo: true}},
		},
	}
	h.Config.Provision.InjectEnvVars = []string{"SOME_SECRET=s3cr3t"}

	var ctx = context.Background()
	var vmProvisionned = object.VirtualMachine{
		Common: object.Common{InventoryPath: "provision-v2-worker"},
	}

	c.EXPECT().ListVirtualMachines(gomock.Any()).DoAndReturn(func(ctx context.Context) ([]mo.VirtualMachine, error) {
		return []mo.VirtualMachine{
			{
				ManagedEntity: mo.ManagedEntity{Name: "provision-v2-worker"},
				Summary: types.VirtualMachineSummary{
					Runtime: types.VirtualMachineRuntimeInfo{
						PowerState: types.VirtualMachinePowerStatePoweredOff,
					},
				},
				Config: &types.VirtualMachineConfigInfo{
					Annotation: `{"provisioning": true, "vmware_model_path": "freebsd14", "ip_address": "192.168.0.1"}`,
				},
			},
		}, nil
	}).Times(2)

	c.EXPECT().LoadVirtualMachine(gomock.Any(), "provision-v2-worker").Return(&vmProvisionned, nil)

	c.EXPECT().LoadVirtualMachineEvents(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine, eventTypes ...string) ([]types.BaseEvent, error) {
			return []types.BaseEvent{
				&types.VmPoweredOffEvent{VmEvent: types.VmEvent{Event: types.Event{
					CreatedTime: time.Now().Add(-10 * time.Minute),
				}}},
			}, nil
		},
	)

	rename := c.EXPECT().RenameVirtualMachine(gomock.Any(), &vmProvisionned, "worker-name").Return(nil)

	annotated := c.EXPECT().SetVirtualMachineAnnotation(gomock.Any(), &vmProvisionned, gomock.Any()).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine, annotStr string) error {
			var a annotation
			require.NoError(t, json.Unmarshal([]byte(annotStr), &a))
			assert.Equal(t, "192.168.0.1", a.IPAddress, "reserved IP must be preserved")
			return nil
		},
	)

	bootstrapped := c.EXPECT().ReconfigureVirtualMachine(gomock.Any(), &vmProvisionned, gomock.Any()).DoAndReturn(
		func(ctx context.Context, vm *object.VirtualMachine, spec types.VirtualMachineConfigSpec) error {
			extra := extraConfigMap(t, spec.ExtraConfig)

			assert.Contains(t, extra["guestinfo.cds.cmd"], "./worker")
			assert.Contains(t, extra["guestinfo.cds.cmd"], "1>/tmp/worker.log 2>&1;")
			assert.Contains(t, extra["guestinfo.cds.cmd"], "shutdown -p now")

			require.Contains(t, extra, "guestinfo.cds.config")
			raw, err := base64.StdEncoding.DecodeString(extra["guestinfo.cds.config"])
			require.NoError(t, err, "guestinfo.cds.config must be base64")
			var cfg map[string]interface{}
			require.NoError(t, json.Unmarshal(raw, &cfg), "guestinfo.cds.config must decode to JSON")
			assert.Equal(t, "worker-name", cfg["name"])

			// The VM was renamed out of its provision name; the guest must apply
			// the worker name on this boot.
			assert.Equal(t, "worker-name", extra["guestinfo.cds.net.hostname"])

			// Injected env vars travel inside the config, which the guest scrubs
			// after use. A separate key would never be read and would outlive it.
			assert.Equal(t, map[string]interface{}{"SOME_SECRET": "s3cr3t"}, cfg["inject_env_vars"])
			for k, v := range extra {
				assert.NotContains(t, v, "s3cr3t", "secret leaked in clear text into %q", k)
			}
			return nil
		},
	)

	// The bootstrap is worthless if the guest boots before it lands.
	started := c.EXPECT().StartVirtualMachine(gomock.Any(), &vmProvisionned).Return(nil)
	gomock.InOrder(rename, annotated, bootstrapped, started)

	// Only SpawnWorker waits for the IP: launchScriptWorker is skipped entirely.
	c.EXPECT().WaitForVirtualMachineIP(gomock.Any(), &vmProvisionned, gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := h.SpawnWorker(ctx, hatchery.SpawnArguments{
		WorkerName:  "worker-name",
		WorkerToken: "worker.token.xxx",
		Model: sdk.WorkerStarterWorkerModel{
			ModelV2: &sdk.V2WorkerModel{
				Name:   "cds-freebsd-model",
				OSArch: "freebsd/amd64",
				Spec:   json.RawMessage(`{"image":"freebsd14"}`),
			},
			VSphereSpec: sdk.V2WorkerModelVSphereSpec{Image: "freebsd14"},
			Cmd:         "./worker",
			PostCmd:     "shutdown -p now",
		},
		JobName:      "job_name",
		JobID:        "666",
		HatcheryName: "hatchery_name",
	})
	require.NoError(t, err)
}
