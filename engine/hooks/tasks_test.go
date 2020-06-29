package hooks

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/ovh/cds/sdk/log"
)

func init() {
	log.Initialize(context.TODO(), &log.Conf{Level: "debug"})
}

func Test_doWebHookExecution(t *testing.T) {
	log.SetLogger(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	task := &sdk.TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeWebHook,
		WebHook: &sdk.WebHookExecution{
			RequestBody: nil,
			RequestURL:  "uid=42413e87905b813a375c7043ce9d4047b7e265ae3730b60180cad02ae81cc62385e5b05b9e7c758b15bb3872498a5e88963f3deac308f636baf345ed9cf1b259&project=IRTM&name=rtm-packaging&branch=master&hash=123456789&message=monmessage&author=sguiheux",
		},
	}
	hs, err := s.doWebHookExecution(context.TODO(), task)
	test.NoError(t, err)

	assert.Equal(t, 1, len(hs))
	assert.Equal(t, "master", hs[0].Payload["branch"])
	assert.Equal(t, "sguiheux", hs[0].Payload["author"])
	assert.Equal(t, "monmessage", hs[0].Payload["message"])
	assert.Equal(t, "123456789", hs[0].Payload["hash"])
	assert.True(t, hs[0].Payload["payload"] != "", "payload should not be empty")
}

func Test_doWebHookExecutionWithRequestBody(t *testing.T) {
	log.SetLogger(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	task := &sdk.TaskExecution{
		UUID: sdk.RandomString(10),
		Type: TypeWebHook,
		WebHook: &sdk.WebHookExecution{
			RequestMethod: string(http.MethodPost),
			RequestHeader: map[string][]string{
				"Content-Type": []string{
					"application/json",
				},
			},
			RequestBody: []byte(`{"test": "hereisatest"}`),
		},
		Config: sdk.WorkflowNodeHookConfig{
			"method": sdk.WorkflowNodeHookConfigValue{
				Value: string(http.MethodPost),
			},
		},
	}
	hs, err := s.doWebHookExecution(context.TODO(), task)
	test.NoError(t, err)

	assert.Equal(t, 1, len(hs))
	assert.Equal(t, "hereisatest", hs[0].Payload["test"])
	assert.True(t, hs[0].Payload["payload"] != "", "payload should not be empty")
}

func Test_dequeueTaskExecutions_ScheduledTask(t *testing.T) {
	log.SetLogger(t)
	s, cancel := setupTestHookService(t)
	defer cancel()

	ctx, cancel := context.WithTimeout(context.TODO(), 60*time.Second)
	defer cancel()

	// Get the mock
	m := s.Client.(*mock_cdsclient.MockInterface)

	// Mock the sync of tasks
	// It will remove all the tasks from the database
	m.EXPECT().WorkflowAllHooksList().Return([]sdk.NodeHook{}, nil)
	m.EXPECT().VCSConfiguration().Return(nil, nil).AnyTimes()
	require.NoError(t, s.synchronizeTasks(ctx))

	// Start the goroutine
	go func() {
		if err := s.dequeueTaskExecutions(ctx); err != nil {
			t.Logf("dequeueTaskExecutions error: %v", err)
		}
	}()

	h := &sdk.NodeHook{
		UUID:          sdk.UUID(),
		HookModelName: TypeScheduler,
		Config: sdk.WorkflowNodeHookConfig{
			sdk.HookConfigProject:  sdk.WorkflowNodeHookConfigValue{Value: "FOO"},
			sdk.HookConfigWorkflow: sdk.WorkflowNodeHookConfigValue{Value: "BAR"},
			sdk.SchedulerModelCron: sdk.WorkflowNodeHookConfigValue{
				Value:        "* * * * *",
				Configurable: true,
			},
			sdk.SchedulerModelTimezone: sdk.WorkflowNodeHookConfigValue{
				Value:        "UTC",
				Configurable: true,
			},
			sdk.Payload: sdk.WorkflowNodeHookConfigValue{
				Value:        "{}",
				Configurable: true,
			},
		},
	}

	// Create a new task
	scheduledTask, err := s.hookToTask(h)
	require.NoError(t, s.Dao.SaveTask(scheduledTask))
	require.NoError(t, s.startTasks(ctx))

	// Check that the task has been correctly saved
	scheduledTask = s.Dao.FindTask(ctx, scheduledTask.UUID)
	assert.False(t, scheduledTask.Stopped)
	assert.Equal(t, 0, scheduledTask.NbExecutionsTotal)
	assert.Equal(t, 0, scheduledTask.NbExecutionsTodo)

	// Setup the expected calls that will be triggered by
	// enqueueScheduledTaskExecutionsRoutine
	m.EXPECT().
		WorkflowRunFromHook(
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
		).
		Return(
			&sdk.WorkflowRun{
				Number: 666,
			},
			nil,
		).
		MinTimes(1)

	// start enqueueScheduledTaskExecutionsRoutine to scheduled the execution
	go func() {
		if err := s.enqueueScheduledTaskExecutionsRoutine(ctx); err != nil {
			t.Logf("enqueueScheduledTaskExecutionsRoutine error: %v", err)
		}
	}()

	// Wait until it's over
	<-ctx.Done()

	// Load the executions to check if the first has been firec and a second one is pending
	execs, err := s.Dao.FindAllTaskExecutions(context.Background(), scheduledTask)
	require.NoError(t, err)
	require.Len(t, execs, 2)
	assert.Equal(t, "DONE", execs[0].Status)
	assert.Equal(t, "SCHEDULED", execs[1].Status)

	// Now we will triggered another hooks sync
	// The mock must return one hook
	m.EXPECT().WorkflowAllHooksList().Return([]sdk.NodeHook{*h}, nil)
	require.NoError(t, s.synchronizeTasks(context.Background()))

	// We must be able to find the task
	scheduledTask2 := s.Dao.FindTask(context.Background(), scheduledTask.UUID)
	assert.Equal(t, scheduledTask, scheduledTask2)

	// Load the executions to check if the first has been firec and a second one is still pending
	execs, err = s.Dao.FindAllTaskExecutions(context.Background(), scheduledTask2)
	require.NoError(t, err)
	require.Len(t, execs, 2)
	assert.Equal(t, "DONE", execs[0].Status)
	assert.Equal(t, "SCHEDULED", execs[1].Status)

	time.Sleep(5 * time.Second)
}
