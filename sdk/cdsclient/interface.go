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
	ActionDelete(actionName string) error
	ActionGet(actionName string, mods ...RequestModifier) (*sdk.Action, error)
	ActionList() ([]sdk.Action, error)
	APIURL() string
	ApplicationAttachToReposistoriesManager(projectKey, appName, reposManager, repoFullname string) error
	ApplicationCreate(projectKey string, app *sdk.Application) error
	ApplicationDelete(projectKey string, appName string) error
	ApplicationGet(projectKey string, appName string, opts ...RequestModifier) (*sdk.Application, error)
	ApplicationList(projectKey string) ([]sdk.Application, error)
	ApplicationGroupsImport(projectKey, appName string, content []byte, format string, force bool) (sdk.Application, error)
	ApplicationKeysList(projectKey string, appName string) ([]sdk.ApplicationKey, error)
	ApplicationKeyCreate(projectKey string, appName string, keyApp *sdk.ApplicationKey) error
	ApplicationKeysDelete(projectKey string, appName string, KeyAppName string) error
	ApplicationVariablesList(projectKey string, appName string) ([]sdk.Variable, error)
	ApplicationVariableCreate(projectKey string, appName string, variable *sdk.Variable) error
	ApplicationVariableDelete(projectKey string, appName string, varName string) error
	ApplicationVariableGet(projectKey string, appName string, varName string) (*sdk.Variable, error)
	ApplicationVariableUpdate(projectKey string, appName string, variable *sdk.Variable) error
	ConfigUser() (map[string]string, error)
	EnvironmentCreate(projectKey string, env *sdk.Environment) error
	EnvironmentDelete(projectKey string, envName string) error
	EnvironmentGet(projectKey string, envName string, opts ...RequestModifier) (*sdk.Environment, error)
	EnvironmentList(projectKey string) ([]sdk.Environment, error)
	EnvironmentKeysList(projectKey string, envName string) ([]sdk.EnvironmentKey, error)
	EnvironmentKeyCreate(projectKey string, envName string, keyEnv *sdk.EnvironmentKey) error
	EnvironmentKeysDelete(projectKey string, envName string, keyEnvName string) error
	EnvironmentVariablesList(key string, envName string) ([]sdk.Variable, error)
	EnvironmentVariableCreate(projectKey string, envName string, variable *sdk.Variable) error
	EnvironmentVariableDelete(projectKey string, envName string, varName string) error
	EnvironmentVariableGet(projectKey string, envName string, varName string) (*sdk.Variable, error)
	EnvironmentVariableUpdate(projectKey string, envName string, variable *sdk.Variable) error
	EnvironmentGroupsImport(projectKey, envName string, content []byte, format string, force bool) (sdk.Environment, error)
	GroupCreate(group *sdk.Group) error
	GroupDelete(name string) error
	GroupGenerateToken(groupName, expiration string) (*sdk.Token, error)
	GroupGet(name string, mods ...RequestModifier) (*sdk.Group, error)
	GroupList() ([]sdk.Group, error)
	GroupUserAdminSet(groupname string, username string) error
	GroupUserAdminRemove(groupname, username string) error
	GroupUserAdd(groupname string, users []string) error
	GroupUserRemove(groupname, username string) error
	HatcheryRefresh(int64) error
	HatcheryRegister(sdk.Hatchery) (*sdk.Hatchery, bool, error)
	MonStatus() ([]string, error)
	MonDBTimes() (*sdk.MonDBTimes, error)
	MonDBMigrate() ([]sdk.MonDBMigrate, error)
	PipelineDelete(projectKey, name string) error
	PipelineCreate(projectKey string, pip *sdk.Pipeline) error
	PipelineExport(projectKey, name string, exportWithPermissions bool, exportFormat string) ([]byte, error)
	PipelineImport(projectKey string, content []byte, format string, force bool) ([]string, error)
	PipelineGroupsImport(projectKey, pipelineName string, content []byte, format string, force bool) (sdk.Pipeline, error)
	PipelineList(projectKey string) ([]sdk.Pipeline, error)
	ProjectCreate(proj *sdk.Project, groupName string) error
	ProjectDelete(projectKey string) error
	ProjectGet(projectKey string, opts ...RequestModifier) (*sdk.Project, error)
	ProjectList() ([]sdk.Project, error)
	ProjectKeysList(projectKey string) ([]sdk.ProjectKey, error)
	ProjectKeyCreate(projectKey string, key *sdk.ProjectKey) error
	ProjectKeysDelete(projectKey string, keyProjectName string) error
	ProjectVariablesList(key string) ([]sdk.Variable, error)
	ProjectVariableCreate(projectKey string, variable *sdk.Variable) error
	ProjectVariableDelete(projectKey string, varName string) error
	ProjectVariableGet(projectKey string, varName string) (*sdk.Variable, error)
	ProjectVariableUpdate(projectKey string, variable *sdk.Variable) error
	ProjectGroupsImport(projectKey string, content []byte, format string, force bool) (sdk.Project, error)
	Queue() ([]sdk.WorkflowNodeJobRun, []sdk.PipelineBuildJob, error)
	QueuePolling(context.Context, chan<- sdk.WorkflowNodeJobRun, chan<- sdk.PipelineBuildJob, chan<- error, time.Duration, int) error
	QueueTakeJob(sdk.WorkflowNodeJobRun, bool) (*worker.WorkflowNodeJobRunInfo, error)
	QueueJobBook(isWorkflowJob bool, id int64) error
	QueueJobInfo(id int64) (*sdk.WorkflowNodeJobRun, error)
	QueueJobSendSpawnInfo(isWorkflowJob bool, id int64, in []sdk.SpawnInfo) error
	QueueSendResult(int64, sdk.Result) error
	QueueArtifactUpload(id int64, tag, filePath string) error
	QueueJobTag(jobID int64, tags []sdk.WorkflowRunTag) error
	Requirements() ([]sdk.Requirement, error)
	ServiceRegister(sdk.Service) (string, error)
	TemplateList() ([]sdk.Template, error)
	TemplateGet(name string) (*sdk.Template, error)
	TemplateApplicationCreate(projectKey, name string, template *sdk.Template) error
	UserLogin(username, password string) (bool, string, error)
	UserList() ([]sdk.User, error)
	UserSignup(username, fullname, email, callback string) error
	UserGet(username string) (*sdk.User, error)
	UserGetGroups(username string) (map[string][]sdk.Group, error)
	UserReset(username, email, callback string) error
	UserConfirm(username, token string) (bool, string, error)
	Version() (*sdk.Version, error)
	WorkerList() ([]sdk.Worker, error)
	WorkerModelSpawnError(id int64, info string) error
	WorkerModelsEnabled() ([]sdk.Model, error)
	WorkerModels() ([]sdk.Model, error)
	WorkerRegister(worker.RegistrationForm) (*sdk.Worker, bool, error)
	WorkerSetStatus(sdk.Status) error
	WorkflowList(projectKey string) ([]sdk.Workflow, error)
	WorkflowGet(projectKey, name string) (*sdk.Workflow, error)
	WorkflowDelete(projectKey string, workflowName string) error
	WorkflowRunGet(projectKey string, name string, number int64) (*sdk.WorkflowRun, error)
	WorkflowRunArtifacts(projectKey string, name string, number int64) ([]sdk.Artifact, error)
	WorkflowRunFromHook(projectKey string, workflowName string, hook sdk.WorkflowNodeRunHookEvent) (*sdk.WorkflowRun, error)
	WorkflowRunFromManual(projectKey string, workflowName string, manual sdk.WorkflowNodeRunManual, number, fromNodeID int64) (*sdk.WorkflowRun, error)
	WorkflowNodeRun(projectKey string, name string, number int64, nodeRunID int64) (*sdk.WorkflowNodeRun, error)
	WorkflowNodeRunArtifacts(projectKey string, name string, number int64, nodeRunID int64) ([]sdk.Artifact, error)
	WorkflowNodeRunArtifactDownload(projectKey string, name string, artifactID int64, w io.Writer) error
	WorkflowNodeRunJobStep(projectKey string, workflowName string, number int64, nodeRunID, job int64, step int) (*sdk.BuildState, error)
	WorkflowNodeRunRelease(projectKey string, workflowName string, runNumber int64, nodeRunID int64, release sdk.WorkflowNodeRunRelease) error
	WorkflowAllHooksList() ([]sdk.WorkflowNodeHook, error)
}

// InterfaceDeprecated is the interface for using deprecated routes with cdsclient package
type InterfaceDeprecated interface {
	ApplicationPipelinesAttach(projectKey string, appName string, pipelineNames ...string) error
	ApplicationPipelineTriggerAdd(t *sdk.PipelineTrigger) error
	ApplicationPipelineTriggersGet(projectKey string, appName string, pipelineName string, envName string) ([]sdk.PipelineTrigger, error)
	AddHookOnRepositoriesManager(projectKey, appName, reposManager, repoFullname, pipelineName string) error
}
