package hatchery_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/hatchery/mock_hatchery"
)

func newWorkerPoolMocks(t *testing.T) (*mock_hatchery.MockInterface, *mock_cdsclient.MockInterface) {
	log.Factory = log.NewTestingWrapper(t)
	require.NoError(t, hatchery.InitMetrics(context.TODO()))
	ctrl := gomock.NewController(t)
	mockHatchery := mock_hatchery.NewMockInterface(ctrl)
	mockCDSClient := mock_cdsclient.NewMockInterface(ctrl)

	mockHatchery.EXPECT().Name().Return(t.Name()).AnyTimes()
	mockHatchery.EXPECT().Type().Return(sdk.TypeHatchery).AnyTimes()
	mockHatchery.EXPECT().CDSClient().Return(mockCDSClient).AnyTimes()
	mockHatchery.EXPECT().CDSClientV2().Return(nil).AnyTimes()
	mockCDSClient.EXPECT().WorkerList(gomock.Any()).Return([]sdk.Worker{
		{ID: "1", Name: "w1", Status: sdk.StatusBuilding},
		{ID: "2", Name: "w2", Status: sdk.StatusBuilding},
	}, nil)
	return mockHatchery, mockCDSClient
}

func TestWorkerPoolDisablesRegisteredWorkerMissingFromOrchestrator(t *testing.T) {
	mockHatchery, mockCDSClient := newWorkerPoolMocks(t)
	mockHatchery.EXPECT().WorkersStarted(gomock.Any()).Return([]string{"w1"}, nil)
	mockCDSClient.EXPECT().WorkerDisable(gomock.Any(), "2").Return(nil)

	pool, err := hatchery.WorkerPool(context.TODO(), mockHatchery)
	require.NoError(t, err)
	require.Len(t, pool, 2)
	require.Equal(t, sdk.StatusDisabled, pool[1].GetStatus())
}

func TestWorkerPoolWithIncompleteWorkersStarted(t *testing.T) {
	mockHatchery, _ := newWorkerPoolMocks(t)
	mockHatchery.EXPECT().WorkersStarted(gomock.Any()).Return([]string{"w1"}, fmt.Errorf("%w: engine unreachable", hatchery.ErrIncompleteWorkersStarted))
	// No call to WorkerDisable: w2 may run on the part of the orchestrator that could not be listed

	pool, err := hatchery.WorkerPool(context.TODO(), mockHatchery)
	require.NoError(t, err)
	require.Len(t, pool, 2)
	require.Equal(t, sdk.StatusBuilding, pool[1].GetStatus())
}

func TestWorkerPoolWithWorkersStartedError(t *testing.T) {
	mockHatchery, _ := newWorkerPoolMocks(t)
	mockHatchery.EXPECT().WorkersStarted(gomock.Any()).Return(nil, errors.New("orchestrator unreachable"))

	_, err := hatchery.WorkerPool(context.TODO(), mockHatchery)
	require.Error(t, err)
}
