package cdsclient

import (
	"archive/tar"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sguiheux/go-coverage"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/venom"
)

type Filter struct {
	Name, Value string
}

// TemplateClient exposes templates functions
type TemplateClient interface {
	TemplateGet(groupName, templateSlug string) (*sdk.WorkflowTemplate, error)
	TemplateGetAll() ([]sdk.WorkflowTemplate, error)
	TemplateApply(groupName, templateSlug string, req sdk.WorkflowTemplateRequest) (*tar.Reader, error)
	TemplateBulk(groupName, templateSlug string, req sdk.WorkflowTemplateBulk) (*sdk.WorkflowTemplateBulk, error)
	TemplateGetBulk(groupName, templateSlug string, id int64) (*sdk.WorkflowTemplateBulk, error)
	TemplatePull(groupName, templateSlug string) (*tar.Reader, error)
	TemplatePush(tarContent io.Reader) ([]string, *tar.Reader, error)
	TemplateDelete(groupName, templateSlug string) error
	TemplateGetInstances(groupName, templateSlug string) ([]sdk.WorkflowTemplateInstance, error)
	TemplateDeleteInstance(groupName, templateSlug string, id int64) error
}

// Admin expose all function to CDS administration
type Admin interface {
	AdminDatabaseMigrationDelete(id string) error
	AdminDatabaseMigrationUnlock(id string) error
	AdminDatabaseMigrationsList() ([]sdk.DatabaseMigrationStatus, error)
	AdminDatabaseSignaturesResume() (sdk.CanonicalFormUsageResume, error)
	AdminDatabaseSignaturesRollEntity(e string) error
	AdminDatabaseSignaturesRollAllEntities() error
	AdminDatabaseListEncryptedEntities() ([]string, error)
	AdminDatabaseRollEncryptedEntity(e string) error
	AdminDatabaseRollAllEncryptedEntities() error
	AdminCDSMigrationList() ([]sdk.Migration, error)
	AdminCDSMigrationCancel(id int64) error
	AdminCDSMigrationReset(id int64) error
	AdminWorkflowUpdateMaxRuns(projectKey string, workflowName string, maxRuns int64) error
	Features() ([]sdk.Feature, error)
	FeatureCreate(f sdk.Feature) error
	FeatureDelete(name string) error
	FeatureGet(name string) (sdk.Feature, error)
	FeatureUpdate(f sdk.Feature) error
	Services() ([]sdk.Service, error)
	ServicesByName(name string) (*sdk.Service, error)
	ServiceDelete(name string) error
	ServicesByType(stype string) ([]sdk.Service, error)
	ServiceNameCallGET(name string, url string) ([]byte, error)
	ServiceCallGET(stype string, url string) ([]byte, error)
	ServiceCallPOST(stype string, url string, body []byte) ([]byte, error)
	ServiceCallPUT(stype string, url string, body []byte) ([]byte, error)
	ServiceCallDELETE(stype string, url string) error
}

// ExportImportInterface exposes pipeline and application export and import function
type ExportImportInterface interface {
	PipelineExport(projectKey, name string, mods ...RequestModifier) ([]byte, error)
	PipelineImport(projectKey string, content io.Reader, mods ...RequestModifier) ([]string, error)
	ApplicationExport(projectKey, name string, mods ...RequestModifier) ([]byte, error)
	ApplicationImport(projectKey string, content io.Reader, mods ...RequestModifier) ([]string, error)
	WorkflowExport(projectKey, name string, mods ...RequestModifier) ([]byte, error)
	WorkflowPull(projectKey, name string, mods ...RequestModifier) (*tar.Reader, error)
	WorkflowImport(projectKey string, content io.Reader, mods ...RequestModifier) ([]string, error)
	WorkerModelExport(groupName, name string, mods ...RequestModifier) ([]byte, error)
	WorkerModelImport(content io.Reader, mods ...RequestModifier) (*sdk.Model, error)
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
	RepositoriesList(projectKey string, repoManager string, resync bool) ([]sdk.VCSRepo, error)
}

// ApplicationClient exposes application related functions
type ApplicationClient interface {
	ApplicationAttachToReposistoriesManager(projectKey, appName, reposManager, repoFullname string) error
	ApplicationCreate(projectKey string, app *sdk.Application) error
	ApplicationUpdate(projectKey string, appName string, app *sdk.Application) error
	ApplicationDelete(projectKey string, appName string) error
	ApplicationGet(projectKey string, appName string, opts ...RequestModifier) (*sdk.Application, error)
	ApplicationList(projectKey string) ([]sdk.Application, error)
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
	EnvironmentExport(projectKey, name string, mods ...RequestModifier) ([]byte, error)
	EnvironmentImport(projectKey string, content io.Reader, mods ...RequestModifier) ([]string, error)
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

// EventsClient listen SSE Events from CDS API
type EventsClient interface {
	// Must be run in a go routine
	WebsocketEventsListen(ctx context.Context, goRoutines *sdk.GoRoutines, chanMsgToSend <-chan []sdk.WebsocketFilter, chanMsgReceived chan<- sdk.WebsocketEvent, chanErrorReceived chan<- error)
}

// DownloadClient exposes download related functions
type DownloadClient interface {
	Download() ([]sdk.DownloadableResource, error)
	DownloadURLFromAPI(name, os, arch, variant string) string
	DownloadURLFromGithub(filename string) (string, error)
}

// ActionClient exposes actions related functions
type ActionClient interface {
	ActionDelete(groupName, name string) error
	ActionGet(groupName, name string, mods ...RequestModifier) (*sdk.Action, error)
	ActionUsage(groupName, name string, mods ...RequestModifier) (*sdk.ActionUsages, error)
	ActionList() ([]sdk.Action, error)
	ActionImport(content io.Reader, mods ...RequestModifier) error
	ActionExport(groupName, name string, mods ...RequestModifier) ([]byte, error)
	ActionBuiltinList() ([]sdk.Action, error)
	ActionBuiltinGet(name string, mods ...RequestModifier) (*sdk.Action, error)
}

// GroupClient exposes groups related functions
type GroupClient interface {
	GroupList() ([]sdk.Group, error)
	GroupGet(name string, mods ...RequestModifier) (*sdk.Group, error)
	GroupCreate(group *sdk.Group) error
	GroupRename(oldName, newName string) error
	GroupDelete(name string) error
	GroupMemberAdd(groupName string, member *sdk.GroupMember) (sdk.Group, error)
	GroupMemberEdit(groupName string, member *sdk.GroupMember) (sdk.Group, error)
	GroupMemberRemove(groupName, username string) error
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
	PipelineGet(projectKey, name string, mods ...RequestModifier) (*sdk.Pipeline, error)
	PipelineDelete(projectKey, name string) error
	PipelineCreate(projectKey string, pip *sdk.Pipeline) error
	PipelineList(projectKey string) ([]sdk.Pipeline, error)
}

// MaintenanceClient manage maintenance mode on CDS
type MaintenanceClient interface {
	Maintenance(enable bool, hooks bool) error
}

// ProjectClient exposes project related functions
type ProjectClient interface {
	ProjectCreate(proj *sdk.Project) error
	ProjectDelete(projectKey string) error
	ProjectGroupAdd(projectKey, groupName string, permission int, projectOnly bool) error
	ProjectGroupDelete(projectKey, groupName string) error
	ProjectGet(projectKey string, opts ...RequestModifier) (*sdk.Project, error)
	ProjectUpdate(key string, project *sdk.Project) error
	ProjectList(withApplications, withWorkflow bool, filters ...Filter) ([]sdk.Project, error)
	ProjectKeysClient
	ProjectVariablesClient
	ProjectGroupsImport(projectKey string, content io.Reader, mods ...RequestModifier) (sdk.Project, error)
	ProjectIntegrationImport(projectKey string, content io.Reader, mods ...RequestModifier) (sdk.ProjectIntegration, error)
	ProjectIntegrationGet(projectKey string, integrationName string, clearPassword bool) (sdk.ProjectIntegration, error)
	ProjectIntegrationList(projectKey string) ([]sdk.ProjectIntegration, error)
	ProjectIntegrationDelete(projectKey string, integrationName string) error
	ProjectRepositoryManagerList(projectKey string) ([]sdk.ProjectVCSServer, error)
	ProjectRepositoryManagerDelete(projectKey string, repoManagerName string, force bool) error
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
	QueueWorkflowNodeJobRun(status ...string) ([]sdk.WorkflowNodeJobRun, error)
	QueueCountWorkflowNodeJobRun(since *time.Time, until *time.Time, modelType string, ratioService *int) (sdk.WorkflowNodeJobRunCount, error)
	QueuePolling(ctx context.Context, goRoutines *sdk.GoRoutines, jobs chan<- sdk.WorkflowNodeJobRun, errs chan<- error, delay time.Duration, modelType string, ratioService *int) error
	QueueTakeJob(ctx context.Context, job sdk.WorkflowNodeJobRun) (*sdk.WorkflowNodeJobRunData, error)
	QueueJobBook(ctx context.Context, id int64) (sdk.WorkflowNodeJobRunBooked, error)
	QueueJobRelease(ctx context.Context, id int64) error
	QueueJobInfo(ctx context.Context, id int64) (*sdk.WorkflowNodeJobRun, error)
	QueueJobSendSpawnInfo(ctx context.Context, id int64, in []sdk.SpawnInfo) error
	QueueSendCoverage(ctx context.Context, id int64, report coverage.Report) error
	QueueSendUnitTests(ctx context.Context, id int64, report venom.Tests) error
	QueueSendLogs(ctx context.Context, id int64, log sdk.Log) error
	QueueSendVulnerability(ctx context.Context, id int64, report sdk.VulnerabilityWorkerReport) error
	QueueSendStepResult(ctx context.Context, id int64, res sdk.StepStatus) error
	QueueSendResult(ctx context.Context, id int64, res sdk.Result) error
	QueueArtifactUpload(ctx context.Context, projectKey, integrationName string, nodeJobRunID int64, tag, filePath string) (bool, time.Duration, error)
	QueueStaticFilesUpload(ctx context.Context, projectKey, integrationName string, nodeJobRunID int64, name, entrypoint, staticKey string, tarContent io.Reader) (string, bool, time.Duration, error)
	QueueJobTag(ctx context.Context, jobID int64, tags []sdk.WorkflowRunTag) error
	QueueServiceLogs(ctx context.Context, logs []sdk.ServiceLog) error
	QueueJobSetVersion(ctx context.Context, jobID int64, version sdk.WorkflowRunVersion) error
}

// UserClient exposes users functions
type UserClient interface {
	UserList() ([]sdk.AuthentifiedUser, error)
	UserGet(username string) (*sdk.AuthentifiedUser, error)
	UserGetMe() (*sdk.AuthentifiedUser, error)
	UserGetGroups(username string) (map[string][]sdk.Group, error)
	UpdateFavorite(params sdk.FavoriteParams) (interface{}, error)
	UserGetSchema() (sdk.SchemaResponse, error)
}

// WorkerClient exposes workers functions
type WorkerClient interface {
	WorkerGet(ctx context.Context, name string, mods ...RequestModifier) (*sdk.Worker, error)
	WorkerModelBook(groupName, name string) error
	WorkerList(ctx context.Context) ([]sdk.Worker, error)
	WorkerRefresh(ctx context.Context) error
	WorkerUnregister(ctx context.Context) error
	WorkerDisable(ctx context.Context, id string) error
	WorkerModelAdd(name, modelType, patternName string, dockerModel *sdk.ModelDocker, vmModel *sdk.ModelVirtualMachine, groupID int64) (sdk.Model, error)
	WorkerModelGet(groupName, name string) (sdk.Model, error)
	WorkerModelDelete(groupName, name string) error
	WorkerModelSpawnError(groupName, name string, info sdk.SpawnErrorForm) error
	WorkerModelList(*WorkerModelFilter) ([]sdk.Model, error)
	WorkerModelEnabledList() ([]sdk.Model, error)
	WorkerModelSecretList(groupName, name string) (sdk.WorkerModelSecrets, error)
	WorkerRegister(ctx context.Context, authToken string, form sdk.WorkerRegistrationForm) (*sdk.Worker, bool, error)
	WorkerSetStatus(ctx context.Context, status string) error
}

// HookClient exposes functions used for hooks services
type HookClient interface {
	PollVCSEvents(uuid string, workflowID int64, vcsServer string, timestamp int64) (events sdk.RepositoryEvents, interval time.Duration, err error)
	VCSConfiguration() (map[string]sdk.VCSConfiguration, error)
}

// ServiceClient exposes functions used for services
type ServiceClient interface {
	ServiceConfigurationGet(context.Context, string) ([]sdk.ServiceConfiguration, error)
}

// WorkflowClient exposes workflows functions
type WorkflowClient interface {
	WorkflowSearch(opts ...RequestModifier) ([]sdk.Workflow, error)
	WorkflowRunsAndNodesIDs(projectkey string) ([]sdk.WorkflowNodeRunIdentifiers, error)
	WorkflowList(projectKey string, opts ...RequestModifier) ([]sdk.Workflow, error)
	WorkflowGet(projectKey, name string, opts ...RequestModifier) (*sdk.Workflow, error)
	WorkflowUpdate(projectKey, name string, wf *sdk.Workflow) error
	WorkflowDelete(projectKey string, workflowName string) error
	WorkflowLabelAdd(projectKey, name, labelName string) error
	WorkflowLabelDelete(projectKey, name string, labelID int64) error
	WorkflowGroupAdd(projectKey, name, groupName string, permission int) error
	WorkflowGroupDelete(projectKey, name, groupName string) error
	WorkflowRunGet(projectKey string, workflowName string, number int64) (*sdk.WorkflowRun, error)
	WorkflowRunsDeleteByBranch(projectKey string, workflowName string, branch string) error
	WorkflowRunResync(projectKey string, workflowName string, number int64) (*sdk.WorkflowRun, error)
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
	WorkflowNodeRunJobStepLink(ctx context.Context, projectKey string, workflowName string, nodeRunID, job int64, step int64) (*sdk.CDNLogLink, error)
	WorkflowNodeRunJobStepLog(ctx context.Context, projectKey string, workflowName string, nodeRunID, job int64, step int64) (*sdk.BuildState, error)
	WorkflowNodeRunJobServiceLink(ctx context.Context, projectKey string, workflowName string, nodeRunID, job int64, serviceName string) (*sdk.CDNLogLink, error)
	WorkflowNodeRunJobServiceLog(ctx context.Context, projectKey string, workflowName string, nodeRunID, job int64, serviceName string) (*sdk.ServiceLog, error)
	WorkflowLogAccess(ctx context.Context, projectKey, workflowName, sessionID string) error
	WorkflowLogDownload(ctx context.Context, link sdk.CDNLogLink) ([]byte, error)
	WorkflowNodeRunRelease(projectKey string, workflowName string, runNumber int64, nodeRunID int64, release sdk.WorkflowNodeRunRelease) error
	WorkflowAllHooksList() ([]sdk.NodeHook, error)
	WorkflowCachePush(projectKey, integrationName, ref string, tarContent io.Reader, size int) error
	WorkflowCachePull(projectKey, integrationName, ref string) (io.Reader, error)
	WorkflowTransformAsCode(projectKey, workflowName, branch, message string) (*sdk.Operation, error)
}

// MonitoringClient exposes monitoring functions
type MonitoringClient interface {
	MonStatus() (*sdk.MonitoringStatus, error)
	MonVersion() (*sdk.Version, error)
	MonDBMigrate() ([]sdk.MonDBMigrate, error)
	MonErrorsGet(requestID string) ([]sdk.Error, error)
}

// IntegrationClient exposes integration functions
type IntegrationClient interface {
	IntegrationModelList() ([]sdk.IntegrationModel, error)
	IntegrationModelGet(name string) (sdk.IntegrationModel, error)
	IntegrationModelAdd(m *sdk.IntegrationModel) error
	IntegrationModelUpdate(m *sdk.IntegrationModel) error
	IntegrationModelDelete(name string) error
}

// Interface is the main interface for cdsclient package
// generate mock with "mockgen -source=interface.go -destination=mock_cdsclient/interface_mock.go Interface" from directory ${GOPATH}/src/github.com/ovh/cds/sdk/cdsclient
type Interface interface {
	Raw
	AuthClient
	ActionClient
	Admin
	APIURL() string
	ApplicationClient
	ConfigUser() (sdk.ConfigUser, error)
	ConfigCDN() (sdk.CDNConfig, error)
	DownloadClient
	EnvironmentClient
	EventsClient
	ExportImportInterface
	FeatureEnabled(name string, params map[string]string) (sdk.FeatureEnabledResponse, error)
	GroupClient
	GRPCPluginsClient
	BroadcastClient
	MaintenanceClient
	PipelineClient
	IntegrationClient
	ProjectClient
	QueueClient
	Navbar() ([]sdk.NavbarProjectData, error)
	Requirements() ([]sdk.Requirement, error)
	RepositoriesManagerInterface
	ServiceClient
	ServiceRegister(context.Context, sdk.Service) (*sdk.Service, error)
	ServiceHeartbeat(*sdk.MonitoringStatus) error
	UserClient
	WorkerClient
	WorkflowClient
	MonitoringClient
	HookClient
	Version() (*sdk.Version, error)
	TemplateClient
	WebsocketClient
}

type WorkerInterface interface {
	GRPCPluginsClient
	ProjectIntegrationGet(projectKey string, integrationName string, clearPassword bool) (sdk.ProjectIntegration, error)
	QueueClient
	Requirements() ([]sdk.Requirement, error)
	ServiceClient
	WorkerClient
	WorkflowRunArtifacts(projectKey string, name string, number int64) ([]sdk.WorkflowNodeRunArtifact, error)
	WorkflowCachePush(projectKey, integrationName, ref string, tarContent io.Reader, size int) error
	WorkflowCachePull(projectKey, integrationName, ref string) (io.Reader, error)
	WorkflowRunList(projectKey string, workflowName string, offset, limit int64) ([]sdk.WorkflowRun, error)
	WorkflowNodeRunArtifactDownload(projectKey string, name string, a sdk.WorkflowNodeRunArtifact, w io.Writer) error
	WorkflowNodeRunRelease(projectKey string, workflowName string, runNumber int64, nodeRunID int64, release sdk.WorkflowNodeRunRelease) error
}

// Raw is a low-level interface exposing HTTP functions
type Raw interface {
	PostJSON(ctx context.Context, path string, in interface{}, out interface{}, mods ...RequestModifier) (int, error)
	PutJSON(ctx context.Context, path string, in interface{}, out interface{}, mods ...RequestModifier) (int, error)
	GetJSON(ctx context.Context, path string, out interface{}, mods ...RequestModifier) (int, error)
	DeleteJSON(ctx context.Context, path string, out interface{}, mods ...RequestModifier) (int, error)
	RequestJSON(ctx context.Context, method, path string, in interface{}, out interface{}, mods ...RequestModifier) ([]byte, http.Header, int, error)
	Request(ctx context.Context, method string, path string, body io.Reader, mods ...RequestModifier) ([]byte, http.Header, int, error)
	HTTPClient() *http.Client
	HTTPSSEClient() *http.Client
	HTTPWebsocketClient() *websocket.Dialer
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
	ApplicationDeploymentStrategyUpdate(projectKey, applicationName, integrationName string, config sdk.IntegrationConfig) error
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

// WithWorkflows allow a provider to retrieve a pipeline with its workflows usage
func WithWorkflows() RequestModifier {
	return func(r *http.Request) {
		q := r.URL.Query()
		q.Set("withWorkflows", "true")
		r.URL.RawQuery = q.Encode()
	}
}

// WithLabels allow a provider to retrieve a workflow with its labels
func WithLabels() RequestModifier {
	return func(r *http.Request) {
		q := r.URL.Query()
		q.Set("withLabels", "true")
		r.URL.RawQuery = q.Encode()
	}
}

// WithPermissions allow a provider to retrieve a workflow with its permissions.
func WithPermissions() RequestModifier {
	return func(r *http.Request) {
		q := r.URL.Query()
		q.Set("withPermissions", "true")
		r.URL.RawQuery = q.Encode()
	}
}

// WithKeys allow a provider to retrieve a project with its keys.
func WithKeys() RequestModifier {
	return func(r *http.Request) {
		q := r.URL.Query()
		q.Set("withKeys", "true")
		r.URL.RawQuery = q.Encode()
	}
}

// WithTemplate allow a provider to retrieve a workflow with template if exists.
func WithTemplate() RequestModifier {
	return func(r *http.Request) {
		q := r.URL.Query()
		q.Set("withTemplate", "true")
		r.URL.RawQuery = q.Encode()
	}
}

func Format(format string) RequestModifier {
	return func(r *http.Request) {
		q := r.URL.Query()
		q.Set("format", url.QueryEscape(format))
		r.URL.RawQuery = q.Encode()
	}
}

func Force() RequestModifier {
	return func(r *http.Request) {
		q := r.URL.Query()
		q.Set("force", "true")
		r.URL.RawQuery = q.Encode()
	}
}

func ContentType(value string) RequestModifier {
	return func(r *http.Request) {
		r.Header.Add("Content-Type", value)
	}
}

// AuthClient is the interface for authentication management.
type AuthClient interface {
	AuthDriverList() (sdk.AuthDriverResponse, error)
	AuthConsumerSignin(sdk.AuthConsumerType, sdk.AuthConsumerSigninRequest) (sdk.AuthConsumerSigninResponse, error)
	AuthConsumerLocalAskResetPassword(sdk.AuthConsumerSigninRequest) error
	AuthConsumerLocalResetPassword(token, newPassword string) (sdk.AuthConsumerSigninResponse, error)
	AuthConsumerLocalSignup(sdk.AuthConsumerSigninRequest) error
	AuthConsumerLocalSignupVerify(token, initToken string) (sdk.AuthConsumerSigninResponse, error)
	AuthConsumerSignout() error
	AuthConsumerListByUser(username string) (sdk.AuthConsumers, error)
	AuthConsumerDelete(username, id string) error
	AuthConsumerRegen(username, id string) (sdk.AuthConsumerCreateResponse, error)
	AuthConsumerCreateForUser(username string, request sdk.AuthConsumer) (sdk.AuthConsumerCreateResponse, error)
	AuthSessionListByUser(username string) (sdk.AuthSessions, error)
	AuthSessionDelete(username, id string) error
	AuthSessionGet(id string) (sdk.AuthCurrentConsumerResponse, error)
	AuthMe() (sdk.AuthCurrentConsumerResponse, error)
}

type WebsocketClient interface {
	RequestWebsocket(ctx context.Context, goRoutines *sdk.GoRoutines, path string, msgToSend <-chan json.RawMessage, msgReceived chan<- json.RawMessage, errorReceived chan<- error) error
}
