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
	ScopeNone = func() HandlerScope { return nil }
)

// InitRouter initializes the router and all the routes
func (api *API) InitRouter() {
	api.Router.URL = api.Config.URL.API
	api.Router.SetHeaderFunc = DefaultHeaders
	api.Router.Middlewares = append(api.Router.Middlewares, api.authMiddleware, api.tracingMiddleware, api.maintenanceMiddleware)
	api.Router.PostMiddlewares = append(api.Router.PostMiddlewares, TracingPostMiddleware)

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
	r.Handle("/accesstoken", Scope(sdk.AccessTokenScopeAccessToken), r.POST(api.postNewAccessTokenHandler))
	r.Handle("/accesstoken/{id}", Scope(sdk.AccessTokenScopeAccessToken), r.PUT(api.putRegenAccessTokenHandler), r.DELETE(api.deleteAccessTokenHandler))
	r.Handle("/accesstoken/user/{id}", Scope(sdk.AccessTokenScopeAccessToken), r.GET(api.getAccessTokenByUserHandler))
	r.Handle("/accesstoken/group/{id}", Scope(sdk.AccessTokenScopeAccessToken), r.GET(api.getAccessTokenByGroupHandler))

	// Action
	r.Handle("/action", Scope(sdk.AccessTokenScopeAction), r.GET(api.getActionsHandler), r.POST(api.postActionHandler))
	r.Handle("/action/import", Scope(sdk.AccessTokenScopeAction), r.POST(api.importActionHandler))
	r.Handle("/action/{permGroupName}/{permActionName}", Scope(sdk.AccessTokenScopeAction), r.GET(api.getActionHandler), r.PUT(api.putActionHandler), r.DELETE(api.deleteActionHandler))
	r.Handle("/action/{permGroupName}/{permActionName}/usage", Scope(sdk.AccessTokenScopeAction), r.GET(api.getActionUsageHandler))
	r.Handle("/action/{permGroupName}/{permActionName}/export", Scope(sdk.AccessTokenScopeAction), r.GET(api.getActionExportHandler))
	r.Handle("/action/{permGroupName}/{permActionName}/audit", Scope(sdk.AccessTokenScopeAction), r.GET(api.getActionAuditHandler))
	r.Handle("/action/{permGroupName}/{permActionName}/audit/{auditID}/rollback", Scope(sdk.AccessTokenScopeAction), r.POST(api.postActionAuditRollbackHandler))
	r.Handle("/action/requirement", Scope(sdk.AccessTokenScopeAction), r.GET(api.getActionsRequirements, Auth(false))) // FIXME add auth used by hatcheries
	r.Handle("/project/{permProjectKey}/action", Scope(sdk.AccessTokenScopeProject), r.GET(api.getActionsForProjectHandler))
	r.Handle("/group/{groupID}/action", Scope(sdk.AccessTokenScopeGroup), r.GET(api.getActionsForGroupHandler))
	r.Handle("/actionBuiltin", ScopeNone(), r.GET(api.getActionsBuiltinHandler))
	r.Handle("/actionBuiltin/{permActionBuiltinName}", ScopeNone(), r.GET(api.getActionBuiltinHandler))
	r.Handle("/actionBuiltin/{permActionBuiltinName}/usage", Scope(sdk.AccessTokenScopeAdmin), r.GET(api.getActionBuiltinUsageHandler))

	// Admin
	r.Handle("/admin/maintenance", Scope(sdk.AccessTokenScopeAdmin), r.POST(api.postMaintenanceHandler, NeedAdmin(true)))
	r.Handle("/admin/warning", Scope(sdk.AccessTokenScopeAdmin), r.DELETE(api.adminTruncateWarningsHandler, NeedAdmin(true)))
	r.Handle("/admin/cds/migration", Scope(sdk.AccessTokenScopeAdmin), r.GET(api.getAdminMigrationsHandler, NeedAdmin(true)))
	r.Handle("/admin/cds/migration/{id}/cancel", Scope(sdk.AccessTokenScopeAdmin), r.POST(api.postAdminMigrationCancelHandler, NeedAdmin(true)))
	r.Handle("/admin/cds/migration/{id}/todo", Scope(sdk.AccessTokenScopeAdmin), r.POST(api.postAdminMigrationTodoHandler, NeedAdmin(true)))
	r.Handle("/admin/database/migration/delete/{id}", Scope(sdk.AccessTokenScopeAdmin), r.DELETE(api.deleteDatabaseMigrationHandler, NeedAdmin(true)))
	r.Handle("/admin/database/migration/unlock/{id}", Scope(sdk.AccessTokenScopeAdmin), r.POST(api.postDatabaseMigrationUnlockedHandler, NeedAdmin(true)))
	r.Handle("/admin/database/migration", Scope(sdk.AccessTokenScopeAdmin), r.GET(api.getDatabaseMigrationHandler, NeedAdmin(true)))
	r.Handle("/admin/debug", Scope(sdk.AccessTokenScopeAdmin), r.GET(api.getProfileIndexHandler, Auth(false)))
	r.Handle("/admin/debug/trace", Scope(sdk.AccessTokenScopeAdmin), r.POST(api.getTraceHandler, NeedAdmin(true)), r.GET(api.getTraceHandler, NeedAdmin(true)))
	r.Handle("/admin/debug/cpu", Scope(sdk.AccessTokenScopeAdmin), r.POST(api.getCPUProfileHandler, NeedAdmin(true)), r.GET(api.getCPUProfileHandler, NeedAdmin(true)))
	r.Handle("/admin/debug/{name}", Scope(sdk.AccessTokenScopeAdmin), r.POST(api.getProfileHandler, NeedAdmin(true)), r.GET(api.getProfileHandler, NeedAdmin(true)))
	r.Handle("/admin/plugin", Scope(sdk.AccessTokenScopeAdmin), r.POST(api.postGRPCluginHandler, NeedAdmin(true)), r.GET(api.getAllGRPCluginHandler, NeedAdmin(true)))
	r.Handle("/admin/plugin/{name}", Scope(sdk.AccessTokenScopeAdmin), r.GET(api.getGRPCluginHandler, NeedAdmin(true)), r.PUT(api.putGRPCluginHandler, NeedAdmin(true)), r.DELETE(api.deleteGRPCluginHandler, NeedAdmin(true)))
	r.Handle("/admin/plugin/{name}/binary", Scope(sdk.AccessTokenScopeAdmin), r.POST(api.postGRPCluginBinaryHandler, NeedAdmin(true)))
	r.Handle("/admin/plugin/{name}/binary/{os}/{arch}", Scope(sdk.AccessTokenScopeAdmin), r.GET(api.getGRPCluginBinaryHandler, Auth(false)), r.DELETE(api.deleteGRPCluginBinaryHandler, NeedAdmin(true)))
	r.Handle("/admin/plugin/{name}/binary/{os}/{arch}/infos", Scope(sdk.AccessTokenScopeAdmin), r.GET(api.getGRPCluginBinaryInfosHandler))

	// Admin service
	r.Handle("/admin/service/{name}", Scope(sdk.AccessTokenScopeAdmin), r.GET(api.getAdminServiceHandler, NeedAdmin(true)), r.DELETE(api.deleteAdminServiceHandler, NeedAdmin(true)))
	r.Handle("/admin/services", Scope(sdk.AccessTokenScopeAdmin), r.GET(api.getAdminServicesHandler, NeedAdmin(true)))
	r.Handle("/admin/services/call", Scope(sdk.AccessTokenScopeAdmin), r.GET(api.getAdminServiceCallHandler, NeedAdmin(true)), r.POST(api.postAdminServiceCallHandler, NeedAdmin(true)), r.PUT(api.putAdminServiceCallHandler, NeedAdmin(true)), r.DELETE(api.deleteAdminServiceCallHandler, NeedAdmin(true)))

	// Download file
	r.Handle("/download", ScopeNone(), r.GET(api.downloadsHandler))
	r.Handle("/download/{name}/{os}/{arch}", ScopeNone(), r.GET(api.downloadHandler, Auth(false)))

	// Group
	r.Handle("/group", Scope(sdk.AccessTokenScopeGroup), r.GET(api.getGroupsHandler), r.POST(api.addGroupHandler))
	r.Handle("/group/{permGroupName}", Scope(sdk.AccessTokenScopeGroup), r.GET(api.getGroupHandler), r.PUT(api.updateGroupHandler), r.DELETE(api.deleteGroupHandler))
	r.Handle("/group/{permGroupName}/user", Scope(sdk.AccessTokenScopeGroup), r.POST(api.addUserInGroupHandler))
	r.Handle("/group/{permGroupName}/user/{user}", Scope(sdk.AccessTokenScopeGroup), r.DELETE(api.removeUserFromGroupHandler))
	r.Handle("/group/{permGroupName}/user/{user}/admin", Scope(sdk.AccessTokenScopeGroup), r.POST(api.setUserGroupAdminHandler), r.DELETE(api.removeUserGroupAdminHandler))
	//r.Handle("/group/{permGroupName}/token", Scope(sdk.AccessTokenScopeGroup), r.GET(api.getGroupTokenListHandler), r.POST(api.generateTokenHandler))
	//r.Handle("/group/{permGroupName}/token/{tokenid}", Scope(sdk.AccessTokenScopeGroup), r.DELETE(api.deleteTokenHandler))

	// Hatchery
	r.Handle("/hatchery/count/{workflowNodeRunID}", Scope(sdk.AccessTokenScopeHatchery), r.GET(api.hatcheryCountHandler))

	// Hooks
	r.Handle("/hook/{uuid}/workflow/{workflowID}/vcsevent/{vcsServer}", Scope(sdk.AccessTokenScopeRun), r.GET(api.getHookPollingVCSEvents))

	// Integration
	r.Handle("/integration/models", ScopeNone(), r.GET(api.getIntegrationModelsHandler), r.POST(api.postIntegrationModelHandler, NeedAdmin(true)))
	r.Handle("/integration/models/{name}", ScopeNone(), r.GET(api.getIntegrationModelHandler), r.PUT(api.putIntegrationModelHandler, NeedAdmin(true)), r.DELETE(api.deleteIntegrationModelHandler, NeedAdmin(true)))

	// Broadcast
	r.Handle("/broadcast", ScopeNone(), r.POST(api.addBroadcastHandler, NeedAdmin(true)), r.GET(api.getBroadcastsHandler))
	r.Handle("/broadcast/{id}", ScopeNone(), r.GET(api.getBroadcastHandler), r.PUT(api.updateBroadcastHandler, NeedAdmin(true)), r.DELETE(api.deleteBroadcastHandler, NeedAdmin(true)))
	r.Handle("/broadcast/{id}/mark", Scope(sdk.AccessTokenScopeProject), r.POST(api.postMarkAsReadBroadcastHandler))

	// Overall health
	r.Handle("/mon/status", ScopeNone(), r.GET(api.statusHandler, Auth(false)))
	r.Handle("/mon/version", ScopeNone(), r.GET(VersionHandler, Auth(false)))
	r.Handle("/mon/db/migrate", ScopeNone(), r.GET(api.getMonDBStatusMigrateHandler, NeedAdmin(true)))
	r.Handle("/mon/metrics", ScopeNone(), r.GET(observability.StatsHandler, Auth(false)))
	r.Handle("/mon/errors/{uuid}", ScopeNone(), r.GET(api.getErrorHandler, NeedAdmin(true)))
	r.Handle("/mon/panic/{uuid}", ScopeNone(), r.GET(api.getPanicDumpHandler, Auth(false)))

	r.Handle("/ui/navbar", ScopeNone(), r.GET(api.getNavbarHandler))
	r.Handle("/ui/project/{permProjectKey}/application/{applicationName}/overview", ScopeNone(), r.GET(api.getApplicationOverviewHandler))

	// Import As Code
	r.Handle("/import/{permProjectKey}", Scope(sdk.AccessTokenScopeProject), r.POST(api.postImportAsCodeHandler))
	r.Handle("/import/{permProjectKey}/{uuid}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getImportAsCodeHandler))
	r.Handle("/import/{permProjectKey}/{uuid}/perform", Scope(sdk.AccessTokenScopeProject), r.POST(api.postPerformImportAsCodeHandler))

	// Bookmarks
	r.Handle("/bookmarks", ScopeNone(), r.GET(api.getBookmarksHandler))

	// Project
	r.Handle("/project", Scope(sdk.AccessTokenScopeProject), r.GET(api.getProjectsHandler, AllowProvider(true), EnableTracing()), r.POST(api.addProjectHandler))
	r.Handle("/project/{permProjectKey}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getProjectHandler), r.PUT(api.updateProjectHandler), r.DELETE(api.deleteProjectHandler))
	r.Handle("/project/{permProjectKey}/labels", Scope(sdk.AccessTokenScopeProject), r.PUT(api.putProjectLabelsHandler))
	r.Handle("/project/{permProjectKey}/group", Scope(sdk.AccessTokenScopeProject), r.POST(api.addGroupInProjectHandler))
	r.Handle("/project/{permProjectKey}/group/import", Scope(sdk.AccessTokenScopeProject), r.POST(api.importGroupsInProjectHandler, DEPRECATED))
	r.Handle("/project/{permProjectKey}/group/{group}", Scope(sdk.AccessTokenScopeProject), r.PUT(api.updateGroupRoleOnProjectHandler), r.DELETE(api.deleteGroupFromProjectHandler))
	r.Handle("/project/{permProjectKey}/variable", Scope(sdk.AccessTokenScopeProject), r.GET(api.getVariablesInProjectHandler))
	r.Handle("/project/{permProjectKey}/encrypt", Scope(sdk.AccessTokenScopeProject), r.POST(api.postEncryptVariableHandler))
	r.Handle("/project/{key}/variable/audit", Scope(sdk.AccessTokenScopeProject), r.GET(api.getVariablesAuditInProjectnHandler))
	r.Handle("/project/{permProjectKey}/variable/{name}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getVariableInProjectHandler), r.POST(api.addVariableInProjectHandler), r.PUT(api.updateVariableInProjectHandler), r.DELETE(api.deleteVariableFromProjectHandler))
	r.Handle("/project/{permProjectKey}/variable/{name}/audit", Scope(sdk.AccessTokenScopeProject), r.GET(api.getVariableAuditInProjectHandler))
	r.Handle("/project/{permProjectKey}/applications", Scope(sdk.AccessTokenScopeProject), r.GET(api.getApplicationsHandler, AllowProvider(true)), r.POST(api.addApplicationHandler))
	r.Handle("/project/{permProjectKey}/integrations", Scope(sdk.AccessTokenScopeProject), r.GET(api.getProjectIntegrationsHandler), r.POST(api.postProjectIntegrationHandler))
	r.Handle("/project/{permProjectKey}/integrations/{integrationName}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getProjectIntegrationHandler /*, AllowServices(true)*/), r.PUT(api.putProjectIntegrationHandler), r.DELETE(api.deleteProjectIntegrationHandler))
	r.Handle("/project/{permProjectKey}/notifications", Scope(sdk.AccessTokenScopeProject), r.GET(api.getProjectNotificationsHandler, DEPRECATED))
	r.Handle("/project/{permProjectKey}/all/keys", Scope(sdk.AccessTokenScopeProject), r.GET(api.getAllKeysProjectHandler))
	r.Handle("/project/{permProjectKey}/keys", Scope(sdk.AccessTokenScopeProject), r.GET(api.getKeysInProjectHandler), r.POST(api.addKeyInProjectHandler))
	r.Handle("/project/{permProjectKey}/keys/{name}", Scope(sdk.AccessTokenScopeProject), r.DELETE(api.deleteKeyInProjectHandler))
	// Import Application
	r.Handle("/project/{permProjectKey}/import/application", Scope(sdk.AccessTokenScopeProject), r.POST(api.postApplicationImportHandler))
	// Export Application
	r.Handle("/project/{permProjectKey}/export/application/{applicationName}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getApplicationExportHandler))

	r.Handle("/warning/{permProjectKey}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getWarningsHandler))
	r.Handle("/warning/{permProjectKey}/{hash}", Scope(sdk.AccessTokenScopeProject), r.PUT(api.putWarningsHandler))

	// Application
	r.Handle("/project/{permProjectKey}/application/{applicationName}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getApplicationHandler), r.PUT(api.updateApplicationHandler), r.DELETE(api.deleteApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/metrics/{metricName}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getApplicationMetricHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/keys", Scope(sdk.AccessTokenScopeProject), r.GET(api.getKeysInApplicationHandler), r.POST(api.addKeyInApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/keys/{name}", Scope(sdk.AccessTokenScopeProject), r.DELETE(api.deleteKeyInApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/vcsinfos", Scope(sdk.AccessTokenScopeProject), r.GET(api.getApplicationVCSInfosHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/clone", Scope(sdk.AccessTokenScopeProject), r.POST(api.cloneApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/variable", Scope(sdk.AccessTokenScopeProject), r.GET(api.getVariablesInApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/variable/audit", Scope(sdk.AccessTokenScopeProject), r.GET(api.getVariablesAuditInApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/variable/{name}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getVariableInApplicationHandler), r.POST(api.addVariableInApplicationHandler), r.PUT(api.updateVariableInApplicationHandler), r.DELETE(api.deleteVariableFromApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/variable/{name}/audit", Scope(sdk.AccessTokenScopeProject), r.GET(api.getVariableAuditInApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/vulnerability/{id}", Scope(sdk.AccessTokenScopeProject), r.POST(api.postVulnerabilityHandler))
	// Application deployment
	r.Handle("/project/{permProjectKey}/application/{applicationName}/deployment/config/{integration}", Scope(sdk.AccessTokenScopeProject), r.POST(api.postApplicationDeploymentStrategyConfigHandler, AllowProvider(true)), r.GET(api.getApplicationDeploymentStrategyConfigHandler), r.DELETE(api.deleteApplicationDeploymentStrategyConfigHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/deployment/config", Scope(sdk.AccessTokenScopeProject), r.GET(api.getApplicationDeploymentStrategiesConfigHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/metadata/{metadata}", Scope(sdk.AccessTokenScopeProject), r.POST(api.postApplicationMetadataHandler, AllowProvider(true)))

	// Pipeline
	r.Handle("/project/{permProjectKey}/pipeline", Scope(sdk.AccessTokenScopeProject), r.GET(api.getPipelinesHandler), r.POST(api.addPipelineHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/parameter", Scope(sdk.AccessTokenScopeProject), r.GET(api.getParametersInPipelineHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/parameter/{name}", Scope(sdk.AccessTokenScopeProject), r.POST(api.addParameterInPipelineHandler), r.PUT(api.updateParameterInPipelineHandler), r.DELETE(api.deleteParameterFromPipelineHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getPipelineHandler), r.PUT(api.updatePipelineHandler), r.DELETE(api.deletePipelineHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/rollback/{auditID}", Scope(sdk.AccessTokenScopeProject), r.POST(api.postPipelineRollbackHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/audits", Scope(sdk.AccessTokenScopeProject), r.GET(api.getPipelineAuditHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/stage", Scope(sdk.AccessTokenScopeProject), r.POST(api.addStageHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/stage/move", Scope(sdk.AccessTokenScopeProject), r.POST(api.moveStageHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/stage/{stageID}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getStageHandler), r.PUT(api.updateStageHandler), r.DELETE(api.deleteStageHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/stage/{stageID}/job", Scope(sdk.AccessTokenScopeProject), r.POST(api.addJobToStageHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/stage/{stageID}/job/{jobID}", Scope(sdk.AccessTokenScopeProject), r.PUT(api.updateJobHandler), r.DELETE(api.deleteJobHandler))

	// Preview pipeline
	r.Handle("/project/{permProjectKey}/preview/pipeline", Scope(sdk.AccessTokenScopeProject), r.POST(api.postPipelinePreviewHandler))
	// Import pipeline
	r.Handle("/project/{permProjectKey}/import/pipeline", Scope(sdk.AccessTokenScopeProject), r.POST(api.importPipelineHandler))
	// Import pipeline (ONLY USE FOR UI)
	r.Handle("/project/{permProjectKey}/import/pipeline/{pipelineKey}", Scope(sdk.AccessTokenScopeProject), r.PUT(api.putImportPipelineHandler))
	// Export pipeline
	r.Handle("/project/{permProjectKey}/export/pipeline/{pipelineKey}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getPipelineExportHandler))

	// Workflows
	r.Handle("/workflow/artifact/{hash}", ScopeNone(), r.GET(api.downloadworkflowArtifactDirectHandler, Auth(false)))

	r.Handle("/project/{permProjectKey}/workflows", Scope(sdk.AccessTokenScopeProject), r.POST(api.postWorkflowHandler, EnableTracing()), r.GET(api.getWorkflowsHandler, AllowProvider(true), EnableTracing()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getWorkflowHandler, AllowProvider(true), EnableTracing()), r.PUT(api.putWorkflowHandler, EnableTracing()), r.DELETE(api.deleteWorkflowHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/icon", Scope(sdk.AccessTokenScopeProject), r.PUT(api.putWorkflowIconHandler), r.DELETE(api.deleteWorkflowIconHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/ascode/{uuid}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getWorkflowAsCodeHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/ascode", Scope(sdk.AccessTokenScopeProject), r.POST(api.postWorkflowAsCodeHandler, EnableTracing()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/ascode/resync/pr", Scope(sdk.AccessTokenScopeProject), r.POST(api.postResyncPRWorkflowAsCodeHandler, EnableTracing()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/label", Scope(sdk.AccessTokenScopeProject), r.POST(api.postWorkflowLabelHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/label/{labelID}", Scope(sdk.AccessTokenScopeProject), r.DELETE(api.deleteWorkflowLabelHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/rollback/{auditID}", Scope(sdk.AccessTokenScopeProject), r.POST(api.postWorkflowRollbackHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/groups", Scope(sdk.AccessTokenScopeProject), r.POST(api.postWorkflowGroupHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/groups/{groupName}", Scope(sdk.AccessTokenScopeProject), r.PUT(api.putWorkflowGroupHandler), r.DELETE(api.deleteWorkflowGroupHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/hooks/{uuid}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getWorkflowHookHandler))
	r.Handle("/project/{key}/workflow/{permWorkflowName}/node/{nodeID}/hook/model", Scope(sdk.AccessTokenScopeProject), r.GET(api.getWorkflowHookModelsHandler))
	r.Handle("/project/{key}/workflow/{permWorkflowName}/node/{nodeID}/outgoinghook/model", Scope(sdk.AccessTokenScopeProject), r.GET(api.getWorkflowOutgoingHookModelsHandler))

	// Outgoing hook model
	r.Handle("/workflow/outgoinghook/model", ScopeNone(), r.GET(api.getWorkflowOutgoingHookModelsHandler))

	// Preview workflows
	r.Handle("/project/{permProjectKey}/preview/workflows", Scope(sdk.AccessTokenScopeProject), r.POST(api.postWorkflowPreviewHandler))
	// Import workflows
	r.Handle("/project/{permProjectKey}/import/workflows", Scope(sdk.AccessTokenScopeProject), r.POST(api.postWorkflowImportHandler))
	// Import workflows (ONLY USE FOR UI EDIT AS CODE)
	r.Handle("/project/{key}/import/workflows/{permWorkflowName}", Scope(sdk.AccessTokenScopeProject), r.PUT(api.putWorkflowImportHandler))
	// Export workflows
	r.Handle("/project/{key}/export/workflows/{permWorkflowName}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getWorkflowExportHandler))
	// Pull workflows
	r.Handle("/project/{key}/pull/workflows/{permWorkflowName}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getWorkflowPullHandler))
	// Push workflows
	r.Handle("/project/{permProjectKey}/push/workflows", Scope(sdk.AccessTokenScopeProject), r.POST(api.postWorkflowPushHandler))

	// Workflows run
	r.Handle("/project/{permProjectKey}/runs", Scope(sdk.AccessTokenScopeProject), r.GET(api.getWorkflowAllRunsHandler, EnableTracing()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/artifact/{artifactId}", Scope(sdk.AccessTokenScopeRun), r.GET(api.getDownloadArtifactHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs", Scope(sdk.AccessTokenScopeRun), r.GET(api.getWorkflowRunsHandler, EnableTracing()), r.POSTEXECUTE(api.postWorkflowRunHandler /*, AllowServices(true)*/, EnableTracing()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/branch/{branch}", Scope(sdk.AccessTokenScopeRun), r.DELETE(api.deleteWorkflowRunsBranchHandler /*, NeedService()*/))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/latest", Scope(sdk.AccessTokenScopeRun), r.GET(api.getLatestWorkflowRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/tags", Scope(sdk.AccessTokenScopeRun), r.GET(api.getWorkflowRunTagsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/num", Scope(sdk.AccessTokenScopeRun), r.GET(api.getWorkflowRunNumHandler), r.POST(api.postWorkflowRunNumHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}", Scope(sdk.AccessTokenScopeRun), r.GET(api.getWorkflowRunHandler /*, AllowServices(true)*/))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/stop", Scope(sdk.AccessTokenScopeRun), r.POSTEXECUTE(api.stopWorkflowRunHandler, EnableTracing()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/vcs/resync", Scope(sdk.AccessTokenScopeRun), r.POSTEXECUTE(api.postResyncVCSWorkflowRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/resync", Scope(sdk.AccessTokenScopeRun), r.POST(api.resyncWorkflowRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/artifacts", Scope(sdk.AccessTokenScopeRun), r.GET(api.getWorkflowRunArtifactsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}", Scope(sdk.AccessTokenScopeRun), r.GET(api.getWorkflowNodeRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/stop", Scope(sdk.AccessTokenScopeRun), r.POSTEXECUTE(api.stopWorkflowNodeRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeID}/history", Scope(sdk.AccessTokenScopeRun), r.GET(api.getWorkflowNodeRunHistoryHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/{nodeName}/commits", Scope(sdk.AccessTokenScopeRun), r.GET(api.getWorkflowCommitsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/job/{runJobId}/info", Scope(sdk.AccessTokenScopeRun), r.GET(api.getWorkflowNodeRunJobSpawnInfosHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/job/{runJobId}/log/service", Scope(sdk.AccessTokenScopeRun), r.GET(api.getWorkflowNodeRunJobServiceLogsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/job/{runJobId}/step/{stepOrder}", Scope(sdk.AccessTokenScopeRun), r.GET(api.getWorkflowNodeRunJobStepHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/node/{nodeID}/triggers/condition", Scope(sdk.AccessTokenScopeRun), r.GET(api.getWorkflowTriggerConditionHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/release", Scope(sdk.AccessTokenScopeRun), r.POST(api.releaseApplicationWorkflowHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/hooks/{hookRunID}/callback", Scope(sdk.AccessTokenScopeRun), r.POST(api.postWorkflowJobHookCallbackHandler /*, AllowServices(true)*/))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/hooks/{hookRunID}/details", Scope(sdk.AccessTokenScopeRun), r.GET(api.getWorkflowJobHookDetailsHandler /*, NeedService()*/))

	// Environment
	r.Handle("/project/{permProjectKey}/environment", Scope(sdk.AccessTokenScopeProject), r.GET(api.getEnvironmentsHandler), r.POST(api.addEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/import", Scope(sdk.AccessTokenScopeProject), r.POST(api.importNewEnvironmentHandler, DEPRECATED))
	r.Handle("/project/{permProjectKey}/environment/import/{environmentName}", Scope(sdk.AccessTokenScopeProject), r.POST(api.importIntoEnvironmentHandler, DEPRECATED))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getEnvironmentHandler), r.PUT(api.updateEnvironmentHandler), r.DELETE(api.deleteEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/usage", Scope(sdk.AccessTokenScopeProject), r.GET(api.getEnvironmentUsageHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/keys", Scope(sdk.AccessTokenScopeProject), r.GET(api.getKeysInEnvironmentHandler), r.POST(api.addKeyInEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/keys/{name}", Scope(sdk.AccessTokenScopeProject), r.DELETE(api.deleteKeyInEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/clone/{cloneName}", Scope(sdk.AccessTokenScopeProject), r.POST(api.cloneEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/variable", Scope(sdk.AccessTokenScopeProject), r.GET(api.getVariablesInEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/variable/{name}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getVariableInEnvironmentHandler), r.POST(api.addVariableInEnvironmentHandler), r.PUT(api.updateVariableInEnvironmentHandler), r.DELETE(api.deleteVariableFromEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/variable/{name}/audit", Scope(sdk.AccessTokenScopeProject), r.GET(api.getVariableAuditInEnvironmentHandler))

	// Import Environment
	r.Handle("/project/{permProjectKey}/import/environment", Scope(sdk.AccessTokenScopeProject), r.POST(api.postEnvironmentImportHandler))
	// Export Environment
	r.Handle("/project/{permProjectKey}/export/environment/{environmentName}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getEnvironmentExportHandler))

	// Project storage
	r.Handle("/project/{permProjectKey}/storage/{integrationName}", Scope(sdk.AccessTokenScopeRunExecution), r.GET(api.getArtifactsStoreHandler))
	r.Handle("/project/{permProjectKey}/storage/{integrationName}/artifact/{ref}", Scope(sdk.AccessTokenScopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobArtifactHandler /*, NeedWorker()*/, EnableTracing(), MaintenanceAware()))
	r.Handle("/project/{permProjectKey}/storage/{integrationName}/artifact/{ref}/url", Scope(sdk.AccessTokenScopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobArtifacWithTempURLHandler /*, NeedWorker()*/, EnableTracing(), MaintenanceAware()))
	r.Handle("/project/{permProjectKey}/storage/{integrationName}/artifact/{ref}/url/callback", Scope(sdk.AccessTokenScopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobArtifactWithTempURLCallbackHandler /*, NeedWorker()*/, EnableTracing(), MaintenanceAware()))
	r.Handle("/project/{permProjectKey}/storage/{integrationName}/staticfiles/{name}", Scope(sdk.AccessTokenScopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobStaticFilesHandler /*, NeedWorker()*/, EnableTracing(), MaintenanceAware()))

	// Cache
	r.Handle("/project/{permProjectKey}/storage/{integrationName}/cache/{tag}", Scope(sdk.AccessTokenScopeRunExecution), r.POSTEXECUTE(api.postPushCacheHandler /*, NeedWorker()*/), r.GET(api.getPullCacheHandler /*, NeedWorker()*/))
	r.Handle("/project/{permProjectKey}/storage/{integrationName}/cache/{tag}/url", Scope(sdk.AccessTokenScopeRunExecution), r.POSTEXECUTE(api.postPushCacheWithTempURLHandler /*, NeedWorker()*/), r.GET(api.getPullCacheWithTempURLHandler /*, NeedWorker()*/))

	//Workflow queue
	r.Handle("/queue/workflows", Scope(sdk.AccessTokenScopeRun), r.GET(api.getWorkflowJobQueueHandler, EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/count", Scope(sdk.AccessTokenScopeRun), r.GET(api.countWorkflowJobQueueHandler, EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{id}/take", Scope(sdk.AccessTokenScopeRunExecution), r.POST(api.postTakeWorkflowJobHandler /*, NeedWorker()*/, EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{id}/book", Scope(sdk.AccessTokenScopeRunExecution), r.POST(api.postBookWorkflowJobHandler /*, NeedHatchery()*/, EnableTracing(), MaintenanceAware()), r.DELETE(api.deleteBookWorkflowJobHandler /*, NeedHatchery()*/, EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{id}/attempt", Scope(sdk.AccessTokenScopeRunExecution), r.POST(api.postIncWorkflowJobAttemptHandler /*, NeedHatchery()*/, EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{id}/infos", Scope(sdk.AccessTokenScopeRunExecution), r.GET(api.getWorkflowJobHandler /*, NeedWorker()*/ /*, NeedHatchery()*/, EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/vulnerability", Scope(sdk.AccessTokenScopeRunExecution), r.POSTEXECUTE(api.postVulnerabilityReportHandler /*, NeedWorker()*/, EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{id}/spawn/infos", Scope(sdk.AccessTokenScopeRunExecution), r.POST(r.Asynchronous(api.postSpawnInfosWorkflowJobHandler, 1) /*, NeedHatchery()*/, EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/result", Scope(sdk.AccessTokenScopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobResultHandler /*, NeedWorker()*/, EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/log", Scope(sdk.AccessTokenScopeRunExecution), r.POSTEXECUTE(r.Asynchronous(api.postWorkflowJobLogsHandler, 1) /*, NeedWorker()*/, MaintenanceAware()))
	r.Handle("/queue/workflows/log/service", Scope(sdk.AccessTokenScopeRunExecution), r.POSTEXECUTE(r.Asynchronous(api.postWorkflowJobServiceLogsHandler, 1) /*, NeedHatchery()*/, MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/coverage", Scope(sdk.AccessTokenScopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobCoverageResultsHandler /*, NeedWorker()*/, EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/test", Scope(sdk.AccessTokenScopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobTestsResultsHandler /*, NeedWorker()*/, EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/tag", Scope(sdk.AccessTokenScopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobTagsHandler /*, NeedWorker()*/, EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/variable", Scope(sdk.AccessTokenScopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobVariableHandler /*, NeedWorker()*/, EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/step", Scope(sdk.AccessTokenScopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobStepStatusHandler /*, NeedWorker()*/, EnableTracing(), MaintenanceAware()))

	r.Handle("/variable/type", ScopeNone(), r.GET(api.getVariableTypeHandler))
	r.Handle("/parameter/type", ScopeNone(), r.GET(api.getParameterTypeHandler))
	r.Handle("/notification/type", ScopeNone(), r.GET(api.getUserNotificationTypeHandler))
	r.Handle("/notification/state", ScopeNone(), r.GET(api.getUserNotificationStateValueHandler))

	// RepositoriesManager
	r.Handle("/repositories_manager", Scope(sdk.AccessTokenScopeProject), r.GET(api.getRepositoriesManagerHandler))
	r.Handle("/repositories_manager/oauth2/callback", Scope(sdk.AccessTokenScopeProject), r.GET(api.repositoriesManagerOAuthCallbackHandler, Auth(false)))
	// RepositoriesManager for projects
	r.Handle("/project/{permProjectKey}/repositories_manager", Scope(sdk.AccessTokenScopeProject), r.GET(api.getRepositoriesManagerForProjectHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/authorize", Scope(sdk.AccessTokenScopeProject), r.POST(api.repositoriesManagerAuthorizeHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/authorize/callback", Scope(sdk.AccessTokenScopeProject), r.POST(api.repositoriesManagerAuthorizeCallbackHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/authorize/basicauth", Scope(sdk.AccessTokenScopeProject), r.POST(api.repositoriesManagerAuthorizeBasicHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}", Scope(sdk.AccessTokenScopeProject), r.DELETE(api.deleteRepositoriesManagerHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/repo", Scope(sdk.AccessTokenScopeProject), r.GET(api.getRepoFromRepositoriesManagerHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/repos", Scope(sdk.AccessTokenScopeProject), r.GET(api.getReposFromRepositoriesManagerHandler))

	// RepositoriesManager for applications
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/application/{applicationName}/attach", Scope(sdk.AccessTokenScopeProject), r.POST(api.attachRepositoriesManagerHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/application/{applicationName}/detach", Scope(sdk.AccessTokenScopeProject), r.POST(api.detachRepositoriesManagerHandler))

	// Suggest
	r.Handle("/suggest/variable/{permProjectKey}", Scope(sdk.AccessTokenScopeProject), r.GET(api.getVariablesHandler))

	//Requirements
	r.Handle("/requirement/types", ScopeNone(), r.GET(api.getRequirementTypesHandler))
	r.Handle("/requirement/types/{type}", ScopeNone(), r.GET(api.getRequirementTypeValuesHandler))

	// config
	r.Handle("/config/user", ScopeNone(), r.GET(api.ConfigUserHandler, Auth(false)))
	r.Handle("/config/vcs", ScopeNone(), r.GET(api.ConfigVCShandler))

	// Users
	r.Handle("/user", Scope(sdk.AccessTokenScopeUser), r.GET(api.getUsersHandler))
	r.Handle("/user/logged", Scope(sdk.AccessTokenScopeUser), r.GET(api.getUserLoggedHandler, Auth(false)))
	r.Handle("/user/me", Scope(sdk.AccessTokenScopeUser), r.GET(api.getUserLoggedHandler, Auth(false), DEPRECATED))
	r.Handle("/user/favorite", Scope(sdk.AccessTokenScopeUser), r.POST(api.postUserFavoriteHandler))
	r.Handle("/user/timeline", Scope(sdk.AccessTokenScopeUser), r.GET(api.getTimelineHandler))
	r.Handle("/user/timeline/filter", Scope(sdk.AccessTokenScopeUser), r.GET(api.getTimelineFilterHandler), r.POST(api.postTimelineFilterHandler))
	r.Handle("/user/signup", ScopeNone(), r.POST(api.addUserHandler, Auth(false)))
	r.Handle("/user/{username}", Scope(sdk.AccessTokenScopeUser), r.GET(api.getUserHandler), r.PUT(api.updateUserHandler), r.DELETE(api.deleteUserHandler))
	r.Handle("/user/{username}/groups", Scope(sdk.AccessTokenScopeUser), r.GET(api.getUserGroupsHandler))
	r.Handle("/user/{username}/confirm/{token}", Scope(sdk.AccessTokenScopeUser), r.GET(api.confirmUserHandler, Auth(false)))
	r.Handle("/user/{username}/reset", Scope(sdk.AccessTokenScopeUser), r.POST(api.resetUserHandler, Auth(false)))

	// Workers
	r.Handle("/worker", Scope(sdk.AccessTokenScopeAdmin, sdk.AccessTokenScopeWorker, sdk.AccessTokenScopeHatchery), r.GET(api.getWorkersHandler), r.POST(api.registerWorkerHandler, Auth(false)))
	r.Handle("/worker/refresh", Scope(sdk.AccessTokenScopeWorker), r.POST(api.refreshWorkerHandler))
	r.Handle("/worker/checking", Scope(sdk.AccessTokenScopeWorker), r.POST(api.workerCheckingHandler))
	r.Handle("/worker/waiting", Scope(sdk.AccessTokenScopeWorker), r.POST(api.workerWaitingHandler))
	r.Handle("/worker/unregister", Scope(sdk.AccessTokenScopeWorker), r.POST(api.unregisterWorkerHandler))
	r.Handle("/worker/{id}/disable", Scope(sdk.AccessTokenScopeAdmin, sdk.AccessTokenScopeHatchery), r.POST(api.disableWorkerHandler))

	// Worker models
	r.Handle("/worker/model", Scope(sdk.AccessTokenScopeWorkerModel), r.POST(api.postWorkerModelHandler), r.GET(api.getWorkerModelsHandler))
	r.Handle("/worker/model/book/{permModelID}", Scope(sdk.AccessTokenScopeWorkerModel), r.PUT(api.bookWorkerModelHandler /*NeedHatchery()*/))
	r.Handle("/worker/model/error/{permModelID}", Scope(sdk.AccessTokenScopeWorkerModel), r.PUT(api.spawnErrorWorkerModelHandler /*NeedHatchery()*/))
	r.Handle("/worker/model/enabled", Scope(sdk.AccessTokenScopeWorkerModel), r.GET(api.getWorkerModelsEnabledHandler /*NeedHatchery()*/))
	r.Handle("/worker/model/type", Scope(sdk.AccessTokenScopeWorkerModel), r.GET(api.getWorkerModelTypesHandler))
	r.Handle("/worker/model/communication", Scope(sdk.AccessTokenScopeWorkerModel), r.GET(api.getWorkerModelCommunicationsHandler))
	r.Handle("/worker/model/capability/type", Scope(sdk.AccessTokenScopeWorkerModel), r.GET(api.getRequirementTypesHandler))
	r.Handle("/worker/model/pattern", Scope(sdk.AccessTokenScopeWorkerModel), r.POST(api.postAddWorkerModelPatternHandler, NeedAdmin(true)), r.GET(api.getWorkerModelPatternsHandler))
	r.Handle("/worker/model/pattern/{type}/{name}", Scope(sdk.AccessTokenScopeWorkerModel), r.GET(api.getWorkerModelPatternHandler), r.PUT(api.putWorkerModelPatternHandler, NeedAdmin(true)), r.DELETE(api.deleteWorkerModelPatternHandler, NeedAdmin(true)))
	r.Handle("/worker/model/import", Scope(sdk.AccessTokenScopeWorkerModel), r.POST(api.postWorkerModelImportHandler))
	r.Handle("/worker/model/{permGroupName}/{permModelName}", Scope(sdk.AccessTokenScopeWorkerModel), r.GET(api.getWorkerModelHandler), r.PUT(api.putWorkerModelHandler), r.DELETE(api.deleteWorkerModelHandler))
	r.Handle("/worker/model/{permGroupName}/{permModelName}/export", Scope(sdk.AccessTokenScopeWorkerModel), r.GET(api.getWorkerModelExportHandler))
	r.Handle("/worker/model/{permGroupName}/{permModelName}/usage", Scope(sdk.AccessTokenScopeWorkerModel), r.GET(api.getWorkerModelUsageHandler))
	r.Handle("/project/{permProjectKey}/worker/model", Scope(sdk.AccessTokenScopeWorkerModel), r.GET(api.getWorkerModelsForProjectHandler))
	r.Handle("/group/{groupID}/worker/model", Scope(sdk.AccessTokenScopeWorkerModel), r.GET(api.getWorkerModelsForGroupHandler))

	// Workflows
	r.Handle("/workflow/hook", Scope(sdk.AccessTokenScopeHooks), r.GET(api.getWorkflowHooksHandler /*, NeedService()*/))
	r.Handle("/workflow/hook/model/{model}", ScopeNone(), r.GET(api.getWorkflowHookModelHandler), r.POST(api.postWorkflowHookModelHandler, NeedAdmin(true)), r.PUT(api.putWorkflowHookModelHandler, NeedAdmin(true)))

	// SSE
	r.Handle("/events", ScopeNone(), r.GET(api.eventsBroker.ServeHTTP))

	// Feature
	r.Handle("/feature/clean", ScopeNone(), r.POST(api.cleanFeatureHandler, NeedToken("X-Izanami-Token", api.Config.Features.Izanami.Token), Auth(false)))

	// Engine µServices
	r.Handle("/services/register", ScopeNone(), r.POST(api.postServiceRegisterHandler, Auth(false)))
	r.Handle("/services/{type}", ScopeNone(), r.GET(api.getExternalServiceHandler /*, NeedWorker()*/))

	// Templates
	r.Handle("/template", Scope(sdk.AccessTokenScopeTemplate), r.GET(api.getTemplatesHandler), r.POST(api.postTemplateHandler))
	r.Handle("/template/push", Scope(sdk.AccessTokenScopeTemplate), r.POST(api.postTemplatePushHandler))
	r.Handle("/template/{permGroupName}/{permTemplateSlug}", Scope(sdk.AccessTokenScopeTemplate), r.GET(api.getTemplateHandler), r.PUT(api.putTemplateHandler), r.DELETE(api.deleteTemplateHandler))
	r.Handle("/template/{permGroupName}/{permTemplateSlug}/pull", Scope(sdk.AccessTokenScopeTemplate), r.POST(api.postTemplatePullHandler))
	r.Handle("/template/{permGroupName}/{permTemplateSlug}/apply", Scope(sdk.AccessTokenScopeTemplate), r.POST(api.postTemplateApplyHandler))
	r.Handle("/template/{permGroupName}/{permTemplateSlug}/bulk", Scope(sdk.AccessTokenScopeTemplate), r.POST(api.postTemplateBulkHandler))
	r.Handle("/template/{permGroupName}/{permTemplateSlug}/bulk/{bulkID}", Scope(sdk.AccessTokenScopeTemplate), r.GET(api.getTemplateBulkHandler))
	r.Handle("/template/{permGroupName}/{permTemplateSlug}/instance", Scope(sdk.AccessTokenScopeTemplate), r.GET(api.getTemplateInstancesHandler))
	r.Handle("/template/{permGroupName}/{permTemplateSlug}/instance/{instanceID}", Scope(sdk.AccessTokenScopeTemplate), r.DELETE(api.deleteTemplateInstanceHandler))
	r.Handle("/template/{permGroupName}/{permTemplateSlug}/audit", Scope(sdk.AccessTokenScopeTemplate), r.GET(api.getTemplateAuditsHandler))
	r.Handle("/template/{permGroupName}/{permTemplateSlug}/usage", Scope(sdk.AccessTokenScopeTemplate), r.GET(api.getTemplateUsageHandler))
	r.Handle("/project/{key}/workflow/{permWorkflowName}/templateInstance", Scope(sdk.AccessTokenScopeTemplate), r.GET(api.getTemplateInstanceHandler))

	//Not Found handler
	r.Mux.NotFoundHandler = http.HandlerFunc(NotFoundHandler)
}
