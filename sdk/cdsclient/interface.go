package cdsclient

import (
	"context"
	"io"
	"time"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
)

// Interface is the main interface for cdsclient package
type Interface interface {
	APIURL() string
	ApplicationCreate(string, *sdk.Application) error
	ApplicationDelete(string, string) error
	ApplicationGet(string, string, ...RequestModifier) (*sdk.Application, error)
	ApplicationList(string) ([]sdk.Application, error)
	ApplicationKeysList(string, string) ([]sdk.ApplicationKey, error)
	ApplicationKeyCreate(string, string, *sdk.ApplicationKey) error
	ApplicationKeysDelete(string, string, string) error
	ConfigUser() (map[string]string, error)
	EnvironmentCreate(string, *sdk.Environment) error
	EnvironmentDelete(string, string) error
	EnvironmentGet(string, string, ...RequestModifier) (*sdk.Environment, error)
	EnvironmentList(string) ([]sdk.Environment, error)
	EnvironmentKeysList(string, string) ([]sdk.EnvironmentKey, error)
	EnvironmentKeyCreate(string, string, *sdk.EnvironmentKey) error
	EnvironmentKeysDelete(string, string, string) error
	HatcheryRefresh(int64) error
	HatcheryRegister(sdk.Hatchery) (*sdk.Hatchery, bool, error)
	MonStatus() ([]string, error)
	ProjectCreate(*sdk.Project) error
	ProjectDelete(string) error
	ProjectGet(string, ...RequestModifier) (*sdk.Project, error)
	ProjectList() ([]sdk.Project, error)
	ProjectKeysList(string) ([]sdk.ProjectKey, error)
	ProjectKeyCreate(string, *sdk.ProjectKey) error
	ProjectKeysDelete(string, string) error
	Queue() ([]sdk.WorkflowNodeJobRun, []sdk.PipelineBuildJob, error)
	QueuePolling(context.Context, chan<- sdk.WorkflowNodeJobRun, chan<- sdk.PipelineBuildJob, chan<- error, time.Duration) error
	QueueTakeJob(sdk.WorkflowNodeJobRun, bool) (*worker.WorkflowNodeJobRunInfo, error)
	QueueJobBook(isWorkflowJob bool, id int64) error
	QueueJobInfo(id int64) (*sdk.WorkflowNodeJobRun, error)
	QueueJobSendSpawnInfo(isWorkflowJob bool, id int64, in []sdk.SpawnInfo) error
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
	WorkerModelSpawnError(id int64, info string) error
	WorkerModelsEnabled() ([]sdk.Model, error)
	WorkerModels() ([]sdk.Model, error)
	WorkerRegister(worker.RegistrationForm) (*sdk.Worker, bool, error)
	WorkerSetStatus(sdk.Status) error
	WorkflowList(projectKey string) ([]sdk.Workflow, error)
	WorkflowGet(projectKey, name string) (*sdk.Workflow, error)
	WorkflowRun(projectKey string, name string, number int64) (*sdk.WorkflowRun, error)
	WorkflowRunArtifacts(projectKey string, name string, number int64) ([]sdk.Artifact, error)
	WorkflowNodeRun(projectKey string, name string, number int64, nodeRunID int64) (*sdk.WorkflowNodeRun, error)
	WorkflowNodeRunArtifacts(projectKey string, name string, number int64, nodeRunID int64) ([]sdk.Artifact, error)
	WorkflowNodeRunArtifactDownload(projectKey string, name string, artifactID int64, w io.Writer) error
}
