package vsphere

import (
	"testing"

	"github.com/ovh/cds/engine/hatchery/vsphere/mock_vsphere"
	"go.uber.org/mock/gomock"
)

func NewVSphereClientTest(t *testing.T) *mock_vsphere.MockVSphereClient {
	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })
	mockClient := mock_vsphere.NewMockVSphereClient(ctrl)
	return mockClient
}
