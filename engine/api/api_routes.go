package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	"github.com/ovh/cds/engine/api/observability"
)

type HandlerScope []string

func Scope(s ...string) HandlerScope {
	return HandlerScope(s)
}

var (
	ScopeNone         = func() HandlerScope { return nil }
	scopeUser         = "User"
	scopeAccessToken  = "AccessToken"
	scopeAction       = "Action"
	scopeAdmin        = "Admin"
	scopeGroup        = "Group"
	scopeTemplate     = "Template"
	scopeProject      = "Project"
	scopeRun          = "Run"
	scopeRunExecution = "RunExecution"
	scopeHooks        = "hooks"
	scopeWorker       = "worker"
	scopeWorkerModel  = "workerModel"
	scopeHatchery     = "hatchery"
)

// InitRouter initializes the router and all the routes
func (api *API) InitRouter() {
	api.Router.URL = api.Config.URL.API
	api.Router.SetHeaderFunc = DefaultHeaders
	api.Router.Middlewares = append(api.Router.Middlewares, api.authMiddleware, api.tracingMiddleware, api.maintenanceMiddleware)
	api.Router.PostMiddlewares = append(api.Router.PostMiddlewares, api.deletePermissionMiddleware, TracingPostMiddleware)

	r := api.Router
	r.Handle("/login", ScopeNone(), r.POST(api.loginUserHandler, Auth(false)))
	r.Handle("/login/callback", ScopeNone(), r.POST(api.loginUserCallbackHandler, Auth(false)))

	log.Info("Initializing Events broker")
	// Initialize event broker
	api.eventsBroker = &eventsBroker{
		router:   api.Router,
		cache:    api.Cache,
		clients:  make(map[string]*eventsBrokerSubscribe),
		dbFunc:   api.DBConnectionFactory.GetDBMap,
		messages: make(chan sdk.Event),
	}
	api.eventsBroker.Init(context.Background(), api.PanicDump())

	// Access token
	r.Handle("/accesstoken", Scope(scopeAccessToken), r.POST(api.postNewAccessTokenHandler))
	r.Handle("/accesstoken/{id}", Scope(scopeAccessToken), r.PUT(api.putRegenAccessTokenHandler), r.DELETE(api.deleteAccessTokenHandler))
	r.Handle("/accesstoken/user/{id}", Scope(scopeAccessToken), r.GET(api.getAccessTokenByUserHandler))
	r.Handle("/accesstoken/group/{id}", Scope(scopeAccessToken), r.GET(api.getAccessTokenByGroupHandler))

	// Action
	r.Handle("/action", Scope(scopeAction), r.GET(api.getActionsHandler), r.POST(api.postActionHandler))
	r.Handle("/action/import", Scope(scopeAction), r.POST(api.importActionHandler))
	r.Handle("/action/{groupName}/{permActionName}", Scope(scopeAction), r.GET(api.getActionHandler), r.PUT(api.putActionHandler), r.DELETE(api.deleteActionHandler))
	r.Handle("/action/{groupName}/{permActionName}/usage", Scope(scopeAction), r.GET(api.getActionUsageHandler))
	r.Handle("/action/{groupName}/{permActionName}/export", Scope(scopeAction), r.GET(api.getActionExportHandler))
	r.Handle("/action/{groupName}/{permActionName}/audit", Scope(scopeAction), r.GET(api.getActionAuditHandler))
	r.Handle("/action/{groupName}/{permActionName}/audit/{auditID}/rollback", Scope(scopeAction), r.POST(api.postActionAuditRollbackHandler))
	r.Handle("/action/requirement", Scope(scopeAction), r.GET(api.getActionsRequirements, Auth(false))) // FIXME add auth used by hatcheries
	r.Handle("/project/{permProjectKey}/action", Scope(scopeProject), r.GET(api.getActionsForProjectHandler))
	r.Handle("/group/{groupID}/action", Scope(scopeGroup), r.GET(api.getActionsForGroupHandler))
	r.Handle("/actionBuiltin", ScopeNone(), r.GET(api.getActionsBuiltinHandler))
	r.Handle("/actionBuiltin/{permActionBuiltinName}", ScopeNone(), r.GET(api.getActionBuiltinHandler))
	r.Handle("/actionBuiltin/{permActionBuiltinName}/usage", Scope(scopeAdmin), r.GET(api.getActionBuiltinUsageHandler))

	// Admin
	r.Handle("/admin/maintenance", Scope(scopeAdmin), r.POST(api.postMaintenanceHandler, NeedAdmin(true)))
	r.Handle("/admin/warning", Scope(scopeAdmin), r.DELETE(api.adminTruncateWarningsHandler, NeedAdmin(true)))
	r.Handle("/admin/cds/migration", Scope(scopeAdmin), r.GET(api.getAdminMigrationsHandler, NeedAdmin(true)))
	r.Handle("/admin/cds/migration/{id}/cancel", Scope(scopeAdmin), r.POST(api.postAdminMigrationCancelHandler, NeedAdmin(true)))
	r.Handle("/admin/cds/migration/{id}/todo", Scope(scopeAdmin), r.POST(api.postAdminMigrationTodoHandler, NeedAdmin(true)))
	r.Handle("/admin/database/migration/delete/{id}", Scope(scopeAdmin), r.DELETE(api.deleteDatabaseMigrationHandler, NeedAdmin(true)))
	r.Handle("/admin/database/migration/unlock/{id}", Scope(scopeAdmin), r.POST(api.postDatabaseMigrationUnlockedHandler, NeedAdmin(true)))
	r.Handle("/admin/database/migration", Scope(scopeAdmin), r.GET(api.getDatabaseMigrationHandler, NeedAdmin(true)))
	r.Handle("/admin/debug", Scope(scopeAdmin), r.GET(api.getProfileIndexHandler, Auth(false)))
	r.Handle("/admin/debug/trace", Scope(scopeAdmin), r.POST(api.getTraceHandler, NeedAdmin(true)), r.GET(api.getTraceHandler, NeedAdmin(true)))
	r.Handle("/admin/debug/cpu", Scope(scopeAdmin), r.POST(api.getCPUProfileHandler, NeedAdmin(true)), r.GET(api.getCPUProfileHandler, NeedAdmin(true)))
	r.Handle("/admin/debug/{name}", Scope(scopeAdmin), r.POST(api.getProfileHandler, NeedAdmin(true)), r.GET(api.getProfileHandler, NeedAdmin(true)))
	r.Handle("/admin/plugin", Scope(scopeAdmin), r.POST(api.postGRPCluginHandler, NeedAdmin(true)), r.GET(api.getAllGRPCluginHandler, NeedAdmin(true)))
	r.Handle("/admin/plugin/{name}", Scope(scopeAdmin), r.GET(api.getGRPCluginHandler, NeedAdmin(true)), r.PUT(api.putGRPCluginHandler, NeedAdmin(true)), r.DELETE(api.deleteGRPCluginHandler, NeedAdmin(true)))
	r.Handle("/admin/plugin/{name}/binary", Scope(scopeAdmin), r.POST(api.postGRPCluginBinaryHandler, NeedAdmin(true)))
	r.Handle("/admin/plugin/{name}/binary/{os}/{arch}", Scope(scopeAdmin), r.GET(api.getGRPCluginBinaryHandler, Auth(false)), r.DELETE(api.deleteGRPCluginBinaryHandler, NeedAdmin(true)))
	r.Handle("/admin/plugin/{name}/binary/{os}/{arch}/infos", Scope(scopeAdmin), r.GET(api.getGRPCluginBinaryInfosHandler))

	// Admin service
	r.Handle("/admin/service/{name}", Scope(scopeAdmin), r.GET(api.getAdminServiceHandler, NeedAdmin(true)), r.DELETE(api.deleteAdminServiceHandler, NeedAdmin(true)))
	r.Handle("/admin/services", Scope(scopeAdmin), r.GET(api.getAdminServicesHandler, NeedAdmin(true)))
	r.Handle("/admin/services/call", Scope(scopeAdmin), r.GET(api.getAdminServiceCallHandler, NeedAdmin(true)), r.POST(api.postAdminServiceCallHandler, NeedAdmin(true)), r.PUT(api.putAdminServiceCallHandler, NeedAdmin(true)), r.DELETE(api.deleteAdminServiceCallHandler, NeedAdmin(true)))

	// Download file
	r.Handle("/download", ScopeNone(), r.GET(api.downloadsHandler))
	r.Handle("/download/{name}/{os}/{arch}", ScopeNone(), r.GET(api.downloadHandler, Auth(false)))

	// Group
	r.Handle("/group", Scope(scopeGroup), r.GET(api.getGroupsHandler), r.POST(api.addGroupHandler))
	r.Handle("/group/{permGroupName}", Scope(scopeGroup), r.GET(api.getGroupHandler), r.PUT(api.updateGroupHandler), r.DELETE(api.deleteGroupHandler))
	r.Handle("/group/{permGroupName}/user", Scope(scopeGroup), r.POST(api.addUserInGroupHandler))
	r.Handle("/group/{permGroupName}/user/{user}", Scope(scopeGroup), r.DELETE(api.removeUserFromGroupHandler))
	r.Handle("/group/{permGroupName}/user/{user}/admin", Scope(scopeGroup), r.POST(api.setUserGroupAdminHandler), r.DELETE(api.removeUserGroupAdminHandler))
	r.Handle("/group/{permGroupName}/token", Scope(scopeGroup), r.GET(api.getGroupTokenListHandler), r.POST(api.generateTokenHandler))
	r.Handle("/group/{permGroupName}/token/{tokenid}", Scope(scopeGroup), r.DELETE(api.deleteTokenHandler))

	// Hatchery
	r.Handle("/hatchery/count/{workflowNodeRunID}", Scope(scopeHatchery), r.GET(api.hatcheryCountHandler))

	// Hooks
	r.Handle("/hook/{uuid}/workflow/{workflowID}/vcsevent/{vcsServer}", Scope(scopeRun), r.GET(api.getHookPollingVCSEvents))

	// Integration
	r.Handle("/integration/models", ScopeNone(), r.GET(api.getIntegrationModelsHandler), r.POST(api.postIntegrationModelHandler, NeedAdmin(true)))
	r.Handle("/integration/models/{name}", ScopeNone(), r.GET(api.getIntegrationModelHandler), r.PUT(api.putIntegrationModelHandler, NeedAdmin(true)), r.DELETE(api.deleteIntegrationModelHandler, NeedAdmin(true)))

	// Broadcast
	r.Handle("/broadcast", ScopeNone(), r.POST(api.addBroadcastHandler, NeedAdmin(true)), r.GET(api.getBroadcastsHandler))
	r.Handle("/broadcast/{id}", ScopeNone(), r.GET(api.getBroadcastHandler), r.PUT(api.updateBroadcastHandler, NeedAdmin(true)), r.DELETE(api.deleteBroadcastHandler, NeedAdmin(true)))
	r.Handle("/broadcast/{id}/mark", Scope(scopeProject), r.POST(api.postMarkAsReadBroadcastHandler))

	// Overall health
	r.Handle("/mon/status", ScopeNone(), r.GET(api.statusHandler, Auth(false)))
	r.Handle("/mon/smtp/ping", ScopeNone(), r.GET(api.smtpPingHandler, Auth(true)))
	r.Handle("/mon/version", ScopeNone(), r.GET(VersionHandler, Auth(false)))
	r.Handle("/mon/db/migrate", ScopeNone(), r.GET(api.getMonDBStatusMigrateHandler, NeedAdmin(true)))
	r.Handle("/mon/metrics", ScopeNone(), r.GET(observability.StatsHandler, Auth(false)))
	r.Handle("/mon/errors/{uuid}", ScopeNone(), r.GET(api.getErrorHandler, NeedAdmin(true)))
	r.Handle("/mon/panic/{uuid}", ScopeNone(), r.GET(api.getPanicDumpHandler, Auth(false)))

	r.Handle("/ui/navbar", ScopeNone(), r.GET(api.getNavbarHandler))
	r.Handle("/ui/project/{permProjectKey}/application/{applicationName}/overview", ScopeNone(), r.GET(api.getApplicationOverviewHandler))

	// Import As Code
	r.Handle("/import/{permProjectKey}", Scope(scopeProject), r.POST(api.postImportAsCodeHandler))
	r.Handle("/import/{permProjectKey}/{uuid}", Scope(scopeProject), r.GET(api.getImportAsCodeHandler))
	r.Handle("/import/{permProjectKey}/{uuid}/perform", Scope(scopeProject), r.POST(api.postPerformImportAsCodeHandler))

	// Bookmarks
	r.Handle("/bookmarks", ScopeNone(), r.GET(api.getBookmarksHandler))

	// Project
	r.Handle("/project", Scope(scopeProject), r.GET(api.getProjectsHandler, AllowProvider(true), EnableTracing()), r.POST(api.addProjectHandler))
	r.Handle("/project/{permProjectKey}", Scope(scopeProject), r.GET(api.getProjectHandler), r.PUT(api.updateProjectHandler), r.DELETE(api.deleteProjectHandler))
	r.Handle("/project/{permProjectKey}/labels", Scope(scopeProject), r.PUT(api.putProjectLabelsHandler))
	r.Handle("/project/{permProjectKey}/group", Scope(scopeProject), r.POST(api.addGroupInProjectHandler))
	r.Handle("/project/{permProjectKey}/group/import", Scope(scopeProject), r.POST(api.importGroupsInProjectHandler, DEPRECATED))
	r.Handle("/project/{permProjectKey}/group/{group}", Scope(scopeProject), r.PUT(api.updateGroupRoleOnProjectHandler), r.DELETE(api.deleteGroupFromProjectHandler))
	r.Handle("/project/{permProjectKey}/variable", Scope(scopeProject), r.GET(api.getVariablesInProjectHandler))
	r.Handle("/project/{permProjectKey}/encrypt", Scope(scopeProject), r.POST(api.postEncryptVariableHandler))
	r.Handle("/project/{key}/variable/audit", Scope(scopeProject), r.GET(api.getVariablesAuditInProjectnHandler))
	r.Handle("/project/{permProjectKey}/variable/{name}", Scope(scopeProject), r.GET(api.getVariableInProjectHandler), r.POST(api.addVariableInProjectHandler), r.PUT(api.updateVariableInProjectHandler), r.DELETE(api.deleteVariableFromProjectHandler))
	r.Handle("/project/{permProjectKey}/variable/{name}/audit", Scope(scopeProject), r.GET(api.getVariableAuditInProjectHandler))
	r.Handle("/project/{permProjectKey}/applications", Scope(scopeProject), r.GET(api.getApplicationsHandler, AllowProvider(true)), r.POST(api.addApplicationHandler))
	r.Handle("/project/{permProjectKey}/integrations", Scope(scopeProject), r.GET(api.getProjectIntegrationsHandler), r.POST(api.postProjectIntegrationHandler))
	r.Handle("/project/{permProjectKey}/integrations/{integrationName}", Scope(scopeProject), r.GET(api.getProjectIntegrationHandler, AllowServices(true)), r.PUT(api.putProjectIntegrationHandler), r.DELETE(api.deleteProjectIntegrationHandler))
	r.Handle("/project/{permProjectKey}/notifications", Scope(scopeProject), r.GET(api.getProjectNotificationsHandler, DEPRECATED))
	r.Handle("/project/{permProjectKey}/all/keys", Scope(scopeProject), r.GET(api.getAllKeysProjectHandler))
	r.Handle("/project/{permProjectKey}/keys", Scope(scopeProject), r.GET(api.getKeysInProjectHandler), r.POST(api.addKeyInProjectHandler))
	r.Handle("/project/{permProjectKey}/keys/{name}", Scope(scopeProject), r.DELETE(api.deleteKeyInProjectHandler))
	// Import Application
	r.Handle("/project/{permProjectKey}/import/application", Scope(scopeProject), r.POST(api.postApplicationImportHandler))
	// Export Application
	r.Handle("/project/{permProjectKey}/export/application/{applicationName}", Scope(scopeProject), r.GET(api.getApplicationExportHandler))

	r.Handle("/warning/{permProjectKey}", Scope(scopeProject), r.GET(api.getWarningsHandler))
	r.Handle("/warning/{permProjectKey}/{hash}", Scope(scopeProject), r.PUT(api.putWarningsHandler))

	// Application
	r.Handle("/project/{permProjectKey}/application/{applicationName}", Scope(scopeProject), r.GET(api.getApplicationHandler), r.PUT(api.updateApplicationHandler), r.DELETE(api.deleteApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/metrics/{metricName}", Scope(scopeProject), r.GET(api.getApplicationMetricHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/keys", Scope(scopeProject), r.GET(api.getKeysInApplicationHandler), r.POST(api.addKeyInApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/keys/{name}", Scope(scopeProject), r.DELETE(api.deleteKeyInApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/vcsinfos", Scope(scopeProject), r.GET(api.getApplicationVCSInfosHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/clone", Scope(scopeProject), r.POST(api.cloneApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/variable", Scope(scopeProject), r.GET(api.getVariablesInApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/variable/audit", Scope(scopeProject), r.GET(api.getVariablesAuditInApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/variable/{name}", Scope(scopeProject), r.GET(api.getVariableInApplicationHandler), r.POST(api.addVariableInApplicationHandler), r.PUT(api.updateVariableInApplicationHandler), r.DELETE(api.deleteVariableFromApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/variable/{name}/audit", Scope(scopeProject), r.GET(api.getVariableAuditInApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/vulnerability/{id}", Scope(scopeProject), r.POST(api.postVulnerabilityHandler))
	// Application deployment
	r.Handle("/project/{permProjectKey}/application/{applicationName}/deployment/config/{integration}", Scope(scopeProject), r.POST(api.postApplicationDeploymentStrategyConfigHandler, AllowProvider(true)), r.GET(api.getApplicationDeploymentStrategyConfigHandler), r.DELETE(api.deleteApplicationDeploymentStrategyConfigHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/deployment/config", Scope(scopeProject), r.GET(api.getApplicationDeploymentStrategiesConfigHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/metadata/{metadata}", Scope(scopeProject), r.POST(api.postApplicationMetadataHandler, AllowProvider(true)))

	// Pipeline
	r.Handle("/project/{permProjectKey}/pipeline", Scope(scopeProject), r.GET(api.getPipelinesHandler), r.POST(api.addPipelineHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/parameter", Scope(scopeProject), r.GET(api.getParametersInPipelineHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/parameter/{name}", Scope(scopeProject), r.POST(api.addParameterInPipelineHandler), r.PUT(api.updateParameterInPipelineHandler), r.DELETE(api.deleteParameterFromPipelineHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}", Scope(scopeProject), r.GET(api.getPipelineHandler), r.PUT(api.updatePipelineHandler), r.DELETE(api.deletePipelineHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/rollback/{auditID}", Scope(scopeProject), r.POST(api.postPipelineRollbackHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/audits", Scope(scopeProject), r.GET(api.getPipelineAuditHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/stage", Scope(scopeProject), r.POST(api.addStageHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/stage/move", Scope(scopeProject), r.POST(api.moveStageHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/stage/{stageID}", Scope(scopeProject), r.GET(api.getStageHandler), r.PUT(api.updateStageHandler), r.DELETE(api.deleteStageHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/stage/{stageID}/job", Scope(scopeProject), r.POST(api.addJobToStageHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/stage/{stageID}/job/{jobID}", Scope(scopeProject), r.PUT(api.updateJobHandler), r.DELETE(api.deleteJobHandler))

	// Preview pipeline
	r.Handle("/project/{permProjectKey}/preview/pipeline", Scope(scopeProject), r.POST(api.postPipelinePreviewHandler))
	// Import pipeline
	r.Handle("/project/{permProjectKey}/import/pipeline", Scope(scopeProject), r.POST(api.importPipelineHandler))
	// Import pipeline (ONLY USE FOR UI)
	r.Handle("/project/{permProjectKey}/import/pipeline/{pipelineKey}", Scope(scopeProject), r.PUT(api.putImportPipelineHandler))
	// Export pipeline
	r.Handle("/project/{permProjectKey}/export/pipeline/{pipelineKey}", Scope(scopeProject), r.GET(api.getPipelineExportHandler))

	// Workflows
	r.Handle("/workflow/artifact/{hash}", ScopeNone(), r.GET(api.downloadworkflowArtifactDirectHandler, Auth(false)))

	r.Handle("/project/{permProjectKey}/workflows", Scope(scopeProject), r.POST(api.postWorkflowHandler, EnableTracing()), r.GET(api.getWorkflowsHandler, AllowProvider(true), EnableTracing()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}", Scope(scopeProject), r.GET(api.getWorkflowHandler, AllowProvider(true), EnableTracing()), r.PUT(api.putWorkflowHandler, EnableTracing()), r.DELETE(api.deleteWorkflowHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/icon", Scope(scopeProject), r.PUT(api.putWorkflowIconHandler), r.DELETE(api.deleteWorkflowIconHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/ascode/{uuid}", Scope(scopeProject), r.GET(api.getWorkflowAsCodeHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/ascode", Scope(scopeProject), r.POST(api.postWorkflowAsCodeHandler, EnableTracing()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/ascode/resync/pr", Scope(scopeProject), r.POST(api.postResyncPRWorkflowAsCodeHandler, EnableTracing()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/label", Scope(scopeProject), r.POST(api.postWorkflowLabelHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/label/{labelID}", Scope(scopeProject), r.DELETE(api.deleteWorkflowLabelHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/rollback/{auditID}", Scope(scopeProject), r.POST(api.postWorkflowRollbackHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/groups", Scope(scopeProject), r.POST(api.postWorkflowGroupHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/groups/{groupName}", Scope(scopeProject), r.PUT(api.putWorkflowGroupHandler), r.DELETE(api.deleteWorkflowGroupHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/hooks/{uuid}", Scope(scopeProject), r.GET(api.getWorkflowHookHandler))
	r.Handle("/project/{key}/workflow/{permWorkflowName}/node/{nodeID}/hook/model", Scope(scopeProject), r.GET(api.getWorkflowHookModelsHandler))
	r.Handle("/project/{key}/workflow/{permWorkflowName}/node/{nodeID}/outgoinghook/model", Scope(scopeProject), r.GET(api.getWorkflowOutgoingHookModelsHandler))

	// Outgoing hook model
	r.Handle("/workflow/outgoinghook/model", ScopeNone(), r.GET(api.getWorkflowOutgoingHookModelsHandler))

	// Preview workflows
	r.Handle("/project/{permProjectKey}/preview/workflows", Scope(scopeProject), r.POST(api.postWorkflowPreviewHandler))
	// Import workflows
	r.Handle("/project/{permProjectKey}/import/workflows", Scope(scopeProject), r.POST(api.postWorkflowImportHandler))
	// Import workflows (ONLY USE FOR UI EDIT AS CODE)
	r.Handle("/project/{key}/import/workflows/{permWorkflowName}", Scope(scopeProject), r.PUT(api.putWorkflowImportHandler))
	// Export workflows
	r.Handle("/project/{key}/export/workflows/{permWorkflowName}", Scope(scopeProject), r.GET(api.getWorkflowExportHandler))
	// Pull workflows
	r.Handle("/project/{key}/pull/workflows/{permWorkflowName}", Scope(scopeProject), r.GET(api.getWorkflowPullHandler))
	// Push workflows
	r.Handle("/project/{permProjectKey}/push/workflows", Scope(scopeProject), r.POST(api.postWorkflowPushHandler))

	// Workflows run
	r.Handle("/project/{permProjectKey}/runs", Scope(scopeProject), r.GET(api.getWorkflowAllRunsHandler, EnableTracing()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/artifact/{artifactId}", Scope(scopeRun), r.GET(api.getDownloadArtifactHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs", Scope(scopeRun), r.GET(api.getWorkflowRunsHandler, EnableTracing()), r.POSTEXECUTE(api.postWorkflowRunHandler, AllowServices(true), EnableTracing()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/branch/{branch}", Scope(scopeRun), r.DELETE(api.deleteWorkflowRunsBranchHandler, NeedService()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/latest", Scope(scopeRun), r.GET(api.getLatestWorkflowRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/tags", Scope(scopeRun), r.GET(api.getWorkflowRunTagsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/num", Scope(scopeRun), r.GET(api.getWorkflowRunNumHandler), r.POST(api.postWorkflowRunNumHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}", Scope(scopeRun), r.GET(api.getWorkflowRunHandler, AllowServices(true)))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/stop", Scope(scopeRun), r.POSTEXECUTE(api.stopWorkflowRunHandler, EnableTracing()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/vcs/resync", Scope(scopeRun), r.POSTEXECUTE(api.postResyncVCSWorkflowRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/resync", Scope(scopeRun), r.POST(api.resyncWorkflowRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/artifacts", Scope(scopeRun), r.GET(api.getWorkflowRunArtifactsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}", Scope(scopeRun), r.GET(api.getWorkflowNodeRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/stop", Scope(scopeRun), r.POSTEXECUTE(api.stopWorkflowNodeRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeID}/history", Scope(scopeRun), r.GET(api.getWorkflowNodeRunHistoryHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/{nodeName}/commits", Scope(scopeRun), r.GET(api.getWorkflowCommitsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/job/{runJobId}/info", Scope(scopeRun), r.GET(api.getWorkflowNodeRunJobSpawnInfosHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/job/{runJobId}/log/service", Scope(scopeRun), r.GET(api.getWorkflowNodeRunJobServiceLogsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/job/{runJobId}/step/{stepOrder}", Scope(scopeRun), r.GET(api.getWorkflowNodeRunJobStepHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/node/{nodeID}/triggers/condition", Scope(scopeRun), r.GET(api.getWorkflowTriggerConditionHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/release", Scope(scopeRun), r.POST(api.releaseApplicationWorkflowHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/hooks/{hookRunID}/callback", Scope(scopeRun), r.POST(api.postWorkflowJobHookCallbackHandler, AllowServices(true)))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/hooks/{hookRunID}/details", Scope(scopeRun), r.GET(api.getWorkflowJobHookDetailsHandler, NeedService()))

	// Environment
	r.Handle("/project/{permProjectKey}/environment", Scope(scopeProject), r.GET(api.getEnvironmentsHandler), r.POST(api.addEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/import", Scope(scopeProject), r.POST(api.importNewEnvironmentHandler, DEPRECATED))
	r.Handle("/project/{permProjectKey}/environment/import/{environmentName}", Scope(scopeProject), r.POST(api.importIntoEnvironmentHandler, DEPRECATED))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}", Scope(scopeProject), r.GET(api.getEnvironmentHandler), r.PUT(api.updateEnvironmentHandler), r.DELETE(api.deleteEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/usage", Scope(scopeProject), r.GET(api.getEnvironmentUsageHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/keys", Scope(scopeProject), r.GET(api.getKeysInEnvironmentHandler), r.POST(api.addKeyInEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/keys/{name}", Scope(scopeProject), r.DELETE(api.deleteKeyInEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/clone/{cloneName}", Scope(scopeProject), r.POST(api.cloneEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/variable", Scope(scopeProject), r.GET(api.getVariablesInEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/variable/{name}", Scope(scopeProject), r.GET(api.getVariableInEnvironmentHandler), r.POST(api.addVariableInEnvironmentHandler), r.PUT(api.updateVariableInEnvironmentHandler), r.DELETE(api.deleteVariableFromEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/variable/{name}/audit", Scope(scopeProject), r.GET(api.getVariableAuditInEnvironmentHandler))

	// Import Environment
	r.Handle("/project/{permProjectKey}/import/environment", Scope(scopeProject), r.POST(api.postEnvironmentImportHandler))
	// Export Environment
	r.Handle("/project/{permProjectKey}/export/environment/{environmentName}", Scope(scopeProject), r.GET(api.getEnvironmentExportHandler))

	// Project storage
	r.Handle("/project/{permProjectKey}/storage/{integrationName}", Scope(scopeRunExecution), r.GET(api.getArtifactsStoreHandler))
	r.Handle("/project/{permProjectKey}/storage/{integrationName}/artifact/{ref}", Scope(scopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobArtifactHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/project/{permProjectKey}/storage/{integrationName}/artifact/{ref}/url", Scope(scopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobArtifacWithTempURLHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/project/{permProjectKey}/storage/{integrationName}/artifact/{ref}/url/callback", Scope(scopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobArtifactWithTempURLCallbackHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/project/{permProjectKey}/storage/{integrationName}/staticfiles/{name}", Scope(scopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobStaticFilesHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))

	// Cache
	r.Handle("/project/{permProjectKey}/storage/{integrationName}/cache/{tag}", Scope(scopeRunExecution), r.POSTEXECUTE(api.postPushCacheHandler, NeedWorker()), r.GET(api.getPullCacheHandler, NeedWorker()))
	r.Handle("/project/{permProjectKey}/storage/{integrationName}/cache/{tag}/url", Scope(scopeRunExecution), r.POSTEXECUTE(api.postPushCacheWithTempURLHandler, NeedWorker()), r.GET(api.getPullCacheWithTempURLHandler, NeedWorker()))

	//Workflow queue
	r.Handle("/queue/workflows", Scope(scopeRun), r.GET(api.getWorkflowJobQueueHandler, EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/count", Scope(scopeRun), r.GET(api.countWorkflowJobQueueHandler, EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{id}/take", Scope(scopeRunExecution), r.POST(api.postTakeWorkflowJobHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{id}/book", Scope(scopeRunExecution), r.POST(api.postBookWorkflowJobHandler, NeedHatchery(), EnableTracing(), MaintenanceAware()), r.DELETE(api.deleteBookWorkflowJobHandler, NeedHatchery(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{id}/attempt", Scope(scopeRunExecution), r.POST(api.postIncWorkflowJobAttemptHandler, NeedHatchery(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{id}/infos", Scope(scopeRunExecution), r.GET(api.getWorkflowJobHandler, NeedWorker(), NeedHatchery(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/vulnerability", Scope(scopeRunExecution), r.POSTEXECUTE(api.postVulnerabilityReportHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{id}/spawn/infos", Scope(scopeRunExecution), r.POST(r.Asynchronous(api.postSpawnInfosWorkflowJobHandler, 1), NeedHatchery(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/result", Scope(scopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobResultHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/log", Scope(scopeRunExecution), r.POSTEXECUTE(r.Asynchronous(api.postWorkflowJobLogsHandler, 1), NeedWorker(), MaintenanceAware()))
	r.Handle("/queue/workflows/log/service", Scope(scopeRunExecution), r.POSTEXECUTE(r.Asynchronous(api.postWorkflowJobServiceLogsHandler, 1), NeedHatchery(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/coverage", Scope(scopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobCoverageResultsHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/test", Scope(scopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobTestsResultsHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/tag", Scope(scopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobTagsHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/variable", Scope(scopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobVariableHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/step", Scope(scopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobStepStatusHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))

	r.Handle("/variable/type", ScopeNone(), r.GET(api.getVariableTypeHandler))
	r.Handle("/parameter/type", ScopeNone(), r.GET(api.getParameterTypeHandler))
	r.Handle("/notification/type", ScopeNone(), r.GET(api.getUserNotificationTypeHandler))
	r.Handle("/notification/state", ScopeNone(), r.GET(api.getUserNotificationStateValueHandler))

	// RepositoriesManager
	r.Handle("/repositories_manager", Scope(scopeProject), r.GET(api.getRepositoriesManagerHandler))
	r.Handle("/repositories_manager/oauth2/callback", Scope(scopeProject), r.GET(api.repositoriesManagerOAuthCallbackHandler, Auth(false)))
	// RepositoriesManager for projects
	r.Handle("/project/{permProjectKey}/repositories_manager", Scope(scopeProject), r.GET(api.getRepositoriesManagerForProjectHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/authorize", Scope(scopeProject), r.POST(api.repositoriesManagerAuthorizeHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/authorize/callback", Scope(scopeProject), r.POST(api.repositoriesManagerAuthorizeCallbackHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/authorize/basicauth", Scope(scopeProject), r.POST(api.repositoriesManagerAuthorizeBasicHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}", Scope(scopeProject), r.DELETE(api.deleteRepositoriesManagerHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/repo", Scope(scopeProject), r.GET(api.getRepoFromRepositoriesManagerHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/repos", Scope(scopeProject), r.GET(api.getReposFromRepositoriesManagerHandler))

	// RepositoriesManager for applications
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/application/{applicationName}/attach", Scope(scopeProject), r.POST(api.attachRepositoriesManagerHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/application/{applicationName}/detach", Scope(scopeProject), r.POST(api.detachRepositoriesManagerHandler))

	// Suggest
	r.Handle("/suggest/variable/{permProjectKey}", Scope(scopeProject), r.GET(api.getVariablesHandler))

	//Requirements
	r.Handle("/requirement/types", ScopeNone(), r.GET(api.getRequirementTypesHandler))
	r.Handle("/requirement/types/{type}", ScopeNone(), r.GET(api.getRequirementTypeValuesHandler))

	// config
	r.Handle("/config/user", ScopeNone(), r.GET(api.ConfigUserHandler, Auth(false)))
	r.Handle("/config/vcs", ScopeNone(), r.GET(api.ConfigVCShandler))

	// Users
	r.Handle("/user", Scope(scopeUser), r.GET(api.getUsersHandler))
	r.Handle("/user/logged", Scope(scopeUser), r.GET(api.getUserLoggedHandler, Auth(false)))
	r.Handle("/user/me", Scope(scopeUser), r.GET(api.getUserLoggedHandler, Auth(false), DEPRECATED))
	r.Handle("/user/favorite", Scope(scopeUser), r.POST(api.postUserFavoriteHandler))
	r.Handle("/user/timeline", Scope(scopeUser), r.GET(api.getTimelineHandler))
	r.Handle("/user/timeline/filter", Scope(scopeUser), r.GET(api.getTimelineFilterHandler), r.POST(api.postTimelineFilterHandler))
	r.Handle("/user/token", Scope(scopeUser), r.GET(api.getUserTokenListHandler))
	r.Handle("/user/token/{token}", Scope(scopeUser), r.GET(api.getUserTokenHandler))
	r.Handle("/user/signup", ScopeNone(), r.POST(api.addUserHandler, Auth(false)))
	r.Handle("/user/{username}", Scope(scopeUser), r.GET(api.getUserHandler, NeedUsernameOrAdmin(true)), r.PUT(api.updateUserHandler, NeedUsernameOrAdmin(true)), r.DELETE(api.deleteUserHandler, NeedUsernameOrAdmin(true)))
	r.Handle("/user/{username}/groups", Scope(scopeUser), r.GET(api.getUserGroupsHandler, NeedUsernameOrAdmin(true)))
	r.Handle("/user/{username}/confirm/{token}", Scope(scopeUser), r.GET(api.confirmUserHandler, Auth(false)))
	r.Handle("/user/{username}/reset", Scope(scopeUser), r.POST(api.resetUserHandler, Auth(false)))

	// Workers
	r.Handle("/worker", Scope(scopeAdmin, scopeWorker, scopeHatchery), r.GET(api.getWorkersHandler), r.POST(api.registerWorkerHandler, Auth(false)))
	r.Handle("/worker/refresh", Scope(scopeWorker), r.POST(api.refreshWorkerHandler))
	r.Handle("/worker/checking", Scope(scopeWorker), r.POST(api.workerCheckingHandler))
	r.Handle("/worker/waiting", Scope(scopeWorker), r.POST(api.workerWaitingHandler))
	r.Handle("/worker/unregister", Scope(scopeWorker), r.POST(api.unregisterWorkerHandler))
	r.Handle("/worker/{id}/disable", Scope(scopeAdmin, scopeHatchery), r.POST(api.disableWorkerHandler))

	// Worker models
	r.Handle("/worker/model", Scope(scopeWorkerModel), r.POST(api.addWorkerModelHandler), r.GET(api.getWorkerModelsHandler))
	r.Handle("/worker/model/import", Scope(scopeWorkerModel), r.POST(api.postWorkerModelImportHandler))
	r.Handle("/worker/model/pattern", Scope(scopeWorkerModel), r.POST(api.postAddWorkerModelPatternHandler, NeedAdmin(true)), r.GET(api.getWorkerModelPatternsHandler))
	r.Handle("/worker/model/pattern/{type}/{name}", Scope(scopeWorkerModel), r.GET(api.getWorkerModelPatternHandler), r.PUT(api.putWorkerModelPatternHandler, NeedAdmin(true)), r.DELETE(api.deleteWorkerModelPatternHandler, NeedAdmin(true)))
	r.Handle("/worker/model/book/{permModelID}", Scope(scopeWorkerModel), r.PUT(api.bookWorkerModelHandler, NeedHatchery()))
	r.Handle("/worker/model/error/{permModelID}", Scope(scopeWorkerModel), r.PUT(api.spawnErrorWorkerModelHandler, NeedHatchery()))
	r.Handle("/worker/model/enabled", Scope(scopeWorkerModel), r.GET(api.getWorkerModelsEnabledHandler, NeedHatchery()))
	r.Handle("/worker/model/type", Scope(scopeWorkerModel), r.GET(api.getWorkerModelTypesHandler))
	r.Handle("/worker/model/communication", Scope(scopeWorkerModel), r.GET(api.getWorkerModelCommunicationsHandler))
	r.Handle("/worker/model/{permModelID}", Scope(scopeWorkerModel), r.PUT(api.updateWorkerModelHandler), r.DELETE(api.deleteWorkerModelHandler))
	r.Handle("/worker/model/{modelID}/export", Scope(scopeWorkerModel), r.GET(api.getWorkerModelExportHandler))
	r.Handle("/worker/model/{modelID}/usage", Scope(scopeWorkerModel), r.GET(api.getWorkerModelUsageHandler))
	r.Handle("/worker/model/capability/type", Scope(scopeWorkerModel), r.GET(api.getRequirementTypesHandler))

	// Workflows
	r.Handle("/workflow/hook", Scope(scopeHooks), r.GET(api.getWorkflowHooksHandler, NeedService()))
	r.Handle("/workflow/hook/model/{model}", ScopeNone(), r.GET(api.getWorkflowHookModelHandler), r.POST(api.postWorkflowHookModelHandler, NeedAdmin(true)), r.PUT(api.putWorkflowHookModelHandler, NeedAdmin(true)))

	// SSE
	r.Handle("/events", ScopeNone(), r.GET(api.eventsBroker.ServeHTTP))

	// Feature
	r.Handle("/feature/clean", ScopeNone(), r.POST(api.cleanFeatureHandler, NeedToken("X-Izanami-Token", api.Config.Features.Izanami.Token), Auth(false)))

	// Engine µServices
	r.Handle("/services/register", ScopeNone(), r.POST(api.postServiceRegisterHandler, Auth(false)))
	r.Handle("/services/{type}", ScopeNone(), r.GET(api.getExternalServiceHandler, NeedWorker()))

	// Templates
	r.Handle("/template", Scope(scopeTemplate), r.GET(api.getTemplatesHandler), r.POST(api.postTemplateHandler))
	r.Handle("/template/push", Scope(scopeTemplate), r.POST(api.postTemplatePushHandler))
	r.Handle("/template/{id}", Scope(scopeTemplate), r.GET(api.getTemplateHandler))
	r.Handle("/template/{groupName}/{templateSlug}", Scope(scopeTemplate), r.GET(api.getTemplateHandler), r.PUT(api.putTemplateHandler), r.DELETE(api.deleteTemplateHandler))
	r.Handle("/template/{groupName}/{templateSlug}/pull", Scope(scopeTemplate), r.POST(api.postTemplatePullHandler))
	r.Handle("/template/{groupName}/{templateSlug}/apply", Scope(scopeTemplate), r.POST(api.postTemplateApplyHandler))
	r.Handle("/template/{groupName}/{templateSlug}/bulk", Scope(scopeTemplate), r.POST(api.postTemplateBulkHandler))
	r.Handle("/template/{groupName}/{templateSlug}/bulk/{bulkID}", Scope(scopeTemplate), r.GET(api.getTemplateBulkHandler))
	r.Handle("/template/{groupName}/{templateSlug}/instance", Scope(scopeTemplate), r.GET(api.getTemplateInstancesHandler))
	r.Handle("/template/{groupName}/{templateSlug}/instance/{instanceID}", Scope(scopeTemplate), r.DELETE(api.deleteTemplateInstanceHandler))
	r.Handle("/template/{groupName}/{templateSlug}/audit", Scope(scopeTemplate), r.GET(api.getTemplateAuditsHandler))
	r.Handle("/template/{groupName}/{templateSlug}/usage", Scope(scopeTemplate), r.GET(api.getTemplateUsageHandler))
	r.Handle("/project/{key}/workflow/{permWorkflowName}/templateInstance", Scope(scopeProject), r.GET(api.getTemplateInstanceHandler))

	//Not Found handler
	r.Mux.NotFoundHandler = http.HandlerFunc(NotFoundHandler)
}
