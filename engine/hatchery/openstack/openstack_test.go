package openstack

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/ssh"
)

func TestHatcheryOpenstack_CanSpawn(t *testing.T) {
	h := &HatcheryOpenstack{}
	h.cache = NewCache(1, 1)

	// no model, no requirement, canSpawn must be true
	canSpawn := h.CanSpawn(context.TODO(), sdk.WorkerStarterWorkerModel{}, "1", nil)
	require.True(t, canSpawn)

	// no model, service requirement, canSpawn must be false: service can't be managed by openstack hatchery
	canSpawn = h.CanSpawn(context.TODO(), sdk.WorkerStarterWorkerModel{}, "1", []sdk.Requirement{{Name: "pg", Type: sdk.ServiceRequirement, Value: "postgres:9.5.4"}})
	require.False(t, canSpawn)

	// no model, memory prerequisite, canSpawn must be false: memory prerequisite can't be managed by openstack hatchery
	canSpawn = h.CanSpawn(context.TODO(), sdk.WorkerStarterWorkerModel{}, "1", []sdk.Requirement{{Name: "mem", Type: sdk.MemoryRequirement, Value: "4096"}})
	require.False(t, canSpawn)

	// no model, hostname prerequisite, canSpawn must be false: hostname can't be managed by openstack hatchery
	canSpawn = h.CanSpawn(context.TODO(), sdk.WorkerStarterWorkerModel{}, "1", []sdk.Requirement{{Type: sdk.HostnameRequirement, Value: "localhost"}})
	require.False(t, canSpawn)
}

func TestHatcheryOpenstack_WorkerModelsEnabled(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	h := &HatcheryOpenstack{
		Config: HatcheryConfiguration{
			DefaultFlavor: "XL",
			Flavors: map[string]string{
				"xs": "d2-2",
				"s":  "b2-7",
				"l":  "b2-120",
				"xl": "b2-240",
			},
			OldFlavorsMapping: map[string]string{
				"d2-2":   "xs",
				"b2-7":   "s",
				"b2-120": "l",
			},
		},
	}
	h.cache = NewCache(1, 1)

	ctrl := gomock.NewController(t)
	mockClient := mock_cdsclient.NewMockInterface(ctrl)
	h.Client = mockClient
	t.Cleanup(func() { ctrl.Finish() })

	mockClient.EXPECT().WorkerModelEnabledList().DoAndReturn(func() ([]sdk.Model, error) {
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
			{
				ID:                  5,
				Type:                sdk.Openstack,
				Name:                "my-model-5",
				Group:               &sdk.Group{ID: 1, Name: "mygroup"},
				ModelVirtualMachine: sdk.ModelVirtualMachine{Flavor: "d2-2"},
			},
		}, nil
	})

	h.flavors = []flavors.Flavor{
		{Name: "b2-7", VCPUs: 4},
		{Name: "b2-30", VCPUs: 16},
		{Name: "b2-120", VCPUs: 32},
		{Name: "d2-2", VCPUs: 2},
		{Name: "s1-4", VCPUs: 1},
	}

	// Only model that match a known flavor should be returned and sorted by CPUs asc
	ms, err := h.WorkerModelsEnabled()
	require.NoError(t, err)
	require.Len(t, ms, 3)
	assert.Equal(t, "my-model-5", ms[0].Name)
	assert.Equal(t, "my-model-3", ms[1].Name)
	assert.Equal(t, "my-model-2", ms[2].Name)
}

func TestHatcheryOpenstack_WorkerModelsEnabled_BackwardCompatibility(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	h := &HatcheryOpenstack{
		Config: HatcheryConfiguration{
			DefaultFlavor: "XXS",
			Flavors: map[string]string{
				"xxs": "s1-4",
				"xs":  "d2-4",
				"s":   "b3-8",
				"l":   "b3-32",
			},
			OldFlavorsMapping: map[string]string{
				"d2-2":   "xs",
				"b2-7":   "s",
				"b2-120": "l",
			},
		},
	}
	h.cache = NewCache(1, 1)

	ctrl := gomock.NewController(t)
	mockClient := mock_cdsclient.NewMockInterface(ctrl)
	h.Client = mockClient
	t.Cleanup(func() { ctrl.Finish() })

	mockClient.EXPECT().WorkerModelEnabledList().DoAndReturn(func() ([]sdk.Model, error) {
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
			{
				ID:                  5,
				Type:                sdk.Openstack,
				Name:                "my-model-5",
				Group:               &sdk.Group{ID: 1, Name: "mygroup"},
				ModelVirtualMachine: sdk.ModelVirtualMachine{Flavor: "d2-2"},
			},
		}, nil
	})

	h.flavors = []flavors.Flavor{
		{Name: "s1-4", VCPUs: 1},
		{Name: "d2-4", VCPUs: 2},
		{Name: "b3-8", VCPUs: 4},
		{Name: "b3-32", VCPUs: 16},
	}

	// Only model that match a known flavor should be returned and sorted by CPUs asc
	ms, err := h.WorkerModelsEnabled()
	require.NoError(t, err)
	require.Len(t, ms, 4)
	assert.Equal(t, "my-model-4", ms[0].Name)
	assert.Equal(t, "my-model-5", ms[1].Name)
	assert.Equal(t, "my-model-3", ms[2].Name)
	assert.Equal(t, "my-model-2", ms[3].Name)
}

func TestHatcheryOpenstack_checkOverrideImagesUsername(t *testing.T) {
	tests := []struct {
		name      string
		overrides []ImageUsernameOverride
		wantErr   bool
	}{
		{
			name:      "empty",
			overrides: []ImageUsernameOverride{},
		},
		{
			name:      "nil",
			overrides: nil,
		},
		{
			name: "valid-values",
			overrides: []ImageUsernameOverride{
				{
					Image:    "foo",
					Username: "bar",
				},
				{
					Image:    "^foo-[a-z]+",
					Username: "baz123",
				},
				{
					Image:    "^baz$",
					Username: "_foobar",
				},
			},
		},
		{
			name: "invalid-image-regexp",
			overrides: []ImageUsernameOverride{
				{
					Image:    "foo[",
					Username: "bar",
				},
			},
			wantErr: true,
		},
		{
			name: "username-starting-with-dash",
			overrides: []ImageUsernameOverride{
				{
					Image:    "^foo$",
					Username: "-baz",
				},
			},
			wantErr: true,
		},
		{
			name: "username-starting-with-number",
			overrides: []ImageUsernameOverride{
				{
					Image:    "^foo$",
					Username: "1baz",
				},
			},
			wantErr: true,
		},
		{
			name: "username-too-long",
			overrides: []ImageUsernameOverride{
				{
					Image:    "^foo$",
					Username: "abcdefghijklmnopqrstuvwxyz0123456",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HatcheryOpenstack{}
			if err := h.checkOverrideImagesUsername(tt.overrides); (err != nil) != tt.wantErr {
				t.Errorf("HatcheryOpenstack.checkOverrideImagesUsername() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHatcheryOpenstack_checkInjectSSHPublicKeys(t *testing.T) {
	rsa, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	publicRsa, err := ssh.NewPublicKey(&rsa.PublicKey)
	require.NoError(t, err)
	pubRsaBytes := ssh.MarshalAuthorizedKey(publicRsa)
	ed, _, err := ed25519.GenerateKey(nil)
	require.NoError(t, err)
	publicEd, err := ssh.NewPublicKey(ed)
	require.NoError(t, err)
	publicEdBytes := ssh.MarshalAuthorizedKey(publicEd)

	tests := []struct {
		name       string
		publicKeys []string
		wantErr    bool
	}{
		{
			name:       "empty",
			publicKeys: []string{},
			wantErr:    false,
		},
		{
			name:       "nil",
			publicKeys: nil,
			wantErr:    false,
		},
		{
			name: "valid-values",
			publicKeys: []string{
				"from=\"0.1.2.3\" " + string(pubRsaBytes),
				"from=\"0.1.2.3\" " + string(publicEdBytes),
			},
			wantErr: false,
		},
		{
			name: "invalid-key-missing-from-option",
			publicKeys: []string{
				string(pubRsaBytes),
				string(publicEdBytes),
			},
			wantErr: true,
		},
		{
			name: "invalid-key",
			publicKeys: []string{
				"invalid-ssh-key-format",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HatcheryOpenstack{}
			if err := h.checkInjectSSHPublicKeys(tt.publicKeys); (err != nil) != tt.wantErr {
				t.Errorf("HatcheryOpenstack.checkInjectSSHPublicKeys() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
