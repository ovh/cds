package cdsclient

import (
	"archive/tar"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/spf13/afero"

	"github.com/gorilla/websocket"

	"github.com/ovh/cds/sdk"
)

type Filter struct {
	Name, Value string
}

// TemplateClient exposes templates functions
type TemplateClient interface {
	TemplateGet(groupName, templateSlug string) (*sdk.WorkflowTemplate, error)
	TemplateGetAll() ([]sdk.WorkflowTemplate, error)
	TemplateApply(groupName, templateSlug string, req sdk.WorkflowTemplateRequest, mods ...RequestModifier) (*tar.Reader, error)
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
	AdminDatabaseMigrationDelete(service string, id string) error
	AdminDatabaseMigrationUnlock(service string, id string) error
	AdminDatabaseMigrationsList(service string) ([]sdk.DatabaseMigrationStatus, error)
	AdminDatabaseSignaturesResume(service string) (sdk.CanonicalFormUsageResume, error)
	AdminDatabaseSignaturesRollEntity(service string, e string, idx *int64) error
	AdminDatabaseSignaturesRollAllEntities(service string) error
	AdminDatabaseListEncryptedEntities(service string) ([]string, error)
	AdminDatabaseRollEncryptedEntity(service string, e string, idx *int64) error
	AdminDatabaseRollAllEncryptedEntities(service string) error
	AdminCDSMigrationList() ([]sdk.Migration, error)
	AdminCDSMigrationCancel(id int64) error
	AdminCDSMigrationReset(id int64) error
	AdminWorkflowUpdateMaxRuns(projectKey string, workflowName string, maxRuns int64) error
	AdminOrganizationCreate(ctx context.Context, orga sdk.Organization) error
	AdminOrganizationList(ctx context.Context) ([]sdk.Organization, error)
	AdminOrganizationDelete(ctx context.Context, orgaIdentifier string) error
	AdminOrganizationMigrateUser(ctx context.Context, orgaIdentifier string) error
	HasProjectRole(ctx context.Context, projectKey, sessionID string, role string) error
	Features() ([]sdk.Feature, error)
	FeatureCreate(f sdk.Feature) error
	FeatureDelete(name sdk.FeatureName) error
	FeatureGet(name sdk.FeatureName) (sdk.Feature, error)
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

type OrganizationClient interface {
	OrganizationAdd(ctx context.Context, organization sdk.Organization) error
	OrganizationGet(ctx context.Context, organizationIdentifier string) (sdk.Organization, error)
	OrganizationList(ctx context.Context) ([]sdk.Organization, error)
	OrganizationDelete(ctx context.Context, organizationIdentifier string) error
}

type RegionClient interface {
	RegionAdd(ctx context.Context, region sdk.Region) error
	RegionGet(ctx context.Context, regionIdentifier string) (sdk.Region, error)
	RegionList(ctx context.Context) ([]sdk.Region, error)
	RegionDelete(ctx context.Context, regionIdentifier string) error
}

type HatcheryClient interface {
	HatcheryAdd(ctx context.Context, h *sdk.Hatchery) (*sdk.HatcheryGetResponse, error)
	HatcheryGet(ctx context.Context, hatcheryIdentifier string) (sdk.HatcheryGetResponse, error)
	HatcheryList(ctx context.Context) ([]sdk.HatcheryGetResponse, error)
	HatcheryDelete(ctx context.Context, hatcheryIdentifier string) error
	HatcheryRegenToken(ctx context.Context, hatcheryIdentifier string) (*sdk.HatcheryGetResponse, error)
}

type HatcheryServiceClient interface {
	Heartbeat(ctx context.Context, mon *sdk.MonitoringStatus) error
	GetWorkerModel(ctx context.Context, projKey string, vcsIdentifier string, repoIdentifier string, workerModelName string, mods ...RequestModifier) (*sdk.V2WorkerModel, error)
	V2HatcheryTakeJob(ctx context.Context, regionName string, jobRunID string) (*sdk.V2WorkflowRunJob, error)

	V2HatcheryReleaseJob(ctx context.Context, regionName string, jobRunID string) error
	EntityGet(ctx context.Context, projKey string, vcsIdentifier string, repoIdentifier string, entityType string, entityName string, mods ...RequestModifier) (*sdk.Entity, error)
	V2QueueClient
	V2WorkerList(ctx context.Context) ([]sdk.V2Worker, error)
}

// ProjectClientV2 exposes project related functions
type ProjectClientV2 interface {
	ProjectNotificationCreate(ctx context.Context, pKey string, notif *sdk.ProjectNotification) error
	ProjectNotificationUpdate(ctx context.Context, pKey string, notif *sdk.ProjectNotification) error
	ProjectNotificationDelete(ctx context.Context, pKey string, notifName string) error
	ProjectNotificationGet(ctx context.Context, pKey string, notifName string) (*sdk.ProjectNotification, error)
	ProjectNotificationList(ctx context.Context, pKey string) ([]sdk.ProjectNotification, error)

	ProjectVariableSetCreate(ctx context.Context, pKey string, vs *sdk.ProjectVariableSet) error
	ProjectVariableSetDelete(ctx context.Context, pKey string, vsName string, mod ...RequestModifier) error
	ProjectVariableSetList(ctx context.Context, pKey string) ([]sdk.ProjectVariableSet, error)
	ProjectVariableSetShow(ctx context.Context, pKey string, vsName string) (*sdk.ProjectVariableSet, error)

	ProjectVariableSetItemAdd(ctx context.Context, pKey string, vsName string, item *sdk.ProjectVariableSetItem) error
	ProjectVariableSetItemUpdate(ctx context.Context, pKey string, vsName string, item *sdk.ProjectVariableSetItem) error
	ProjectVariableSetItemDelete(ctx context.Context, pKey string, vsName string, itemName string) error
	ProjectVariableSetItemGet(ctx context.Context, pKey string, vsName string, itemName string) (*sdk.ProjectVariableSetItem, error)
}

// ProjectClient exposes project related functions
type ProjectClient interface {
	ProjectCreate(proj *sdk.Project) error
	ProjectDelete(projectKey string) error
	ProjectGroupAdd(projectKey, groupName string, permission int, projectOnly bool) error
	ProjectGroupDelete(projectKey, groupName string) error
	ProjectGet(projectKey string, opts ...RequestModifier) (*sdk.Project, error)
	ProjectUpdate(key string, project *sdk.Project) error
	ProjectList(withApplications, withWorkflow, withFavorites bool, filters ...Filter) ([]sdk.Project, error)
	ProjectKeysClient
	ProjectVariablesClient
	ProjectIntegrationImport(projectKey string, content io.Reader, mods ...RequestModifier) (sdk.ProjectIntegration, error)
	ProjectIntegrationGet(projectKey string, integrationName string, clearPassword bool) (sdk.ProjectIntegration, error)
	ProjectIntegrationList(projectKey string) ([]sdk.ProjectIntegration, error)
	ProjectIntegrationDelete(projectKey string, integrationName string) error
	ProjectAccess(ctx context.Context, projectKey, sessionID string, itemType sdk.CDNItemType) error
	ProjectIntegrationWorkerHookGet(projectKey string, integrationName string) (*sdk.WorkerHookProjectIntegrationModel, error)
	ProjectIntegrationWorkerHooksImport(projectKey string, integrationName string, hook sdk.WorkerHookProjectIntegrationModel) error
	ProjectVCSImport(ctx context.Context, projectKey string, vcs sdk.VCSProject, mods ...RequestModifier) (sdk.VCSProject, error)
	ProjectVCSGet(ctx context.Context, projectKey string, integrationName string) (sdk.VCSProject, error)
	ProjectVCSList(ctx context.Context, projectKey string) ([]sdk.VCSProject, error)
	ProjectVCSDelete(ctx context.Context, projectKey string, vcsName string) error
	ProjectVCSRepositoryAdd(ctx context.Context, projectKey string, vcsName string, repo sdk.ProjectRepository) error
	ProjectVCSRepositoryList(ctx context.Context, projectKey string, vcsName string) ([]sdk.ProjectRepository, error)
	ProjectRepositoryHookSecret(ctx context.Context, projectKey, vcsType, vcsName, repoName string) (sdk.HookAccessData, error)
	ProjectRepositoryDelete(ctx context.Context, projectKey string, vcsName string, repositoryName string) error
	ProjectRepositoryAnalysis(ctx context.Context, analysis sdk.AnalysisRequest) (sdk.AnalysisResponse, error)
	ProjectRepositoryAnalysisList(ctx context.Context, projectKey string, vcsIdentifier string, repositoryIdentifier string) ([]sdk.ProjectRepositoryAnalysis, error)
	ProjectRepositoryAnalysisGet(ctx context.Context, projectKey string, vcsIdentifier string, repositoryIdentifier string, analysisID string) (sdk.ProjectRepositoryAnalysis, error)
	ProjectRepositoryEvents(ctx context.Context, projectKey, vcsName, repoName string) ([]sdk.HookRepositoryEvent, error)
	ProjectRepositoryEvent(ctx context.Context, projectKey, vcsName, repoName, eventID string) (*sdk.HookRepositoryEvent, error)
}

type RBACClient interface {
	RBACImport(ctx context.Context, rbacRule sdk.RBAC, mods ...RequestModifier) (sdk.RBAC, error)
	RBACDelete(ctx context.Context, permissionIdentifier string) error
	RBACGet(ctx context.Context, permissionIdentifier string) (sdk.RBAC, error)
	RBACList(ctx context.Context) ([]sdk.RBAC, error)
}

// ProjectKeysClient exposes project keys related functions
type ProjectKeysClient interface {
	ProjectKeysList(projectKey string) ([]sdk.ProjectKey, error)
	ProjectKeyCreate(projectKey string, key *sdk.ProjectKey) error
	ProjectKeysDelete(projectKey string, keyProjectName string) error
	ProjectKeysDisable(projectKey string, keyProjectName string) error
	ProjectKeysEnable(projectKey string, keyProjectName string) error
}

// ProjectVariablesClient exposes project variables related functions
type ProjectVariablesClient interface {
	ProjectVariablesList(key string) ([]sdk.Variable, error)
	ProjectVariableCreate(projectKey string, variable *sdk.Variable) error
	ProjectVariableDelete(projectKey string, varName string) error
	ProjectVariableGet(projectKey string, varName string) (*sdk.Variable, error)
	ProjectVariableUpdate(projectKey string, variable *sdk.Variable) error
	VariableEncrypt(projectKey string, varName string, content string) (*sdk.Variable, error)
	VariableListEncrypt(projectKey string) ([]sdk.Secret, error)
	VariableEncryptDelete(projectKey, name string) error
}

type V2QueueClient interface {
	V2QueueGetJobRun(ctx context.Context, regionName string, id string) (*sdk.V2QueueJobInfo, error)
	V2QueuePolling(ctx context.Context, region string, goRoutines *sdk.GoRoutines, hatcheryMetrics *sdk.HatcheryMetrics, pendingWorkerCreation *sdk.HatcheryPendingWorkerCreation, jobs chan<- sdk.V2QueueJobInfo, errs chan<- error, delay time.Duration, ms ...RequestModifier) error
	V2QueueJobResult(ctx context.Context, region string, jobRunID string, result sdk.V2WorkflowRunJobResult) error
	V2QueueJobRunResultGet(ctx context.Context, regionName string, jobRunID string, runResultID string) (*sdk.V2WorkflowRunResult, error)
	V2QueueJobRunResultsGet(ctx context.Context, regionName string, jobRunID string) ([]sdk.V2WorkflowRunResult, error)
	V2QueueJobRunResultCreate(ctx context.Context, regionName string, jobRunID string, result *sdk.V2WorkflowRunResult) error
	V2QueueJobRunResultUpdate(ctx context.Context, regionName string, jobRunID string, result *sdk.V2WorkflowRunResult) error
	V2QueuePushRunInfo(ctx context.Context, regionName string, jobRunID string, msg sdk.V2WorkflowRunInfo) error
	V2QueuePushJobInfo(ctx context.Context, regionName string, jobRunID string, msg sdk.V2SendJobRunInfo) error
	V2QueueWorkerTakeJob(ctx context.Context, region, runJobID string) (*sdk.V2TakeJobResponse, error)
	V2QueueJobStepUpdate(ctx context.Context, regionName string, id string, stepsStatus sdk.JobStepsStatus) error
}

// QueueClient exposes queue related functions
type QueueClient interface {
	QueueWorkflowNodeJobRun(mods ...RequestModifier) ([]sdk.WorkflowNodeJobRun, error)
	QueueCountWorkflowNodeJobRun(since *time.Time, until *time.Time, modelType string) (sdk.WorkflowNodeJobRunCount, error)
	QueuePolling(ctx context.Context, goRoutines *sdk.GoRoutines, hatcheryMetrics *sdk.HatcheryMetrics, pendingWorkerCreation *sdk.HatcheryPendingWorkerCreation, jobs chan<- sdk.WorkflowNodeJobRun, errs chan<- error, filters []sdk.WebsocketFilter, delay time.Duration, ms ...RequestModifier) error
	QueueTakeJob(ctx context.Context, job sdk.WorkflowNodeJobRun) (*sdk.WorkflowNodeJobRunData, error)
	QueueJobBook(ctx context.Context, id string) (sdk.WorkflowNodeJobRunBooked, error)
	QueueJobRelease(ctx context.Context, id string) error
	QueueJobInfo(ctx context.Context, id string) (*sdk.WorkflowNodeJobRun, error)
	QueueJobSendSpawnInfo(ctx context.Context, id string, in []sdk.SpawnInfo) error
	QueueSendUnitTests(ctx context.Context, id int64, report sdk.JUnitTestsSuites) error
	QueueSendStepResult(ctx context.Context, id int64, res sdk.StepStatus) error
	QueueSendResult(ctx context.Context, id int64, res sdk.Result) error
	QueueJobTag(ctx context.Context, jobID int64, tags []sdk.WorkflowRunTag) error
	QueueJobSetVersion(ctx context.Context, jobID int64, version sdk.WorkflowRunVersion) error
	QueueWorkerCacheLink(ctx context.Context, jobID int64, tag string) (sdk.CDNItemLinks, error)
	QueueWorkflowRunResultsAdd(ctx context.Context, jobID int64, addRequest sdk.WorkflowRunResult) error
	QueueWorkflowRunResultCheck(ctx context.Context, jobID int64, runResultCheck sdk.WorkflowRunResultCheck) (int, error)
	QueueWorkflowRunResultsRelease(ctx context.Context, permJobID int64, runResultIDs []string, to string) error
	QueueWorkflowRunResultsPromote(ctx context.Context, permJobID int64, runResultIDs []string, to string) error
}

// UserClient exposes users functions
type UserClient interface {
	UserList(ctx context.Context) ([]sdk.AuthentifiedUser, error)
	UserGet(ctx context.Context, username string) (*sdk.AuthentifiedUser, error)
	UserUpdate(ctx context.Context, username string, user *sdk.AuthentifiedUser) error
	UserGetMe(ctx context.Context) (*sdk.AuthentifiedUser, error)
	UserGetGroups(ctx context.Context, username string) (map[string][]sdk.Group, error)
	UpdateFavorite(ctx context.Context, params sdk.FavoriteParams) (interface{}, error)
	UserGetSchema(ctx context.Context) (sdk.SchemaResponse, error)
	UserGetSchemaV2(ctx context.Context, entityType string) (sdk.Schema, error)
	UserGpgKeyList(ctx context.Context, username string) ([]sdk.UserGPGKey, error)
	UserGpgKeyGet(ctx context.Context, keyID string) (sdk.UserGPGKey, error)
	UserGpgKeyDelete(ctx context.Context, username string, keyID string) error
	UserGpgKeyCreate(ctx context.Context, username string, publicKey string) (sdk.UserGPGKey, error)
}

type V2WorkerClient interface {
	V2WorkerRegister(ctx context.Context, authToken string, form sdk.WorkerRegistrationForm, region, runJobID string) (*sdk.V2Worker, error)
	V2WorkerUnregister(ctx context.Context, region, runJobID string) error
	V2WorkerRefresh(ctx context.Context, region, runJobID string) error
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

	WorkerModelv2List(ctx context.Context, projKey string, vcsIdentifier string, repoIdentifier string, filter *WorkerModelV2Filter) ([]sdk.V2WorkerModel, error)
	V2WorkerGet(ctx context.Context, name string, mods ...RequestModifier) (*sdk.V2Worker, error)
	V2WorkerList(ctx context.Context) ([]sdk.V2Worker, error)
	CDNClient
}

type CDNClient interface {
	CDNItemUpload(ctx context.Context, cdnAddr string, signature string, fs afero.Fs, path string) (time.Duration, error)
	CDNItemDownload(ctx context.Context, cdnAddr string, hash string, itemType sdk.CDNItemType, md5 string, writer io.WriteSeeker) error
	CDNItemStream(ctx context.Context, cdnAddr string, hash string, itemType sdk.CDNItemType) (io.Reader, error)
}

// HookClient exposes functions used for hooks services
type HookClient interface {
	PollVCSEvents(uuid string, workflowID int64, vcsServer string, timestamp int64) (events sdk.RepositoryEvents, interval time.Duration, err error)
	VCSGerritConfiguration() (map[string]sdk.VCSGerritConfiguration, error)

	HookRepositoriesList(ctx context.Context, vcsServer, repoName string) ([]sdk.ProjectRepository, error)
	ListWorkflowToTrigger(ctx context.Context, req sdk.HookListWorkflowRequest) ([]sdk.V2WorkflowHook, error)
	RetrieveHookEventSigningKey(ctx context.Context, req sdk.HookRetrieveSignKeyRequest) (sdk.Operation, error)
	RetrieveHookEventSigningKeyOperation(ctx context.Context, operationUUID string) (sdk.Operation, error)
	RetrieveHookEventUser(ctx context.Context, req sdk.HookRetrieveUserRequest) (sdk.HookRetrieveUserResponse, error)
}

// ServiceClient exposes functions used for services
type ServiceClient interface {
	ServiceConfigurationGet(context.Context, string) ([]sdk.ServiceConfiguration, error)
}

type WorkflowV2Client interface {
	WorkflowV2RunFromHook(ctx context.Context, projectKey, vcsIdentifier, repoIdentifier, wkfName string, runRequest sdk.V2WorkflowRunHookRequest, mods ...RequestModifier) (*sdk.V2WorkflowRun, error)
	WorkflowV2Run(ctx context.Context, projectKey, vcsIdentifier, repoIdentifier, wkfName string, payload sdk.V2WorkflowRunManualRequest, mods ...RequestModifier) (*sdk.HookRepositoryEvent, error)
	WorkflowV2Restart(ctx context.Context, projectKey, runIdentifier string, mods ...RequestModifier) (*sdk.V2WorkflowRun, error)
	WorkflowV2JobStart(ctx context.Context, projectKey, runIdentifier, jobIdentifier string, payload map[string]interface{}, mods ...RequestModifier) (*sdk.V2WorkflowRun, error)
	WorkflowV2RunSearchAllProjects(ctx context.Context, offset, limit int64, mods ...RequestModifier) ([]sdk.V2WorkflowRun, error)
	WorkflowV2RunSearch(ctx context.Context, projectKey string, mods ...RequestModifier) ([]sdk.V2WorkflowRun, error)
	WorkflowV2RunInfoList(ctx context.Context, projectKey, runIdentifier string, mods ...RequestModifier) ([]sdk.V2WorkflowRunInfo, error)
	WorkflowV2RunStatus(ctx context.Context, projectKey, runIdentifier string) (*sdk.V2WorkflowRun, error)
	WorkflowV2RunJobs(ctx context.Context, projKey, runIdentifier string) ([]sdk.V2WorkflowRunJob, error)
	WorkflowV2RunJob(ctx context.Context, projKey, runIdentifier, jobIdentifier string) (*sdk.V2WorkflowRunJob, error)
	WorkflowV2RunJobInfoList(ctx context.Context, projKey, runIdentifier, jobIdentifier string) ([]sdk.V2WorkflowRunJobInfo, error)
	WorkflowV2RunJobLogLinks(ctx context.Context, projKey, runIdentifier, jobIdentifier string) (sdk.CDNLogLinks, error)
	WorkflowV2Stop(ctx context.Context, projKey, runIdentifier string) error
	WorkflowV2StopJob(ctx context.Context, projKey, runIdentifier, jobIdentifier string) error
}

// WorkflowClient exposes workflows functions
type WorkflowClient interface {
	WorkflowSearch(opts ...RequestModifier) ([]sdk.Workflow, error)
	WorkflowList(projectKey string, opts ...RequestModifier) ([]sdk.Workflow, error)
	WorkflowGet(projectKey, name string, opts ...RequestModifier) (*sdk.Workflow, error)
	WorkflowUpdate(projectKey, name string, wf *sdk.Workflow) error
	WorkflowDelete(projectKey string, workflowName string, opts ...RequestModifier) error
	WorkflowLabelAdd(projectKey, name, labelName string) error
	WorkflowLabelDelete(projectKey, name string, labelID int64) error
	WorkflowGroupAdd(projectKey, name, groupName string, permission int) error
	WorkflowGroupDelete(projectKey, name, groupName string) error
	WorkflowRunGet(projectKey string, workflowName string, number int64) (*sdk.WorkflowRun, error)
	WorkflowRunsDeleteByBranch(projectKey string, workflowName string, branch string) error
	WorkflowRunSearch(projectKey string, offset, limit int64, filter ...Filter) ([]sdk.WorkflowRun, error)
	WorkflowRunList(projectKey string, workflowName string, offset, limit int64) ([]sdk.WorkflowRun, error)
	WorkflowRunDelete(projectKey string, workflowName string, runNumber int64) error
	WorkflowRunArtifactsLinks(projectKey string, name string, number int64) (sdk.CDNItemLinks, error)
	WorkflowRunResultsList(ctx context.Context, projectKey string, name string, number int64) ([]sdk.WorkflowRunResult, error)
	WorkflowRunFromHook(projectKey string, workflowName string, hook sdk.WorkflowNodeRunHookEvent) (*sdk.WorkflowRun, error)
	WorkflowRunFromManual(projectKey string, workflowName string, manual sdk.WorkflowNodeRunManual, number, fromNodeID int64) (*sdk.WorkflowRun, error)
	WorkflowRunNumberGet(projectKey string, workflowName string) (*sdk.WorkflowRunNumber, error)
	WorkflowRunNumberSet(projectKey string, workflowName string, number int64) error
	WorkflowStop(projectKey string, workflowName string, number int64) (*sdk.WorkflowRun, error)
	WorkflowNodeStop(projectKey string, workflowName string, number, fromNodeID int64) (*sdk.WorkflowNodeRun, error)
	WorkflowNodeRun(projectKey string, name string, number int64, nodeRunID int64) (*sdk.WorkflowNodeRun, error)
	WorkflowNodeRunJobStepLinks(ctx context.Context, projectKey string, workflowName string, nodeRunID, job int64) (*sdk.CDNLogLinks, error)
	WorkflowNodeRunJobStepLink(ctx context.Context, projectKey string, workflowName string, nodeRunID, job int64, step int64) (*sdk.CDNLogLink, error)
	WorkflowNodeRunJobServiceLink(ctx context.Context, projectKey string, workflowName string, nodeRunID, job int64, serviceName string) (*sdk.CDNLogLink, error)
	WorkflowAccess(ctx context.Context, projectKey string, workflowID int64, sessionID string, itemType sdk.CDNItemType) error
	WorkflowLogDownload(ctx context.Context, link sdk.CDNLogLink) ([]byte, error)
	WorkflowNodeRunRelease(projectKey string, workflowName string, runNumber int64, nodeRunID int64, release sdk.WorkflowNodeRunRelease) error
	WorkflowAllHooksList() ([]sdk.NodeHook, error)
	WorkflowAllHooksExecutions() ([]string, error)
	WorkflowTransformAsCode(projectKey, workflowName, branch, message string) (*sdk.Operation, error)
}

type WorkflowV3Client interface {
	WorkflowV3Get(projectKey string, workflowName string, opts ...RequestModifier) ([]byte, error)
}

// MonitoringClient exposes monitoring functions
type MonitoringClient interface {
	MonStatus() (*sdk.MonitoringStatus, error)
	MonVersion() (*sdk.Version, error)
	MonDBMigrate() ([]sdk.MonDBMigrate, error)
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
//
//go:generate mockgen -source=interface.go -destination=mock_cdsclient/interface_mock.go Interface
type Interface interface {
	Raw
	AuthClient
	ActionClient
	Admin
	APIURL() string
	CDNURL() (string, error)
	ApplicationClient
	ConfigUser() (sdk.ConfigUser, error)
	ConfigCDN() (sdk.CDNConfig, error)
	ConfigVCSGPGKeys() (map[string][]sdk.Key, error)
	DownloadClient
	EnvironmentClient
	EventsClient
	ExportImportInterface
	FeatureEnabled(name sdk.FeatureName, params map[string]string) (sdk.FeatureEnabledResponse, error)
	GroupClient
	GRPCPluginsClient
	GRPCPluginsV2Client
	HatcheryClient
	MaintenanceClient
	PipelineClient
	IntegrationClient
	ProjectClient
	ProjectClientV2
	RBACClient
	OrganizationClient
	RegionClient
	QueueClient
	Navbar() ([]sdk.NavbarProjectData, error)
	Requirements() ([]sdk.Requirement, error)
	RepositoriesManagerInterface
	ServiceClient
	ServiceHeartbeat(*sdk.MonitoringStatus) error
	UserClient
	WorkerClient
	WorkflowClient
	WorkflowV2Client
	WorkflowV3Client
	MonitoringClient
	HookClient
	Version() (*sdk.Version, error)
	TemplateClient
	WebsocketClient
	V2QueueClient
	EntityLint(ctx context.Context, entityType string, data interface{}) (*sdk.EntityCheckResponse, error)
}

type V2WorkerInterface interface {
	V2WorkerClient
	V2QueueClient
	GRPCPluginsClient
	ProjectIntegrationGet(projectKey string, integrationName string, clearPassword bool) (sdk.ProjectIntegration, error)
	ProjectIntegrationWorkerHookGet(projectKey string, integrationName string) (*sdk.WorkerHookProjectIntegrationModel, error)
}

type WorkerInterface interface {
	GRPCPluginsClient
	ProjectIntegrationGet(projectKey string, integrationName string, clearPassword bool) (sdk.ProjectIntegration, error)
	ProjectIntegrationWorkerHookGet(projectKey string, integrationName string) (*sdk.WorkerHookProjectIntegrationModel, error)
	QueueClient
	Requirements() ([]sdk.Requirement, error)
	ServiceClient
	WorkerClient
	WorkflowRunGet(projectKey string, workflowName string, number int64) (*sdk.WorkflowRun, error)
	WorkflowRunList(projectKey string, workflowName string, offset, limit int64) ([]sdk.WorkflowRun, error)
	WorkflowNodeRunRelease(projectKey string, workflowName string, runNumber int64, nodeRunID int64, release sdk.WorkflowNodeRunRelease) error
	WorkflowRunArtifactsLinks(projectKey string, name string, number int64) (sdk.CDNItemLinks, error)
	WorkflowRunResultsList(ctx context.Context, projectKey string, name string, number int64) ([]sdk.WorkflowRunResult, error)
}

// Raw is a low-level interface exposing HTTP functions
type Raw interface {
	PostJSON(ctx context.Context, path string, in interface{}, out interface{}, mods ...RequestModifier) (int, error)
	PutJSON(ctx context.Context, path string, in interface{}, out interface{}, mods ...RequestModifier) (int, error)
	GetJSON(ctx context.Context, path string, out interface{}, mods ...RequestModifier) (int, error)
	DeleteJSON(ctx context.Context, path string, out interface{}, mods ...RequestModifier) (int, error)
	RequestJSON(ctx context.Context, method, path string, in interface{}, out interface{}, mods ...RequestModifier) ([]byte, http.Header, int, error)
	Request(ctx context.Context, method string, path string, body io.Reader, mods ...RequestModifier) ([]byte, http.Header, int, error)
	Stream(ctx context.Context, httpClient HTTPClient, method string, path string, body io.Reader, mods ...RequestModifier) (io.ReadCloser, http.Header, int, error)
	HTTPClient() *http.Client
	HTTPNoTimeoutClient() *http.Client
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

type GRPCPluginsV2Client interface {
	PluginImport(*sdk.GRPCPlugin, ...RequestModifier) error
}

/*
	 ProviderClient exposes allowed methods for providers
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

// WithDeepPipelines allows to get pipelines details on a workflow.
func WithDeepPipelines() RequestModifier {
	return func(r *http.Request) {
		q := r.URL.Query()
		q.Set("withDeepPipelines", "true")
		r.URL.RawQuery = q.Encode()
	}
}

// Full allows to get job details on a workflow v3.
func Full() RequestModifier {
	return func(r *http.Request) {
		q := r.URL.Query()
		q.Set("full", "true")
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

func Status(status ...string) RequestModifier {
	return func(r *http.Request) {
		if len(status) > 0 {
			q := r.URL.Query()
			for _, s := range status {
				q.Add("status", s)
			}
			r.URL.RawQuery = q.Encode()
		}
	}
}

func Region(regions ...string) RequestModifier {
	return func(r *http.Request) {
		if len(regions) > 0 {
			q := r.URL.Query()
			for _, r := range regions {
				q.Add("region", r)
			}
			r.URL.RawQuery = q.Encode()
		}
	}
}

func ModelType(modelType string) RequestModifier {
	return func(r *http.Request) {
		q := r.URL.Query()
		q.Set("modelType", modelType)
		r.URL.RawQuery = q.Encode()
	}
}

func Workflows(ws ...string) RequestModifier {
	return func(r *http.Request) {
		q := r.URL.Query()
		for _, w := range ws {
			q.Add("workflow", w)
		}
		r.URL.RawQuery = q.Encode()
	}
}

// AuthClient is the interface for authentication management.
type AuthClient interface {
	AuthDriverList() (sdk.AuthDriverResponse, error)
	AuthConsumerSignin(sdk.AuthConsumerType, interface{}) (sdk.AuthConsumerSigninResponse, error)
	AuthConsumerHatcherySigninV2(request interface{}) (sdk.AuthConsumerHatcherySigninResponse, error)
	AuthConsumerLocalAskResetPassword(sdk.AuthConsumerSigninRequest) error
	AuthConsumerLocalResetPassword(token, newPassword string) (sdk.AuthConsumerSigninResponse, error)
	AuthConsumerLocalSignup(sdk.AuthConsumerSigninRequest) error
	AuthConsumerLocalSignupVerify(token, initToken string) (sdk.AuthConsumerSigninResponse, error)
	AuthConsumerSignout() error
	AuthConsumerListByUser(username string) (sdk.AuthUserConsumers, error)
	AuthConsumerDelete(username, id string) error
	AuthConsumerRegen(username, id string, newDuration int64, overlapDuration string) (sdk.AuthConsumerCreateResponse, error)
	AuthConsumerCreateForUser(username string, request sdk.AuthUserConsumer) (sdk.AuthConsumerCreateResponse, error)
	AuthSessionListByUser(username string) (sdk.AuthSessions, error)
	AuthSessionDelete(username, id string) error
	AuthSessionGet(id string) (sdk.AuthCurrentConsumerResponse, error)
	AuthMe() (sdk.AuthCurrentConsumerResponse, error)
}

type WebsocketClient interface {
	RequestWebsocket(ctx context.Context, goRoutines *sdk.GoRoutines, path string, msgToSend <-chan json.RawMessage, msgReceived chan<- json.RawMessage, errorReceived chan<- error) error
}
