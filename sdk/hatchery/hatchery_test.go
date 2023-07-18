package hatchery_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/rockbears/log"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/ovh/cds/sdk/hatchery/mock_hatchery"
	"github.com/ovh/cds/sdk/jws"
)

func TestCreate(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	ctx := context.TODO()
	ctx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()
	ctrl1 := gomock.NewController(t)
	ctrl2 := gomock.NewController(t)

	t.Cleanup(func() {
		ctrl1.Finish()
		ctrl2.Finish()
	})

	mockHatchery := mock_hatchery.NewMockInterface(ctrl1)
	mockCDSClient := mock_cdsclient.NewMockInterface(ctrl2)

	grtn := sdk.NewGoRoutines(ctx)
	hatcheryConfig := service.HatcheryCommonConfiguration{
		Name: t.Name(),
	}
	hatcheryConfig.Provision.MaxWorker = 1

	hatchery.CacheSpawnIDsTTL = 2 * time.Second                 // decrease this cache TTL to speedup the test
	hatcheryConfig.Provision.MaxAttemptsNumberBeforeFailure = 2 // decrease this value to speedup the test

	mockHatchery.EXPECT().Name().Return(t.Name()).AnyTimes()
	mockHatchery.EXPECT().Type().Return(sdk.TypeHatchery).AnyTimes()
	mockHatchery.EXPECT().Service().Return(&sdk.Service{}).AnyTimes()
	mockHatchery.EXPECT().InitHatchery(gomock.Any()).Return(nil)
	mockHatchery.EXPECT().Configuration().Return(hatcheryConfig).AnyTimes()
	mockHatchery.EXPECT().GetGoRoutines().Return(grtn).AnyTimes()
	mockHatchery.EXPECT().CDSClient().Return(mockCDSClient).AnyTimes()
	mockHatchery.EXPECT().CDSClientV2().Return(nil).AnyTimes()
	mockCDSClient.EXPECT().QueuePolling(gomock.Any(), grtn, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, goRoutines *sdk.GoRoutines, jobs chan<- sdk.WorkflowNodeJobRun, errs chan<- error, delay time.Duration, ms ...cdsclient.RequestModifier) error {
			j := sdk.WorkflowNodeJobRun{
				ProjectID:         1,
				ID:                666,
				WorkflowNodeRunID: 1,
				Status:            sdk.StatusWaiting,
				Job: sdk.ExecutedJob{
					Job: sdk.Job{},
				},
				Start: time.Now(),
			}

			jobs <- j                   // Send the job a first time, it will trigger the first call on SpawnWorker
			time.Sleep(1 * time.Second) // Wait
			jobs <- j                   // This one must be ignored with a log "already spawned in previous routine"
			time.Sleep(2 * time.Second) // Wait
			jobs <- j                   // This will trigger a second call on SpawnWorker
			time.Sleep(3 * time.Second) // Wait
			jobs <- j                   // This shoud not trigger the call on SpawnWorker but should fail the job

			<-ctx.Done()
			return ctx.Err()
		},
	)

	// This calls are expected for each job received in the channel
	mockCDSClient.EXPECT().WorkerList(gomock.Any()).Return(nil, nil).AnyTimes()
	mockHatchery.EXPECT().WorkersStarted(gomock.Any()).Return(nil, nil).AnyTimes()
	mockHatchery.EXPECT().CanSpawn(gomock.Any(), gomock.Any(), "666", gomock.Any()).Return(true).AnyTimes()
	mockCDSClient.EXPECT().QueueJobBook(gomock.Any(), "666").Return(sdk.WorkflowNodeJobRunBooked{}, nil).AnyTimes()
	mockCDSClient.EXPECT().QueueJobSendSpawnInfo(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	privateKey, err := jws.NewRandomRSAKey()
	require.NoError(t, err)
	mockHatchery.EXPECT().GetPrivateKey().Return(privateKey).AnyTimes()

	// Call to SpawnWorker regarding what append in "QueuePolling"
	mockHatchery.EXPECT().SpawnWorker(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	// Expecing a call to QueueSendResult
	mockCDSClient.EXPECT().QueueSendResult(gomock.Any(), int64(666), gomock.Any()).Return(nil)

	hatchery.Create(ctx, mockHatchery)

	<-ctx.Done()

}

func getMockLogger() *logrus.Logger {
	log := logrus.New()
	log.AddHook(&HookMock{})
	return log
}

type HookMock struct{}

func (h *HookMock) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.InfoLevel,
	}
}
func (h *HookMock) Fire(e *logrus.Entry) error {
	return nil
}
