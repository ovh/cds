package cdsclient

import (
	"archive/tar"
	"context"
	"io"
	"time"

	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
)

// ExportImportInterface exposes pipeline and application export and import function
type ExportImportInterface interface {
	PipelineExport(projectKey, name string, exportWithPermissions bool, exportFormat string) ([]byte, error)
	PipelineImport(projectKey string, content io.Reader, format string, force bool) ([]string, error)
	ApplicationExport(projectKey, name string, exportWithPermissions bool, format string) ([]byte, error)
	ApplicationImport(projectKey string, content io.Reader, format string, force bool) ([]string, error)
	WorkflowExport(projectKey, name string, exportWithPermissions bool, exportFormat string) ([]byte, error)
	WorkflowPull(projectKey, name string, exportWithPermissions bool) (*tar.Reader, error)
	WorkflowImport(projectKey string, content io.Reader, format string, force bool) ([]string, error)
	WorkflowPush(projectKey string, tarContent io.Reader) ([]string, error)
}

// ApplicationClient exposes application related functions
type ApplicationClient interface {
	ApplicationAttachToReposistoriesManager(projectKey, appName, reposManager, repoFullname string) error
	ApplicationCreate(projectKey string, app *sdk.Application) error
	ApplicationDelete(projectKey string, appName string) error
	ApplicationGet(projectKey string, appName string, opts ...RequestModifier) (*sdk.Application, error)
	ApplicationList(projectKey string) ([]sdk.Application, error)
	ApplicationGroupsImport(projectKey, appName string, content io.Reader, format string, force bool) (sdk.Application, error)
	ApplicationVariableClient
	ApplicationKeysClient
}

// ApplicationKeysClient exposes application keys related functions
type ApplicationKeysClient interface {
	ApplicationKeysList(projectKey string, appName string) ([]sdk.ApplicationKey, error)
	ApplicationKeyCreate(projectKey string, appName string, keyApp *sdk.ApplicationKey) error
	ApplicationKeysDelete(projectKey string, appName string, KeyAppName string) error
}

// ApplicationVariableClient exposes application variables related functions
type ApplicationVariableClient interface {
	ApplicationVariablesList(projectKey string, appName string) ([]sdk.Variable, error)
	ApplicationVariableCreate(projectKey string, appName string, variable *sdk.Variable) error
	ApplicationVariableDelete(projectKey string, appName string, varName string) error
	ApplicationVariableGet(projectKey string, appName string, varName string) (*sdk.Variable, error)
	ApplicationVariableUpdate(projectKey string, appName string, variable *sdk.Variable) error
}

// EnvironmentClient exposes environment related functions
type EnvironmentClient interface {
	EnvironmentCreate(projectKey string, env *sdk.Environment) error
	EnvironmentDelete(projectKey string, envName string) error
	EnvironmentGet(projectKey string, envName string, opts ...RequestModifier) (*sdk.Environment, error)
	EnvironmentList(projectKey string) ([]sdk.Environment, error)
	EnvironmentGroupsImport(projectKey, envName string, content io.Reader, format string, force bool) (sdk.Environment, error)
	EnvironmentVariableClient
	EnvironmentKeysClient
}

// EnvironmentKeysClient exposes environment keys related functions
type EnvironmentKeysClient interface {
	EnvironmentKeysList(projectKey string, envName string) ([]sdk.EnvironmentKey, error)
	EnvironmentKeyCreate(projectKey string, envName string, keyEnv *sdk.EnvironmentKey) error
	EnvironmentKeysDelete(projectKey string, envName string, keyEnvName string) error
}

// EnvironmentVariableClient exposes environment variables related functions
type EnvironmentVariableClient interface {
	EnvironmentVariablesList(key string, envName string) ([]sdk.Variable, error)
	EnvironmentVariableCreate(projectKey string, envName string, variable *sdk.Variable) error
	EnvironmentVariableDelete(projectKey string, envName string, varName string) error
	EnvironmentVariableGet(projectKey string, envName string, varName string) (*sdk.Variable, error)
	EnvironmentVariableUpdate(projectKey string, envName string, variable *sdk.Variable) error
}

// DownloadClient exposes download related functions
type DownloadClient interface {
	Download() ([]sdk.Download, error)
	DownloadURLFromAPI(name, os, arch string) string
	DownloadURLFromGithub(name, os, arch string) (string, error)
}

// ActionClient exposes actions related functions
type ActionClient interface {
	ActionDelete(actionName string) error
	ActionGet(actionName string, mods ...RequestModifier) (*sdk.Action, error)
	ActionList() ([]sdk.Action, error)
}

// GroupClient exposes groups related functions
type GroupClient interface {
	GroupCreate(group *sdk.Group) error
	GroupDelete(name string) error
	GroupGenerateToken(groupName, expiration string) (*sdk.Token, error)
	GroupGet(name string, mods ...RequestModifier) (*sdk.Group, error)
	GroupList() ([]sdk.Group, error)
	GroupUserAdminSet(groupname string, username string) error
	GroupUserAdminRemove(groupname, username string) error
	GroupUserAdd(groupname string, users []string) error
	GroupUserRemove(groupname, username string) error
}

// HatcheryClient exposes hatcheries related functions
type HatcheryClient interface {
	HatcheryRefresh(int64) error
	HatcheryRegister(sdk.Hatchery) (*sdk.Hatchery, bool, error)
}

// PipelineClient exposes pipelines related functions
type PipelineClient interface {
	PipelineDelete(projectKey, name string) error
	PipelineCreate(projectKey string, pip *sdk.Pipeline) error
	PipelineGroupsImport(projectKey, pipelineName string, content io.Reader, format string, force bool) (sdk.Pipeline, error)
	PipelineList(projectKey string) ([]sdk.Pipeline, error)
}

// ProjectClient exposes project related functions
type ProjectClient interface {
	ProjectCreate(proj *sdk.Project, groupName string) error
	ProjectDelete(projectKey string) error
	ProjectGet(projectKey string, opts ...RequestModifier) (*sdk.Project, error)
	ProjectList() ([]sdk.Project, error)
	ProjectKeysClient
	ProjectVariablesClient
	ProjectGroupsImport(projectKey string, content io.Reader, format string, force bool) (sdk.Project, error)
}

// ProjectKeysClient exposes project keys related functions
type ProjectKeysClient interface {
	ProjectKeysList(projectKey string) ([]sdk.ProjectKey, error)
	ProjectKeyCreate(projectKey string, key *sdk.ProjectKey) error
	ProjectKeysDelete(projectKey string, keyProjectName string) error
}

// ProjectVariablesClient exposes project variables related functions
type ProjectVariablesClient interface {
	ProjectVariablesList(key string) ([]sdk.Variable, error)
	ProjectVariableCreate(projectKey string, variable *sdk.Variable) error
	ProjectVariableDelete(projectKey string, varName string) error
	ProjectVariableGet(projectKey string, varName string) (*sdk.Variable, error)
	ProjectVariableUpdate(projectKey string, variable *sdk.Variable) error
	VariableEnrypt(projectKey string, varName string, content string) (*sdk.Variable, error)
}

// QueueClient exposes queue related functions
type QueueClient interface {
	QueueWorkflowNodeJobRun() ([]sdk.WorkflowNodeJobRun, error)
	QueueCountWorkflowNodeJobRun() (sdk.WorkflowNodeJobRunCount, error)
	QueuePipelineBuildJob() ([]sdk.PipelineBuildJob, error)
	QueuePolling(context.Context, chan<- sdk.WorkflowNodeJobRun, chan<- sdk.PipelineBuildJob, chan<- error, time.Duration, int) error
	QueueTakeJob(sdk.WorkflowNodeJobRun, bool) (*worker.WorkflowNodeJobRunInfo, error)
	QueueJobBook(isWorkflowJob bool, id int64) error
	QueueJobInfo(id int64) (*sdk.WorkflowNodeJobRun, error)
	QueueJobSendSpawnInfo(isWorkflowJob bool, id int64, in []sdk.SpawnInfo) error
	QueueSendResult(int64, sdk.Result) error
	QueueArtifactUpload(id int64, tag, filePath string) (bool, time.Duration, error)
	QueueJobTag(jobID int64, tags []sdk.WorkflowRunTag) error
}

// TemplateClient exposes queue related functions
type TemplateClient interface {
	TemplateApplicationCreate(projectKey, name string, template *sdk.Template) error
	TemplateList() ([]sdk.Template, error)
	TemplateGet(name string) (*sdk.Template, error)
}

// UserClient exposes users functions
type UserClient interface {
	UserConfirm(username, token string) (bool, string, error)
	UserList() ([]sdk.User, error)
	UserGet(username string) (*sdk.User, error)
	UserGetGroups(username string) (map[string][]sdk.Group, error)
	UserLogin(username, password string) (bool, string, error)
	UserReset(username, email, callback string) error
	UserSignup(username, fullname, email, callback string) error
}

// WorkerClient exposes workers functions
type WorkerClient interface {
	WorkerModelBook(id int64) error
	WorkerList() ([]sdk.Worker, error)
	WorkerModelSpawnError(id int64, info string) error
	WorkerModelsEnabled() ([]sdk.Model, error)
	WorkerModels() ([]sdk.Model, error)
	WorkerRegister(sdk.WorkerRegistrationForm) (*sdk.Worker, bool, error)
	WorkerSetStatus(sdk.Status) error
}

// WorkflowClient exposes workflows functions
type WorkflowClient interface {
	WorkflowList(projectKey string) ([]sdk.Workflow, error)
	WorkflowGet(projectKey, name string) (*sdk.Workflow, error)
	WorkflowDelete(projectKey string, workflowName string) error
	WorkflowRunGet(projectKey string, workflowName string, number int64) (*sdk.WorkflowRun, error)
	WorkflowRunList(projectKey string, workflowName string, offset, limit int64) ([]sdk.WorkflowRun, error)
	WorkflowRunArtifacts(projectKey string, name string, number int64) ([]sdk.WorkflowNodeRunArtifact, error)
	WorkflowRunFromHook(projectKey string, workflowName string, hook sdk.WorkflowNodeRunHookEvent) (*sdk.WorkflowRun, error)
	WorkflowRunFromManual(projectKey string, workflowName string, manual sdk.WorkflowNodeRunManual, number, fromNodeID int64) (*sdk.WorkflowRun, error)
	WorkflowNodeRun(projectKey string, name string, number int64, nodeRunID int64) (*sdk.WorkflowNodeRun, error)
	WorkflowNodeRunArtifacts(projectKey string, name string, number int64, nodeRunID int64) ([]sdk.WorkflowNodeRunArtifact, error)
	WorkflowNodeRunArtifactDownload(projectKey string, name string, a sdk.WorkflowNodeRunArtifact, w io.Writer) error
	WorkflowNodeRunJobStep(projectKey string, workflowName string, number int64, nodeRunID, job int64, step int) (*sdk.BuildState, error)
	WorkflowNodeRunRelease(projectKey string, workflowName string, runNumber int64, nodeRunID int64, release sdk.WorkflowNodeRunRelease) error
	WorkflowAllHooksList() ([]sdk.WorkflowNodeHook, error)
}

// MonitoringClient exposes monitoring functions
type MonitoringClient interface {
	MonStatus() (*sdk.MonitoringStatus, error)
	MonDBTimes() (*sdk.MonDBTimes, error)
	MonDBMigrate() ([]sdk.MonDBMigrate, error)
}

// Interface is the main interface for cdsclient package
type Interface interface {
	ActionClient
	APIURL() string
	ApplicationClient
	ConfigUser() (map[string]string, error)
	DownloadClient
	EnvironmentClient
	ExportImportInterface
	GroupClient
	HatcheryClient
	PipelineClient
	ProjectClient
	QueueClient
	Requirements() ([]sdk.Requirement, error)
	ServiceRegister(sdk.Service) (string, error)
	TemplateClient
	UserClient
	WorkerClient
	WorkflowClient
	MonitoringClient
	Version() (*sdk.Version, error)
}

// InterfaceDeprecated is the interface for using deprecated routes with cdsclient package
type InterfaceDeprecated interface {
	ApplicationPipelinesAttach(projectKey string, appName string, pipelineNames ...string) error
	ApplicationPipelineTriggerAdd(t *sdk.PipelineTrigger) error
	ApplicationPipelineTriggersGet(projectKey string, appName string, pipelineName string, envName string) ([]sdk.PipelineTrigger, error)
	AddHookOnRepositoriesManager(projectKey, appName, reposManager, repoFullname, pipelineName string) error
}

// Raw is a low-level interface exposing HTTP functions
type Raw interface {
	PostJSON(path string, in interface{}, out interface{}, mods ...RequestModifier) (int, error)
	PutJSON(path string, in interface{}, out interface{}, mods ...RequestModifier) (int, error)
	GetJSON(path string, out interface{}, mods ...RequestModifier) (int, error)
	DeleteJSON(path string, out interface{}, mods ...RequestModifier) (int, error)
	Request(method string, path string, body io.Reader, mods ...RequestModifier) ([]byte, int, error)
}
