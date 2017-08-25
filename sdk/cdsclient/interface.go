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
	APIURL() string
	HatcheryRegister(sdk.Hatchery) (*sdk.Hatchery, error)
	MonStatus() ([]string, error)
	ProjectCreate(*sdk.Project) error
	ProjectDelete(string) error
	ProjectGet(string, ...RequestModifier) (*sdk.Project, error)
	ProjectList() ([]sdk.Project, error)
	ProjectKeysList(string) ([]sdk.ProjectKey, error)
	ProjectKeyCreate(string, *sdk.ProjectKey) error
	ProjectKeysDelete(string, string) error
	QueuePolling(context.Context, chan<- sdk.WorkflowNodeJobRun, chan<- sdk.PipelineBuildJob, chan<- error, time.Duration) error
	QueueTakeJob(sdk.WorkflowNodeJobRun, bool) (*worker.WorkflowNodeJobRunInfo, error)
	QueueJobInfo(int64) (*sdk.WorkflowNodeJobRun, error)
	QueueSendResult(int64, sdk.Result) error
	QueueArtifactUpload(id int64, tag, filePath string) error
	Requirements() ([]sdk.Requirement, error)
	UserLogin(username, password string) (bool, string, error)
	UserList() ([]sdk.User, error)
	UserSignup(username, fullname, email, callback string) error
	UserGet(username string) (*sdk.User, error)
	UserGetGroups(username string) (map[string][]sdk.Group, error)
	UserReset(username, email string) error
	UserConfirm(username, token string) (bool, string, error)
	WorkerList() ([]sdk.Worker, error)
	WorkerModelsEnabled() ([]sdk.Model, error)
	WorkerModels() ([]sdk.Model, error)
	WorkerRegister(worker.RegistrationForm) (string, bool, error)
	WorkerSetStatus(sdk.Status) error
	WorkflowList(projectKey string) ([]sdk.Workflow, error)
	WorkflowGet(projectKey, name string) (*sdk.Workflow, error)
	WorkflowRun(projectKey string, name string, number int64) (*sdk.WorkflowRun, error)
	WorkflowRunArtifacts(projectKey string, name string, number int64) ([]sdk.Artifact, error)
	WorkflowNodeRun(projectKey string, name string, number int64, nodeRunID int64) (*sdk.WorkflowNodeRun, error)
	WorkflowNodeRunArtifacts(projectKey string, name string, number int64, nodeRunID int64) ([]sdk.Artifact, error)
	WorkflowNodeRunArtifactDownload(projectKey string, name string, artifactID int64, w io.Writer) error
}
