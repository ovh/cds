package cdsclient

import (
	"archive/tar"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/ovh/cds/sdk"
)

type Filter struct {
	Name, Value string
}

// AdminService expose all function to CDS services
type AdminService interface {
	Services() ([]sdk.Service, error)
	ServicesByName(name string) (*sdk.Service, error)
	ServicesByType(stype string) ([]sdk.Service, error)
	ServiceCallGET(stype string, url string) ([]byte, error)
	ServiceCallPOST(stype string, url string, body []byte) ([]byte, error)
	ServiceCallPUT(stype string, url string, body []byte) ([]byte, error)
	ServiceCallDELETE(stype string, url string) error
}

// ExportImportInterface exposes pipeline and application export and import function
type ExportImportInterface interface {
	PipelineExport(projectKey, name string, exportWithPermissions bool, exportFormat string) ([]byte, error)
	PipelineImport(projectKey string, content io.Reader, format string, force bool) ([]string, error)
	ApplicationExport(projectKey, name string, exportWithPermissions bool, format string) ([]byte, error)
	ApplicationImport(projectKey string, content io.Reader, format string, force bool) ([]string, error)
	WorkflowExport(projectKey, name string, exportWithPermissions bool, exportFormat string) ([]byte, error)
	WorkflowPull(projectKey, name string, exportWithPermissions bool) (*tar.Reader, error)
	WorkflowImport(projectKey string, content io.Reader, format string, force bool) ([]string, error)
	WorkflowPush(projectKey string, tarContent io.Reader, mods ...RequestModifier) ([]string, *tar.Reader, error)
	WorkflowAsCodeInterface
}

// WorkflowAsCodeInterface exposes all workflow as code functions
type WorkflowAsCodeInterface interface {
	WorkflowAsCodeStart(projectKey string, repoURL string, repoStrategy sdk.RepositoryStrategy) (*sdk.Operation, error)
	WorkflowAsCodeInfo(projectKey string, operationID string) (*sdk.Operation, error)
	WorkflowAsCodePerform(projectKey string, operationID string) ([]string, error)
}

// RepositoriesManagerInterface exposes all repostories manager functions
type RepositoriesManagerInterface interface {
	RepositoriesList(projectKey string, repoManager string) ([]sdk.VCSRepo, error)
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
	EnvironmentExport(projectKey, name string, exportWithPermissions bool, format string) ([]byte, error)
	EnvironmentImport(projectKey string, content io.Reader, format string, force bool) ([]string, error)
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
	DownloadURLFromGithub(filename string) (string, error)
}

// ActionClient exposes actions related functions
type ActionClient interface {
	ActionDelete(actionName string) error
	ActionGet(actionName string, mods ...RequestModifier) (*sdk.Action, error)
	ActionList() ([]sdk.Action, error)
	ActionImport(content io.Reader, format string) error
	ActionExport(name string, format string) ([]byte, error)
}

// GroupClient exposes groups related functions
type GroupClient interface {
	GroupCreate(group *sdk.Group) error
	GroupDelete(name string) error
	GroupGenerateToken(groupName, expiration, description string) (*sdk.Token, error)
	GroupListToken(groupName string) ([]sdk.Token, error)
	GroupDeleteToken(groupName string, tokenID int64) error
	GroupGet(name string, mods ...RequestModifier) (*sdk.Group, error)
	GroupList() ([]sdk.Group, error)
	GroupUserAdminSet(groupname string, username string) error
	GroupUserAdminRemove(groupname, username string) error
	GroupUserAdd(groupname string, users []string) error
	GroupUserRemove(groupname, username string) error
	GroupRename(oldGroupname, newGroupname string) error
}

// HatcheryClient exposes hatcheries related functions
type HatcheryClient interface {
	HatcheryCount(wfNodeRunID int64) (int64, error)
}

// BroadcastClient expose all function for CDS Broadcasts
type BroadcastClient interface {
	Broadcasts() ([]sdk.Broadcast, error)
	BroadcastCreate(broadcast *sdk.Broadcast) error
	BroadcastGet(id string) (*sdk.Broadcast, error)
	BroadcastDelete(id string) error
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
	ProjectList(withApplications, withWorkflow bool, filters ...Filter) ([]sdk.Project, error)
	ProjectKeysClient
	ProjectVariablesClient
	ProjectGroupsImport(projectKey string, content io.Reader, format string, force bool) (sdk.Project, error)
	ProjectPlatformImport(projectKey string, content io.Reader, format string, force bool) (sdk.ProjectPlatform, error)
	ProjectPlatformGet(projectKey string, platformName string, clearPassword bool) (sdk.ProjectPlatform, error)
	ProjectPlatformList(projectKey string) ([]sdk.ProjectPlatform, error)
	ProjectPlatformDelete(projectKey string, platformName string) error
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
	VariableEncrypt(projectKey string, varName string, content string) (*sdk.Variable, error)
}

// QueueClient exposes queue related functions
type QueueClient interface {
	QueueWorkflowNodeJobRun(status ...sdk.Status) ([]sdk.WorkflowNodeJobRun, error)
	QueueCountWorkflowNodeJobRun(since *time.Time, until *time.Time, modelType string, ratioService *int) (sdk.WorkflowNodeJobRunCount, error)
	QueuePipelineBuildJob() ([]sdk.PipelineBuildJob, error)
	QueuePolling(ctx context.Context, jobs chan<- sdk.WorkflowNodeJobRun, pbjobs chan<- sdk.PipelineBuildJob, errs chan<- error, delay time.Duration, graceTime int, modelType string, ratioService *int, exceptWfJobID *int64) error
	QueueTakeJob(sdk.WorkflowNodeJobRun, bool) (*sdk.WorkflowNodeJobRunData, error)
	QueueJobBook(isWorkflowJob bool, id int64) error
	QueueJobRelease(isWorkflowJob bool, id int64) error
	QueueJobInfo(id int64) (*sdk.WorkflowNodeJobRun, error)
	QueueJobSendSpawnInfo(isWorkflowJob bool, id int64, in []sdk.SpawnInfo) error
	QueueSendResult(int64, sdk.Result) error
	QueueArtifactUpload(id int64, tag, filePath string) (bool, time.Duration, error)
	QueueJobTag(jobID int64, tags []sdk.WorkflowRunTag) error
	QueueJobIncAttempts(jobID int64) ([]int64, error)
	QueueServiceLogs(logs []sdk.ServiceLog) error
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
	ListAllTokens() ([]sdk.Token, error)
	FindToken(token string) (sdk.Token, error)
	UpdateFavorite(params sdk.FavoriteParams) (interface{}, error)
}

// WorkerClient exposes workers functions
type WorkerClient interface {
	WorkerModelBook(id int64) error
	WorkerList() ([]sdk.Worker, error)
	WorkerRefresh() error
	WorkerDisable(id string) error
	WorkerModelAdd(name, modelType, patternName string, dockerModel *sdk.ModelDocker, vmModel *sdk.ModelVirtualMachine, groupID int64) (sdk.Model, error)
	WorkerModelUpdate(ID int64, name string, modelType string, dockerModel *sdk.ModelDocker, vmModel *sdk.ModelVirtualMachine, groupID int64) (sdk.Model, error)
	WorkerModel(name string) (sdk.Model, error)
	WorkerModelDelete(name string) error
	WorkerModelSpawnError(id int64, info string) error
	WorkerModelsEnabled() ([]sdk.Model, error)
	WorkerModels() ([]sdk.Model, error)
	WorkerModelsByBinary(binary string) ([]sdk.Model, error)
	WorkerModelsByState(state string) ([]sdk.Model, error)
	WorkerRegister(sdk.WorkerRegistrationForm) (*sdk.Worker, bool, error)
	WorkerSetStatus(sdk.Status) error
}

// HookClient exposes functions used for hooks services
type HookClient interface {
	PollVCSEvents(uuid string, workflowID int64, vcsServer string, timestamp int64) (events sdk.RepositoryEvents, interval time.Duration, err error)
}

// WorkflowClient exposes workflows functions
type WorkflowClient interface {
	WorkflowList(projectKey string) ([]sdk.Workflow, error)
	WorkflowGet(projectKey, name string) (*sdk.Workflow, error)
	WorkflowDelete(projectKey string, workflowName string) error
	WorkflowRunGet(projectKey string, workflowName string, number int64) (*sdk.WorkflowRun, error)
	WorkflowRunSearch(projectKey string, offset, limit int64, filter ...Filter) ([]sdk.WorkflowRun, error)
	WorkflowRunList(projectKey string, workflowName string, offset, limit int64) ([]sdk.WorkflowRun, error)
	WorkflowRunArtifacts(projectKey string, name string, number int64) ([]sdk.WorkflowNodeRunArtifact, error)
	WorkflowRunFromHook(projectKey string, workflowName string, hook sdk.WorkflowNodeRunHookEvent) (*sdk.WorkflowRun, error)
	WorkflowRunFromManual(projectKey string, workflowName string, manual sdk.WorkflowNodeRunManual, number, fromNodeID int64) (*sdk.WorkflowRun, error)
	WorkflowRunNumberGet(projectKey string, workflowName string) (*sdk.WorkflowRunNumber, error)
	WorkflowRunNumberSet(projectKey string, workflowName string, number int64) error
	WorkflowStop(projectKey string, workflowName string, number int64) (*sdk.WorkflowRun, error)
	WorkflowNodeStop(projectKey string, workflowName string, number, fromNodeID int64) (*sdk.WorkflowNodeRun, error)
	WorkflowNodeRun(projectKey string, name string, number int64, nodeRunID int64) (*sdk.WorkflowNodeRun, error)
	WorkflowNodeRunArtifactDownload(projectKey string, name string, a sdk.WorkflowNodeRunArtifact, w io.Writer) error
	WorkflowNodeRunJobStep(projectKey string, workflowName string, number int64, nodeRunID, job int64, step int) (*sdk.BuildState, error)
	WorkflowNodeRunRelease(projectKey string, workflowName string, runNumber int64, nodeRunID int64, release sdk.WorkflowNodeRunRelease) error
	WorkflowAllHooksList() ([]sdk.WorkflowNodeHook, error)
	WorkflowCachePush(projectKey, ref string, tarContent io.Reader) error
	WorkflowCachePull(projectKey, ref string) (io.Reader, error)
}

// MonitoringClient exposes monitoring functions
type MonitoringClient interface {
	MonStatus() (*sdk.MonitoringStatus, error)
	MonVersion() (*sdk.Version, error)
	MonDBMigrate() ([]sdk.MonDBMigrate, error)
}

// PlatformClient exposes platform functions
type PlatformClient interface {
	PlatformModelList() ([]sdk.PlatformModel, error)
	PlatformModelGet(name string) (sdk.PlatformModel, error)
	PlatformModelAdd(m *sdk.PlatformModel) error
	PlatformModelUpdate(m *sdk.PlatformModel) error
	PlatformModelDelete(name string) error
}

// Interface is the main interface for cdsclient package
type Interface interface {
	ActionClient
	AdminService
	APIURL() string
	ApplicationClient
	ConfigUser() (map[string]string, error)
	DownloadClient
	EnvironmentClient
	ExportImportInterface
	GroupClient
	GRPCPluginsClient
	HatcheryClient
	BroadcastClient
	PipelineClient
	PlatformClient
	ProjectClient
	QueueClient
	Navbar() ([]sdk.NavbarProjectData, error)
	Requirements() ([]sdk.Requirement, error)
	RepositoriesManagerInterface
	GetService() *sdk.Service
	ServiceRegister(sdk.Service) (string, error)
	UserClient
	WorkerClient
	WorkflowClient
	MonitoringClient
	HookClient
	Version() (*sdk.Version, error)
}

// InterfaceDeprecated is the interface for using deprecated routes with cdsclient package
type InterfaceDeprecated interface {
	ApplicationPipelinesAttach(projectKey string, appName string, pipelineNames ...string) error
	ApplicationPipelineTriggerAdd(t *sdk.PipelineTrigger) error
	ApplicationPipelineTriggersGet(projectKey string, appName string, pipelineName string, envName string) ([]sdk.PipelineTrigger, error)
	AddHookOnRepositoriesManager(projectKey, appName, reposManager, repoFullname, pipelineName string) error
	ApplicationDoMigrationWorkflow(projectKey, appName string, force, disablePrefixW, withCurrentVersion, withRepositoryWebHook bool) error
	ApplicationCleanOldWorkflow(projectKey, appName string) error
}

// Raw is a low-level interface exposing HTTP functions
type Raw interface {
	PostJSON(path string, in interface{}, out interface{}, mods ...RequestModifier) (int, error)
	PutJSON(path string, in interface{}, out interface{}, mods ...RequestModifier) (int, error)
	GetJSON(path string, out interface{}, mods ...RequestModifier) (int, error)
	DeleteJSON(path string, out interface{}, mods ...RequestModifier) (int, error)
	Request(method string, path string, body io.Reader, mods ...RequestModifier) ([]byte, http.Header, int, error)
}

// GRPCPluginsClient exposes plugins API
type GRPCPluginsClient interface {
	PluginsList() ([]sdk.GRPCPlugin, error)
	PluginsGet(string) (*sdk.GRPCPlugin, error)
	PluginAdd(*sdk.GRPCPlugin) error
	PluginUpdate(*sdk.GRPCPlugin) error
	PluginDelete(string) error
	PluginAddBinary(*sdk.GRPCPlugin, *sdk.GRPCPluginBinary) error
	PluginDeleteBinary(name, os, arch string) error
	PluginGetBinary(name, os, arch string, w io.Writer) error
	PluginGetBinaryInfos(name, os, arch string) (*sdk.GRPCPluginBinary, error)
}

/* ProviderClient exposes allowed methods for providers
 Usage:

 	cfg := ProviderConfig{
		Host: "https://my-cds-api:8081",
		Name: "my-provider-name",
		Token: "my-very-long-secret-token",
	}
	client := NewProviderClient(cfg)
	//Get the writable projects of a user
	projects, err := client.ProjectsList(FilterByUser("a-username"), FilterByWritablePermission())
	...
*/
type ProviderClient interface {
	ApplicationsList(projectKey string, opts ...RequestModifier) ([]sdk.Application, error)
	ApplicationDeploymentStrategyUpdate(projectKey, applicationName, platformName string, config sdk.PlatformConfig) error
	ApplicationMetadataUpdate(projectKey, applicationName, key, value string) error
	ProjectsList(opts ...RequestModifier) ([]sdk.Project, error)
	WorkflowsList(projectKey string) ([]sdk.Workflow, error)
	WorkflowLoad(projectKey, workflowName string) (*sdk.Workflow, error)
}

// FilterByUser allow a provider to perform a request as a user identified by its username
func FilterByUser(username string) RequestModifier {
	return func(req *http.Request) {
		req.Header.Set("X-Cds-Username", username)
	}
}

// FilterByWritablePermission allow a provider to filter only writable objects
func FilterByWritablePermission() RequestModifier {
	return func(r *http.Request) {
		q := r.URL.Query()
		q.Set("permission", "W")
		r.URL.RawQuery = q.Encode()
	}
}

// WithUsage allow a provider to retrieve an application with its usage
func WithUsage() RequestModifier {
	return func(r *http.Request) {
		q := r.URL.Query()
		q.Set("withUsage", "true")
		r.URL.RawQuery = q.Encode()
	}
}
