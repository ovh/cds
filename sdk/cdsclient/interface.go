package cdsclient

import (
	"context"
	"time"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
)

// Interface is the main interface for cdsclient package
type Interface interface {
	QueuePolling(context.Context, chan<- sdk.WorkflowNodeJobRun, chan<- sdk.PipelineBuildJob, chan<- error, time.Duration) error
	QueueTakeJob(sdk.WorkflowNodeJobRun, bool) (*worker.WorkflowNodeJobRunInfo, error)
	Requirements() ([]sdk.Requirement, error)
	WorkerRegister(worker.RegistrationForm) (string, bool, error)
	WorkerSetStatus(sdk.Status) error
}
