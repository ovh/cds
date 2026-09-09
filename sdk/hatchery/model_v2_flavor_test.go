package hatchery_test

import (
	"context"
	"testing"

	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/hatchery/mock_hatchery"
)

const testModelV2Path = "MYPROJ/myvcs/my/repo/my-model@master"

func modelV2Mocks(t *testing.T, ctrl *gomock.Controller, modelType string, spec interface{}) *mock_hatchery.MockInterfaceWithModels {
	specRaw, err := yaml.Marshal(spec)
	require.NoError(t, err)

	mockClient := mock_cdsclient.NewMockHatcheryServiceClient(ctrl)
	mockClient.EXPECT().
		GetWorkerModel(gomock.Any(), "MYPROJ", "myvcs", "my/repo", "my-model", gomock.Any()).
		Return(&sdk.V2WorkerModel{Name: "my-model", Type: modelType, Spec: specRaw}, nil)

	mockHatchery := mock_hatchery.NewMockInterfaceWithModels(ctrl)
	mockHatchery.EXPECT().CDSClientV2().Return(mockClient).AnyTimes()
	mockHatchery.EXPECT().ModelType().Return(modelType).AnyTimes()
	mockHatchery.EXPECT().CanSpawn(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true).AnyTimes()

	return mockHatchery
}

// A workflow v1 job requires a worker model v2 by its path. For an openstack model the
// flavor comes from a flavor requirement on the job, and from nowhere else: the flavor of
// the model spec is not read on this path.
func TestCanRunJobWithModelV2OpenstackFlavor(t *testing.T) {
	tests := []struct {
		name           string
		spec           sdk.V2WorkerModelOpenstackSpec
		requirements   []sdk.Requirement
		expectedFlavor string
	}{
		{
			name: "the flavor requirement sizes the virtual machine",
			spec: sdk.V2WorkerModelOpenstackSpec{Image: "my-openstack-image"},
			requirements: []sdk.Requirement{
				{Type: sdk.FlavorRequirement, Value: "large"},
			},
			expectedFlavor: "large",
		},
		{
			name: "the flavor requirement wins over the flavor of the spec",
			spec: sdk.V2WorkerModelOpenstackSpec{Image: "my-openstack-image", Flavor: "small"},
			requirements: []sdk.Requirement{
				{Type: sdk.FlavorRequirement, Value: "large"},
			},
			expectedFlavor: "large",
		},
		{
			name:           "without the requirement no flavor is set, the spec one is not read",
			spec:           sdk.V2WorkerModelOpenstackSpec{Image: "my-openstack-image", Flavor: "small"},
			expectedFlavor: "",
		},
		{
			name: "a requirement of another type sets no flavor",
			spec: sdk.V2WorkerModelOpenstackSpec{Image: "my-openstack-image"},
			requirements: []sdk.Requirement{
				{Type: sdk.RegionRequirement, Value: "eu"},
			},
			expectedFlavor: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			h := modelV2Mocks(t, ctrl, sdk.WorkerModelTypeOpenstack, tt.spec)

			model, _, err := hatchery.CanRunJobWithModelV2ForTest(context.TODO(), h, tt.requirements, testModelV2Path)
			require.NoError(t, err)
			require.NotNil(t, model)
			require.Equal(t, tt.expectedFlavor, model.ModelVirtualMachine.Flavor)
			require.Equal(t, tt.spec.Image, model.ModelVirtualMachine.Image)
		})
	}
}

// A vsphere model is passed through as its v2 spec rather than converted to a model v1,
// so its own flavor sizes the machine and a flavor requirement is not read.
func TestCanRunJobWithModelV2VSphereFlavorComesFromTheSpec(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	spec := sdk.V2WorkerModelVSphereSpec{Image: "my-vsphere-template", Flavor: "small"}
	h := modelV2Mocks(t, ctrl, sdk.WorkerModelTypeVSphere, spec)

	requirements := []sdk.Requirement{{Type: sdk.FlavorRequirement, Value: "large"}}

	model, starterModel, err := hatchery.CanRunJobWithModelV2ForTest(context.TODO(), h, requirements, testModelV2Path)
	require.NoError(t, err)
	require.Nil(t, model, "a vsphere model is not converted to a model v1")
	require.NotNil(t, starterModel)
	require.Equal(t, "small", starterModel.VSphereSpec.Flavor, "the flavor requirement does not override the spec")
}
