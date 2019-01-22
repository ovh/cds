package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	"github.com/ovh/cds/engine/api/observability"
)

// InitRouter initializes the router and all the routes
func (api *API) InitRouter() {
	api.Router.URL = api.Config.URL.API
	api.Router.SetHeaderFunc = DefaultHeaders
	api.Router.Middlewares = append(api.Router.Middlewares, api.authMiddleware, api.tracingMiddleware, api.maintenanceMiddleware)
	api.Router.PostMiddlewares = append(api.Router.PostMiddlewares, api.deletePermissionMiddleware, TracingPostMiddleware)

	r := api.Router
	r.Handle("/login", r.POST(api.loginUserHandler, Auth(false)))

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
	r.Handle("/accesstoken", r.POST(api.postNewAccessTokenHandler))
	r.Handle("/accesstoken/{id}", r.PUT(api.putRegenAccessTokenHandler))
	r.Handle("/accesstoken/user/{id}", r.GET(api.getAccessTokenByUserHandler))
	r.Handle("/accesstoken/group/{id}", r.GET(api.getAccessTokenByGroupHandler))

	// Action
	r.Handle("/action", r.GET(api.getActionsHandler))
	r.Handle("/action/import", r.POST(api.importActionHandler, NeedAdmin(true)))

	r.Handle("/action/requirement", r.GET(api.getActionsRequirements, Auth(false)))
	r.Handle("/action/{permActionName}", r.GET(api.getActionHandler), r.POST(api.addActionHandler), r.PUT(api.updateActionHandler), r.DELETE(api.deleteActionHandler))
	r.Handle("/action/{actionName}/using", r.GET(api.getPipelinesUsingActionHandler, NeedAdmin(true)))
	r.Handle("/action/{permActionName}/export", r.GET(api.getActionExportHandler))
	r.Handle("/action/{actionID}/audit", r.GET(api.getActionAuditHandler, NeedAdmin(true)))

	// Admin
	r.Handle("/admin/maintenance", r.POST(api.postMaintenanceHandler, NeedAdmin(true)))
	r.Handle("/admin/warning", r.DELETE(api.adminTruncateWarningsHandler, NeedAdmin(true)))
	r.Handle("/admin/cds/migration", r.GET(api.getAdminMigrationsHandler, NeedAdmin(true)))
	r.Handle("/admin/cds/migration/{id}/cancel", r.POST(api.postAdminMigrationCancelHandler, NeedAdmin(true)))
	r.Handle("/admin/cds/migration/{id}/todo", r.POST(api.postAdminMigrationTodoHandler, NeedAdmin(true)))
	r.Handle("/admin/database/migration/delete/{id}", r.DELETE(api.deleteDatabaseMigrationHandler, NeedAdmin(true)))
	r.Handle("/admin/database/migration/unlock/{id}", r.POST(api.postDatabaseMigrationUnlockedHandler, NeedAdmin(true)))
	r.Handle("/admin/database/migration", r.GET(api.getDatabaseMigrationHandler, NeedAdmin(true)))
	r.Handle("/admin/debug", r.GET(api.getProfileIndexHandler, Auth(false)))
	r.Handle("/admin/debug/trace", r.POST(api.getTraceHandler, NeedAdmin(true)), r.GET(api.getTraceHandler, NeedAdmin(true)))
	r.Handle("/admin/debug/cpu", r.POST(api.getCPUProfileHandler, NeedAdmin(true)), r.GET(api.getCPUProfileHandler, NeedAdmin(true)))
	r.Handle("/admin/debug/{name}", r.POST(api.getProfileHandler, NeedAdmin(true)), r.GET(api.getProfileHandler, NeedAdmin(true)))
	r.Handle("/admin/plugin", r.POST(api.postPGRPCluginHandler, NeedAdmin(true)), r.GET(api.getAllGRPCluginHandler, NeedAdmin(true)))
	r.Handle("/admin/plugin/{name}", r.GET(api.getGRPCluginHandler, NeedAdmin(true)), r.PUT(api.putGRPCluginHandler, NeedAdmin(true)), r.DELETE(api.deleteGRPCluginHandler, NeedAdmin(true)))
	r.Handle("/admin/plugin/{name}/binary", r.POST(api.postGRPCluginBinaryHandler, NeedAdmin(true)))
	r.Handle("/admin/plugin/{name}/binary/{os}/{arch}", r.GET(api.getGRPCluginBinaryHandler, Auth(false)), r.DELETE(api.deleteGRPCluginBinaryHandler, NeedAdmin(true)))
	r.Handle("/admin/plugin/{name}/binary/{os}/{arch}/infos", r.GET(api.getGRPCluginBinaryInfosHandler))

	// Admin service
	r.Handle("/admin/service/{name}", r.GET(api.getAdminServiceHandler, NeedAdmin(true)), r.DELETE(api.deleteAdminServiceHandler, NeedAdmin(true)))
	r.Handle("/admin/services", r.GET(api.getAdminServicesHandler, NeedAdmin(true)))
	r.Handle("/admin/services/call", r.GET(api.getAdminServiceCallHandler, NeedAdmin(true)), r.POST(api.postAdminServiceCallHandler, NeedAdmin(true)), r.PUT(api.putAdminServiceCallHandler, NeedAdmin(true)), r.DELETE(api.deleteAdminServiceCallHandler, NeedAdmin(true)))

	// Download file
	r.Handle("/download", r.GET(api.downloadsHandler))
	r.Handle("/download/{name}/{os}/{arch}", r.GET(api.downloadHandler, Auth(false)))

	// Group
	r.Handle("/group", r.GET(api.getGroupsHandler), r.POST(api.addGroupHandler))
	r.Handle("/group/public", r.GET(api.getPublicGroupsHandler))
	r.Handle("/group/{permGroupName}", r.GET(api.getGroupHandler), r.PUT(api.updateGroupHandler), r.DELETE(api.deleteGroupHandler))
	r.Handle("/group/{permGroupName}/user", r.POST(api.addUserInGroupHandler))
	r.Handle("/group/{permGroupName}/user/{user}", r.DELETE(api.removeUserFromGroupHandler))
	r.Handle("/group/{permGroupName}/user/{user}/admin", r.POST(api.setUserGroupAdminHandler), r.DELETE(api.removeUserGroupAdminHandler))
	r.Handle("/group/{permGroupName}/token", r.GET(api.getGroupTokenListHandler), r.POST(api.generateTokenHandler))
	r.Handle("/group/{permGroupName}/token/{tokenid}", r.DELETE(api.deleteTokenHandler))

	// Hatchery
	r.Handle("/hatchery/count/{workflowNodeRunID}", r.GET(api.hatcheryCountHandler))

	// Hooks
	r.Handle("/hook/{uuid}/workflow/{workflowID}/vcsevent/{vcsServer}", r.GET(api.getHookPollingVCSEvents))

	// Platform
	r.Handle("/platform/models", r.GET(api.getPlatformModelsHandler), r.POST(api.postPlatformModelHandler, NeedAdmin(true)))
	r.Handle("/platform/models/{name}", r.GET(api.getPlatformModelHandler), r.PUT(api.putPlatformModelHandler, NeedAdmin(true)), r.DELETE(api.deletePlatformModelHandler, NeedAdmin(true)))

	// Broadcast
	r.Handle("/broadcast", r.POST(api.addBroadcastHandler, NeedAdmin(true)), r.GET(api.getBroadcastsHandler))
	r.Handle("/broadcast/{id}", r.GET(api.getBroadcastHandler), r.PUT(api.updateBroadcastHandler, NeedAdmin(true)), r.DELETE(api.deleteBroadcastHandler, NeedAdmin(true)))
	r.Handle("/broadcast/{id}/mark", r.POST(api.postMarkAsReadBroadcastHandler))

	// Overall health
	r.Handle("/mon/status", r.GET(api.statusHandler, Auth(false)))
	r.Handle("/mon/smtp/ping", r.GET(api.smtpPingHandler, Auth(true)))
	r.Handle("/mon/version", r.GET(VersionHandler, Auth(false)))
	r.Handle("/mon/db/migrate", r.GET(api.getMonDBStatusMigrateHandler, NeedAdmin(true)))
	r.Handle("/mon/metrics", r.GET(observability.StatsHandler, Auth(false)))
	r.Handle("/mon/errors/{uuid}", r.GET(api.getErrorHandler, NeedAdmin(true)))
	r.Handle("/mon/panic/{uuid}", r.GET(api.getPanicDumpHandler, Auth(false)))

	r.Handle("/ui/navbar", r.GET(api.getNavbarHandler))
	r.Handle("/ui/project/{key}/application/{permApplicationName}/overview", r.GET(api.getApplicationOverviewHandler))

	// Import As Code
	r.Handle("/import/{permProjectKey}", r.POST(api.postImportAsCodeHandler))
	r.Handle("/import/{permProjectKey}/{uuid}", r.GET(api.getImportAsCodeHandler))
	r.Handle("/import/{permProjectKey}/{uuid}/perform", r.POST(api.postPerformImportAsCodeHandler))

	// Bookmarks
	r.Handle("/bookmarks", r.GET(api.getBookmarksHandler))

	// Project
	r.Handle("/project", r.GET(api.getProjectsHandler, AllowProvider(true), EnableTracing()), r.POST(api.addProjectHandler))
	r.Handle("/project/{permProjectKey}", r.GET(api.getProjectHandler), r.PUT(api.updateProjectHandler), r.DELETE(api.deleteProjectHandler))
	r.Handle("/project/{permProjectKey}/labels", r.PUT(api.putProjectLabelsHandler))
	r.Handle("/project/{permProjectKey}/group", r.POST(api.addGroupInProjectHandler))
	r.Handle("/project/{permProjectKey}/group/import", r.POST(api.importGroupsInProjectHandler, DEPRECATED))
	r.Handle("/project/{permProjectKey}/group/{group}", r.PUT(api.updateGroupRoleOnProjectHandler), r.DELETE(api.deleteGroupFromProjectHandler))
	r.Handle("/project/{permProjectKey}/variable", r.GET(api.getVariablesInProjectHandler))
	r.Handle("/project/{permProjectKey}/encrypt", r.POST(api.postEncryptVariableHandler))
	r.Handle("/project/{key}/variable/audit", r.GET(api.getVariablesAuditInProjectnHandler))
	r.Handle("/project/{permProjectKey}/variable/{name}", r.GET(api.getVariableInProjectHandler), r.POST(api.addVariableInProjectHandler), r.PUT(api.updateVariableInProjectHandler), r.DELETE(api.deleteVariableFromProjectHandler))
	r.Handle("/project/{permProjectKey}/variable/{name}/audit", r.GET(api.getVariableAuditInProjectHandler))
	r.Handle("/project/{permProjectKey}/applications", r.GET(api.getApplicationsHandler, AllowProvider(true)), r.POST(api.addApplicationHandler))
	r.Handle("/project/{permProjectKey}/platforms", r.GET(api.getProjectPlatformsHandler), r.POST(api.postProjectPlatformHandler))
	r.Handle("/project/{permProjectKey}/platforms/{platformName}", r.GET(api.getProjectPlatformHandler, AllowServices(true)), r.PUT(api.putProjectPlatformHandler), r.DELETE(api.deleteProjectPlatformHandler))
	r.Handle("/project/{permProjectKey}/notifications", r.GET(api.getProjectNotificationsHandler, DEPRECATED))
	r.Handle("/project/{permProjectKey}/all/keys", r.GET(api.getAllKeysProjectHandler))
	r.Handle("/project/{permProjectKey}/keys", r.GET(api.getKeysInProjectHandler), r.POST(api.addKeyInProjectHandler))
	r.Handle("/project/{permProjectKey}/keys/{name}", r.DELETE(api.deleteKeyInProjectHandler))
	// Import Application
	r.Handle("/project/{permProjectKey}/import/application", r.POST(api.postApplicationImportHandler))
	// Export Application
	r.Handle("/project/{key}/export/application/{permApplicationName}", r.GET(api.getApplicationExportHandler))

	r.Handle("/warning/{permProjectKey}", r.GET(api.getWarningsHandler))
	r.Handle("/warning/{permProjectKey}/{hash}", r.PUT(api.putWarningsHandler))

	// Application
	r.Handle("/project/{key}/application/{permApplicationName}", r.GET(api.getApplicationHandler), r.PUT(api.updateApplicationHandler), r.DELETE(api.deleteApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/metrics/{metricName}", r.GET(api.getApplicationMetricHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/keys", r.GET(api.getKeysInApplicationHandler), r.POST(api.addKeyInApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/keys/{name}", r.DELETE(api.deleteKeyInApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/vcsinfos", r.GET(api.getApplicationVCSInfosHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/clone", r.POST(api.cloneApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/group", r.POST(api.addGroupInApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/group/import", r.POST(api.importGroupsInApplicationHandler, DEPRECATED))
	r.Handle("/project/{key}/application/{permApplicationName}/group/{group}", r.PUT(api.updateGroupRoleOnApplicationHandler), r.DELETE(api.deleteGroupFromApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/variable", r.GET(api.getVariablesInApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/variable/audit", r.GET(api.getVariablesAuditInApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/variable/{name}", r.GET(api.getVariableInApplicationHandler), r.POST(api.addVariableInApplicationHandler), r.PUT(api.updateVariableInApplicationHandler), r.DELETE(api.deleteVariableFromApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/variable/{name}/audit", r.GET(api.getVariableAuditInApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/vulnerability/{id}", r.POST(api.postVulnerabilityHandler))
	// Application deployment
	r.Handle("/project/{key}/application/{permApplicationName}/deployment/config/{platform}", r.POST(api.postApplicationDeploymentStrategyConfigHandler, AllowProvider(true)), r.GET(api.getApplicationDeploymentStrategyConfigHandler), r.DELETE(api.deleteApplicationDeploymentStrategyConfigHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/deployment/config", r.GET(api.getApplicationDeploymentStrategiesConfigHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/metadata/{metadata}", r.POST(api.postApplicationMetadataHandler, AllowProvider(true)))

	// Pipeline
	r.Handle("/project/{permProjectKey}/pipeline", r.GET(api.getPipelinesHandler), r.POST(api.addPipelineHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/group", r.POST(api.addGroupInPipelineHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/group/import", r.POST(api.importGroupsInPipelineHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/group/{group}", r.PUT(api.updateGroupRoleOnPipelineHandler), r.DELETE(api.deleteGroupFromPipelineHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/parameter", r.GET(api.getParametersInPipelineHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/parameter/{name}", r.POST(api.addParameterInPipelineHandler), r.PUT(api.updateParameterInPipelineHandler), r.DELETE(api.deleteParameterFromPipelineHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}", r.GET(api.getPipelineHandler), r.PUT(api.updatePipelineHandler), r.DELETE(api.deletePipelineHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/rollback/{auditID}", r.POST(api.postPipelineRollbackHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/audits", r.GET(api.getPipelineAuditHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage", r.POST(api.addStageHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/move", r.POST(api.moveStageHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}", r.GET(api.getStageHandler), r.PUT(api.updateStageHandler), r.DELETE(api.deleteStageHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/job", r.POST(api.addJobToStageHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/job/{jobID}", r.PUT(api.updateJobHandler), r.DELETE(api.deleteJobHandler))

	// Preview pipeline
	r.Handle("/project/{permProjectKey}/preview/pipeline", r.POST(api.postPipelinePreviewHandler))
	// Import pipeline
	r.Handle("/project/{permProjectKey}/import/pipeline", r.POST(api.importPipelineHandler))
	// Import pipeline (ONLY USE FOR UI)
	r.Handle("/project/{key}/import/pipeline/{permPipelineKey}", r.PUT(api.putImportPipelineHandler))
	// Export pipeline
	r.Handle("/project/{key}/export/pipeline/{permPipelineKey}", r.GET(api.getPipelineExportHandler))

	// Workflows
	r.Handle("/workflow/artifact/{hash}", r.GET(api.downloadworkflowArtifactDirectHandler, Auth(false)))

	r.Handle("/project/{permProjectKey}/workflows", r.POST(api.postWorkflowHandler, EnableTracing()), r.GET(api.getWorkflowsHandler, AllowProvider(true), EnableTracing()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}", r.GET(api.getWorkflowHandler, AllowProvider(true), EnableTracing()), r.PUT(api.putWorkflowHandler, EnableTracing()), r.DELETE(api.deleteWorkflowHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/ascode/{uuid}", r.GET(api.getWorkflowAsCodeHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/ascode", r.POST(api.postWorkflowAsCodeHandler, EnableTracing()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/label", r.POST(api.postWorkflowLabelHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/label/{labelID}", r.DELETE(api.deleteWorkflowLabelHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/rollback/{auditID}", r.POST(api.postWorkflowRollbackHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/groups", r.POST(api.postWorkflowGroupHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/groups/{groupName}", r.PUT(api.putWorkflowGroupHandler), r.DELETE(api.deleteWorkflowGroupHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/hooks/{uuid}", r.GET(api.getWorkflowHookHandler))
	r.Handle("/project/{key}/workflow/{permWorkflowName}/node/{nodeID}/hook/model", r.GET(api.getWorkflowHookModelsHandler))
	r.Handle("/project/{key}/workflow/{permWorkflowName}/node/{nodeID}/outgoinghook/model", r.GET(api.getWorkflowOutgoingHookModelsHandler))

	// Outgoing hook model
	r.Handle("/workflow/outgoinghook/model", r.GET(api.getWorkflowOutgoingHookModelsHandler))

	// Preview workflows
	r.Handle("/project/{permProjectKey}/preview/workflows", r.POST(api.postWorkflowPreviewHandler))
	// Import workflows
	r.Handle("/project/{permProjectKey}/import/workflows", r.POST(api.postWorkflowImportHandler))
	// Import workflows (ONLY USE FOR UI EDIT AS CODE)
	r.Handle("/project/{key}/import/workflows/{permWorkflowName}", r.PUT(api.putWorkflowImportHandler))
	// Export workflows
	r.Handle("/project/{key}/export/workflows/{permWorkflowName}", r.GET(api.getWorkflowExportHandler))
	// Pull workflows
	r.Handle("/project/{key}/pull/workflows/{permWorkflowName}", r.GET(api.getWorkflowPullHandler))
	// Push workflows
	r.Handle("/project/{permProjectKey}/push/workflows", r.POST(api.postWorkflowPushHandler))

	// Workflows run
	r.Handle("/project/{permProjectKey}/runs", r.GET(api.getWorkflowAllRunsHandler, EnableTracing()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs", r.GET(api.getWorkflowRunsHandler, EnableTracing()), r.POSTEXECUTE(api.postWorkflowRunHandler, AllowServices(true), EnableTracing()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/latest", r.GET(api.getLatestWorkflowRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/tags", r.GET(api.getWorkflowRunTagsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/num", r.GET(api.getWorkflowRunNumHandler), r.POST(api.postWorkflowRunNumHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}", r.GET(api.getWorkflowRunHandler, AllowServices(true)))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/stop", r.POSTEXECUTE(api.stopWorkflowRunHandler, EnableTracing()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/vcs/resync", r.POSTEXECUTE(api.postResyncVCSWorkflowRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/resync", r.POST(api.resyncWorkflowRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/artifacts", r.GET(api.getWorkflowRunArtifactsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}", r.GET(api.getWorkflowNodeRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/stop", r.POSTEXECUTE(api.stopWorkflowNodeRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeID}/history", r.GET(api.getWorkflowNodeRunHistoryHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/{nodeName}/commits", r.GET(api.getWorkflowCommitsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/job/{runJobId}/info", r.GET(api.getWorkflowNodeRunJobSpawnInfosHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/job/{runJobId}/log/service", r.GET(api.getWorkflowNodeRunJobServiceLogsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/job/{runJobId}/step/{stepOrder}", r.GET(api.getWorkflowNodeRunJobStepHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/artifact/{artifactId}", r.GET(api.getDownloadArtifactHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/node/{nodeID}/triggers/condition", r.GET(api.getWorkflowTriggerConditionHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/release", r.POST(api.releaseApplicationWorkflowHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/hooks/{hookRunID}/callback", r.POST(api.postWorkflowJobHookCallbackHandler, AllowServices(true)))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/hooks/{hookRunID}/details", r.GET(api.getWorkflowJobHookDetailsHandler, NeedService()))

	// Environment
	r.Handle("/project/{permProjectKey}/environment", r.GET(api.getEnvironmentsHandler), r.POST(api.addEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/import", r.POST(api.importNewEnvironmentHandler, DEPRECATED))
	r.Handle("/project/{key}/environment/import/{permEnvironmentName}", r.POST(api.importIntoEnvironmentHandler, DEPRECATED))
	r.Handle("/project/{key}/environment/{permEnvironmentName}", r.GET(api.getEnvironmentHandler), r.PUT(api.updateEnvironmentHandler), r.DELETE(api.deleteEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/usage", r.GET(api.getEnvironmentUsageHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/keys", r.GET(api.getKeysInEnvironmentHandler), r.POST(api.addKeyInEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/keys/{name}", r.DELETE(api.deleteKeyInEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/clone/{cloneName}", r.POST(api.cloneEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/group", r.POST(api.addGroupInEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/groups", r.POST(api.addGroupsInEnvironmentHandler, DEPRECATED))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/group/import", r.POST(api.importGroupsInEnvironmentHandler, DEPRECATED))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/group/{group}", r.PUT(api.updateGroupRoleOnEnvironmentHandler), r.DELETE(api.deleteGroupFromEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/variable", r.GET(api.getVariablesInEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/variable/{name}", r.GET(api.getVariableInEnvironmentHandler), r.POST(api.addVariableInEnvironmentHandler), r.PUT(api.updateVariableInEnvironmentHandler), r.DELETE(api.deleteVariableFromEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/variable/{name}/audit", r.GET(api.getVariableAuditInEnvironmentHandler))

	// Import Environment
	r.Handle("/project/{permProjectKey}/import/environment", r.POST(api.postEnvironmentImportHandler))
	// Export Environment
	r.Handle("/project/{key}/export/environment/{permEnvironmentName}", r.GET(api.getEnvironmentExportHandler))

	// Artifacts
	r.Handle("/staticfiles/store", r.GET(api.getStaticFilesStoreHandler, Auth(false)))
	r.Handle("/artifact/store", r.GET(api.getArtifactsStoreHandler, Auth(false)))

	// Cache
	r.Handle("/project/{permProjectKey}/cache/{tag}", r.POSTEXECUTE(api.postPushCacheHandler, NeedWorker()), r.GET(api.getPullCacheHandler, NeedWorker()))
	r.Handle("/project/{permProjectKey}/cache/{tag}/url", r.POSTEXECUTE(api.postPushCacheWithTempURLHandler, NeedWorker()), r.GET(api.getPullCacheWithTempURLHandler, NeedWorker()))

	//Workflow queue
	r.Handle("/queue/workflows", r.GET(api.getWorkflowJobQueueHandler, EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/count", r.GET(api.countWorkflowJobQueueHandler, EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{id}/take", r.POST(api.postTakeWorkflowJobHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{id}/book", r.POST(api.postBookWorkflowJobHandler, NeedHatchery(), EnableTracing(), MaintenanceAware()), r.DELETE(api.deleteBookWorkflowJobHandler, NeedHatchery(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{id}/attempt", r.POST(api.postIncWorkflowJobAttemptHandler, NeedHatchery(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{id}/infos", r.GET(api.getWorkflowJobHandler, NeedWorker(), NeedHatchery(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/vulnerability", r.POSTEXECUTE(api.postVulnerabilityReportHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{id}/spawn/infos", r.POST(r.Asynchronous(api.postSpawnInfosWorkflowJobHandler, 1), NeedHatchery(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/result", r.POSTEXECUTE(api.postWorkflowJobResultHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/log", r.POSTEXECUTE(r.Asynchronous(api.postWorkflowJobLogsHandler, 1), NeedWorker(), MaintenanceAware()))
	r.Handle("/queue/workflows/log/service", r.POSTEXECUTE(r.Asynchronous(api.postWorkflowJobServiceLogsHandler, 1), NeedHatchery(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/coverage", r.POSTEXECUTE(api.postWorkflowJobCoverageResultsHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/test", r.POSTEXECUTE(api.postWorkflowJobTestsResultsHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/tag", r.POSTEXECUTE(api.postWorkflowJobTagsHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/variable", r.POSTEXECUTE(api.postWorkflowJobVariableHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/step", r.POSTEXECUTE(api.postWorkflowJobStepStatusHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/artifact/{ref}", r.POSTEXECUTE(api.postWorkflowJobArtifactHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/artifact/{ref}/url", r.POSTEXECUTE(api.postWorkflowJobArtifacWithTempURLHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/artifact/{ref}/url/callback", r.POSTEXECUTE(api.postWorkflowJobArtifactWithTempURLCallbackHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/staticfiles/{name}", r.POSTEXECUTE(api.postWorkflowJobStaticFilesHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/staticfiles/{name}/url", r.POSTEXECUTE(api.postWorkflowJobStaticFilesWithTempURLHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))
	r.Handle("/queue/workflows/{permID}/staticfiles/{name}/url/callback", r.POSTEXECUTE(api.postWorkflowJobStaticFilesWithTempURLCallbackHandler, NeedWorker(), EnableTracing(), MaintenanceAware()))

	r.Handle("/variable/type", r.GET(api.getVariableTypeHandler))
	r.Handle("/parameter/type", r.GET(api.getParameterTypeHandler))
	r.Handle("/pipeline/type", r.GET(api.getPipelineTypeHandler))
	r.Handle("/notification/type", r.GET(api.getUserNotificationTypeHandler))
	r.Handle("/notification/state", r.GET(api.getUserNotificationStateValueHandler))

	// RepositoriesManager
	r.Handle("/repositories_manager", r.GET(api.getRepositoriesManagerHandler))
	r.Handle("/repositories_manager/oauth2/callback", r.GET(api.repositoriesManagerOAuthCallbackHandler, Auth(false)))
	// RepositoriesManager for projects
	r.Handle("/project/{permProjectKey}/repositories_manager", r.GET(api.getRepositoriesManagerForProjectHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/authorize", r.POST(api.repositoriesManagerAuthorizeHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/authorize/callback", r.POST(api.repositoriesManagerAuthorizeCallbackHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}", r.DELETE(api.deleteRepositoriesManagerHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/repo", r.GET(api.getRepoFromRepositoriesManagerHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/repos", r.GET(api.getReposFromRepositoriesManagerHandler))

	// RepositoriesManager for applications
	r.Handle("/project/{key}/repositories_manager/{name}/application/{permApplicationName}/attach", r.POST(api.attachRepositoriesManagerHandler))
	r.Handle("/project/{key}/repositories_manager/{name}/application/{permApplicationName}/detach", r.POST(api.detachRepositoriesManagerHandler))

	// Suggest
	r.Handle("/suggest/variable/{permProjectKey}", r.GET(api.getVariablesHandler))

	//Requirements
	r.Handle("/requirement/types", r.GET(api.getRequirementTypesHandler))
	r.Handle("/requirement/types/{type}", r.GET(api.getRequirementTypeValuesHandler))

	//Requirements
	r.Handle("/requirement/types", r.GET(api.getRequirementTypesHandler))
	r.Handle("/requirement/types/{type}", r.GET(api.getRequirementTypeValuesHandler))

	// config
	r.Handle("/config/user", r.GET(api.ConfigUserHandler, Auth(true)))

	// Users
	r.Handle("/user", r.GET(api.getUsersHandler))
	r.Handle("/user/logged", r.GET(api.getUserLoggedHandler, Auth(false)))
	r.Handle("/user/me", r.GET(api.getUserLoggedHandler, Auth(false), DEPRECATED))
	r.Handle("/user/favorite", r.POST(api.postUserFavoriteHandler))
	r.Handle("/user/timeline", r.GET(api.getTimelineHandler))
	r.Handle("/user/timeline/filter", r.GET(api.getTimelineFilterHandler), r.POST(api.postTimelineFilterHandler))
	r.Handle("/user/token", r.GET(api.getUserTokenListHandler))
	r.Handle("/user/token/{token}", r.GET(api.getUserTokenHandler))
	r.Handle("/user/signup", r.POST(api.addUserHandler, Auth(false)))
	r.Handle("/user/import", r.POST(api.importUsersHandler, NeedAdmin(true)))
	r.Handle("/user/{username}", r.GET(api.getUserHandler, NeedUsernameOrAdmin(true)), r.PUT(api.updateUserHandler, NeedUsernameOrAdmin(true)), r.DELETE(api.deleteUserHandler, NeedUsernameOrAdmin(true)))
	r.Handle("/user/{username}/groups", r.GET(api.getUserGroupsHandler, NeedUsernameOrAdmin(true)))
	r.Handle("/user/{username}/confirm/{token}", r.GET(api.confirmUserHandler, Auth(false)))
	r.Handle("/user/{username}/reset", r.POST(api.resetUserHandler, Auth(false)))
	r.Handle("/auth/mode", r.GET(api.authModeHandler, Auth(false)))

	// Workers
	r.Handle("/worker", r.GET(api.getWorkersHandler), r.POST(api.registerWorkerHandler, Auth(false)))
	r.Handle("/worker/refresh", r.POST(api.refreshWorkerHandler))
	r.Handle("/worker/checking", r.POST(api.workerCheckingHandler))
	r.Handle("/worker/waiting", r.POST(api.workerWaitingHandler))
	r.Handle("/worker/unregister", r.POST(api.unregisterWorkerHandler))
	r.Handle("/worker/{id}/disable", r.POST(api.disableWorkerHandler))

	// Worker models
	r.Handle("/worker/model", r.POST(api.addWorkerModelHandler), r.GET(api.getWorkerModelsHandler))
	r.Handle("/worker/model/import", r.POST(api.postWorkerModelImportHandler))
	r.Handle("/worker/model/pattern", r.POST(api.postAddWorkerModelPatternHandler, NeedAdmin(true)), r.GET(api.getWorkerModelPatternsHandler))
	r.Handle("/worker/model/pattern/{type}/{name}", r.GET(api.getWorkerModelPatternHandler), r.PUT(api.putWorkerModelPatternHandler, NeedAdmin(true)), r.DELETE(api.deleteWorkerModelPatternHandler, NeedAdmin(true)))
	r.Handle("/worker/model/book/{permModelID}", r.PUT(api.bookWorkerModelHandler, NeedHatchery()))
	r.Handle("/worker/model/error/{permModelID}", r.PUT(api.spawnErrorWorkerModelHandler, NeedHatchery()))
	r.Handle("/worker/model/enabled", r.GET(api.getWorkerModelsEnabledHandler, NeedHatchery()))
	r.Handle("/worker/model/type", r.GET(api.getWorkerModelTypesHandler))
	r.Handle("/worker/model/communication", r.GET(api.getWorkerModelCommunicationsHandler))
	r.Handle("/worker/model/{permModelID}", r.PUT(api.updateWorkerModelHandler), r.DELETE(api.deleteWorkerModelHandler))
	r.Handle("/worker/model/{permModelID}/export", r.GET(api.getWorkerModelExportHandler))
	r.Handle("/worker/model/{modelID}/usage", r.GET(api.getWorkerModelUsageHandler))
	r.Handle("/worker/model/capability/type", r.GET(api.getRequirementTypesHandler))

	// Workflows
	r.Handle("/workflow/hook", r.GET(api.getWorkflowHooksHandler, NeedService()))
	r.Handle("/workflow/hook/model/{model}", r.GET(api.getWorkflowHookModelHandler), r.POST(api.postWorkflowHookModelHandler, NeedAdmin(true)), r.PUT(api.putWorkflowHookModelHandler, NeedAdmin(true)))

	// SSE
	r.Handle("/events", r.GET(api.eventsBroker.ServeHTTP))

	// Feature
	r.Handle("/feature/clean", r.POST(api.cleanFeatureHandler, NeedToken("X-Izanami-Token", api.Config.Features.Izanami.Token), Auth(false)))

	// Engine µServices
	r.Handle("/services/register", r.POST(api.postServiceRegisterHandler, Auth(false)))
	r.Handle("/services/{type}", r.GET(api.getExternalServiceHandler, NeedWorker()))

	// Templates
	r.Handle("/template", r.GET(api.getTemplatesHandler), r.POST(api.postTemplateHandler))
	r.Handle("/template/push", r.POST(api.postTemplatePushHandler))
	r.Handle("/template/{id}", r.GET(api.getTemplateHandler))
	r.Handle("/template/{groupName}/{templateSlug}", r.GET(api.getTemplateHandler), r.PUT(api.putTemplateHandler), r.DELETE(api.deleteTemplateHandler))
	r.Handle("/template/{groupName}/{templateSlug}/pull", r.POST(api.postTemplatePullHandler))
	r.Handle("/template/{groupName}/{templateSlug}/apply", r.POST(api.postTemplateApplyHandler))
	r.Handle("/template/{groupName}/{templateSlug}/bulk", r.POST(api.postTemplateBulkHandler))
	r.Handle("/template/{groupName}/{templateSlug}/bulk/{bulkID}", r.GET(api.getTemplateBulkHandler))
	r.Handle("/template/{groupName}/{templateSlug}/instance", r.GET(api.getTemplateInstancesHandler))
	r.Handle("/template/{groupName}/{templateSlug}/audit", r.GET(api.getTemplateAuditsHandler))
	r.Handle("/template/{groupName}/{templateSlug}/usage", r.GET(api.getTemplateUsageHandler))
	r.Handle("/project/{key}/workflow/{permWorkflowName}/templateInstance", r.GET(api.getTemplateInstanceHandler))

	//Not Found handler
	r.Mux.NotFoundHandler = http.HandlerFunc(NotFoundHandler)
}
