package cdsclient

import (
	"context"
	"time"

	"io"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
)

// Interface is the main interface for cdsclient package
type Interface interface {
	QueuePolling(context.Context, chan<- sdk.WorkflowNodeJobRun, chan<- sdk.PipelineBuildJob, chan<- error, time.Duration) error
	QueueTakeJob(sdk.WorkflowNodeJobRun, bool) (*worker.WorkflowNodeJobRunInfo, error)
	QueueJobInfo(int64) (*sdk.WorkflowNodeJobRun, error)
	QueueSendResult(int64, sdk.Result) error
	QueueArtifactUpload(id int64, tag, filePath string) error
	Requirements() ([]sdk.Requirement, error)
	WorkerRegister(worker.RegistrationForm) (string, bool, error)
	WorkerSetStatus(sdk.Status) error
	WorkflowRun(projectKey string, name string, number int64) (*sdk.WorkflowRun, error)
	WorkflowRunArtifacts(projectKey string, name string, number int64) ([]sdk.Artifact, error)
	WorkflowNodeRun(projectKey string, name string, number int64, nodeRunID int64) (*sdk.WorkflowNodeRun, error)
	WorkflowNodeRunArtifacts(projectKey string, name string, number int64, nodeRunID int64) ([]sdk.Artifact, error)
	WorkflowNodeRunArtifactDownload(projectKey string, name string, artifactID int64, w io.Writer) error
}
