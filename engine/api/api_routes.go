package api

import (
	"net/http"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

type HandlerScope []sdk.AuthConsumerScope

// Scope set for handler. If multiple scopes are given, one should match consumer scopes.
func Scope(s ...sdk.AuthConsumerScope) HandlerScope {
	return HandlerScope(s)
}

var (
	ScopeNone = func() HandlerScope { return nil }
)

// InitRouter initializes the router and all the routes
func (api *API) InitRouter() {
	api.Router.URL = api.Config.URL.API
	api.Router.SetHeaderFunc = service.DefaultHeaders
	api.Router.Middlewares = append(api.Router.Middlewares, api.tracingMiddleware, api.jwtMiddleware)
	api.Router.DefaultAuthMiddleware = api.authMiddleware
	api.Router.PostAuthMiddlewares = append(api.Router.PostAuthMiddlewares, api.xsrfMiddleware, api.maintenanceMiddleware)
	api.Router.PostMiddlewares = append(api.Router.PostMiddlewares, TracingPostMiddleware)

	r := api.Router

	// Auth
	r.Handle("/auth/driver", ScopeNone(), r.GET(api.getAuthDriversHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/auth/me", ScopeNone(), r.GET(api.getAuthMe))
	r.Handle("/auth/scope", ScopeNone(), r.GET(api.getAuthScopesHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/auth/consumer/local/signup", ScopeNone(), r.POST(api.postAuthLocalSignupHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/auth/consumer/local/signin", ScopeNone(), r.POST(api.postAuthLocalSigninHandler, service.OverrideAuth(service.NoAuthMiddleware), MaintenanceAware()))
	r.Handle("/auth/consumer/local/verify", ScopeNone(), r.POST(api.postAuthLocalVerifyHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/auth/consumer/local/askReset", ScopeNone(), r.POST(api.postAuthLocalAskResetHandler, service.OverrideAuth(service.NoAuthMiddleware), MaintenanceAware()))
	r.Handle("/auth/consumer/local/reset", ScopeNone(), r.POST(api.postAuthLocalResetHandler, service.OverrideAuth(service.NoAuthMiddleware), MaintenanceAware()))
	r.Handle("/auth/consumer/builtin/signin", ScopeNone(), r.POST(api.postAuthBuiltinSigninHandler, service.OverrideAuth(service.NoAuthMiddleware), MaintenanceAware()))
	r.Handle("/auth/consumer/worker/signin", ScopeNone(), r.POST(api.postRegisterWorkerHandler, service.OverrideAuth(service.NoAuthMiddleware), MaintenanceAware()))
	r.Handle("/auth/consumer/worker/signout", ScopeNone(), r.POST(api.postUnregisterWorkerHandler, MaintenanceAware()))
	r.Handle("/auth/consumer/{consumerType}/askSignin", ScopeNone(), r.GET(api.getAuthAskSigninHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/auth/consumer/{consumerType}/signin", Scope(sdk.AuthConsumerScopeAccessToken), r.POST(api.postAuthSigninHandler, service.OverrideAuth(api.authOptionalMiddleware), MaintenanceAware()))
	r.Handle("/auth/consumer/{consumerType}/detach", Scope(sdk.AuthConsumerScopeAccessToken), r.POST(api.postAuthDetachHandler))
	r.Handle("/auth/consumer/signout", ScopeNone(), r.POST(api.postAuthSignoutHandler))
	r.Handle("/auth/session/{sessionID}", ScopeNone(), r.GET(api.getAuthSession))

	// Action
	r.Handle("/action", Scope(sdk.AuthConsumerScopeAction), r.GET(api.getActionsHandler), r.POST(api.postActionHandler))
	r.Handle("/action/import", Scope(sdk.AuthConsumerScopeAction), r.POST(api.importActionHandler))
	r.Handle("/action/{permGroupName}/{permActionName}", Scope(sdk.AuthConsumerScopeAction), r.GET(api.getActionHandler), r.PUT(api.putActionHandler), r.DELETE(api.deleteActionHandler))
	r.Handle("/action/{permGroupName}/{permActionName}/usage", Scope(sdk.AuthConsumerScopeAction), r.GET(api.getActionUsageHandler))
	r.Handle("/action/{permGroupName}/{permActionName}/export", Scope(sdk.AuthConsumerScopeAction), r.GET(api.getActionExportHandler))
	r.Handle("/action/{permGroupName}/{permActionName}/audit", Scope(sdk.AuthConsumerScopeAction), r.GET(api.getActionAuditHandler))
	r.Handle("/action/{permGroupName}/{permActionName}/audit/{auditID}/rollback", Scope(sdk.AuthConsumerScopeAction), r.POST(api.postActionAuditRollbackHandler))
	r.Handle("/action/requirement", Scope(sdk.AuthConsumerScopeAction), r.GET(api.getActionsRequirements, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/project/{permProjectKey}/action", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getActionsForProjectHandler))
	r.Handle("/group/{permGroupName}/action", Scope(sdk.AuthConsumerScopeGroup), r.GET(api.getActionsForGroupHandler))
	r.Handle("/actionBuiltin", ScopeNone(), r.GET(api.getActionsBuiltinHandler))
	r.Handle("/actionBuiltin/{permActionBuiltinName}", ScopeNone(), r.GET(api.getActionBuiltinHandler))
	r.Handle("/actionBuiltin/{permActionBuiltinName}/usage", Scope(sdk.AuthConsumerScopeAdmin), r.GET(api.getActionBuiltinUsageHandler))

	// Admin
	r.Handle("/admin/maintenance", Scope(sdk.AuthConsumerScopeAdmin), r.POST(api.postMaintenanceHandler, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/admin/cds/migration", Scope(sdk.AuthConsumerScopeAdmin), r.GET(api.getAdminMigrationsHandler, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/admin/cds/migration/{id}/cancel", Scope(sdk.AuthConsumerScopeAdmin), r.POST(api.postAdminMigrationCancelHandler, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/admin/cds/migration/{id}/todo", Scope(sdk.AuthConsumerScopeAdmin), r.POST(api.postAdminMigrationTodoHandler, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/admin/database/migration/delete/{id}", Scope(sdk.AuthConsumerScopeAdmin), r.DELETE(api.deleteDatabaseMigrationHandler, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/admin/database/migration/unlock/{id}", Scope(sdk.AuthConsumerScopeAdmin), r.POST(api.postDatabaseMigrationUnlockedHandler, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/admin/database/migration", Scope(sdk.AuthConsumerScopeAdmin), r.GET(api.getDatabaseMigrationHandler, service.OverrideAuth(api.authAdminMiddleware)))

	r.Handle("/admin/debug/profiles", Scope(sdk.AuthConsumerScopeAdmin), r.GET(api.getDebugProfilesHandler, service.OverrideAuth(api.authMaintainerMiddleware)))
	r.Handle("/admin/debug/goroutines", Scope(sdk.AuthConsumerScopeAdmin), r.GET(api.getDebugGoroutinesHandler, service.OverrideAuth(api.authMaintainerMiddleware)))
	r.Handle("/admin/debug/trace", Scope(sdk.AuthConsumerScopeAdmin), r.POST(api.getTraceHandler, service.OverrideAuth(api.authAdminMiddleware)), r.GET(api.getTraceHandler, service.OverrideAuth(api.authMaintainerMiddleware)))
	r.Handle("/admin/debug/cpu", Scope(sdk.AuthConsumerScopeAdmin), r.POST(api.getCPUProfileHandler, service.OverrideAuth(api.authAdminMiddleware)), r.GET(api.getCPUProfileHandler, service.OverrideAuth(api.authMaintainerMiddleware)))
	r.Handle("/admin/debug/{name}", Scope(sdk.AuthConsumerScopeAdmin), r.POST(api.getProfileHandler, service.OverrideAuth(api.authAdminMiddleware)), r.GET(api.getProfileHandler, service.OverrideAuth(api.authMaintainerMiddleware)))

	r.Handle("/admin/plugin", Scope(sdk.AuthConsumerScopeAdmin), r.POST(api.postGRPCluginHandler, service.OverrideAuth(api.authAdminMiddleware)), r.GET(api.getAllGRPCluginHandler, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/admin/plugin/{name}", Scope(sdk.AuthConsumerScopeAdmin), r.GET(api.getGRPCluginHandler, service.OverrideAuth(api.authAdminMiddleware)), r.PUT(api.putGRPCluginHandler, service.OverrideAuth(api.authAdminMiddleware)), r.DELETE(api.deleteGRPCluginHandler, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/admin/plugin/{name}/binary", Scope(sdk.AuthConsumerScopeAdmin), r.POST(api.postGRPCluginBinaryHandler, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/admin/plugin/{name}/binary/{os}/{arch}", Scope(sdk.AuthConsumerScopeAdmin), r.GET(api.getGRPCluginBinaryHandler, service.OverrideAuth(service.NoAuthMiddleware)), r.DELETE(api.deleteGRPCluginBinaryHandler, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/admin/plugin/{name}/binary/{os}/{arch}/infos", Scope(sdk.AuthConsumerScopeAdmin), r.GET(api.getGRPCluginBinaryInfosHandler))

	// Admin service
	r.Handle("/admin/service/{name}", Scope(sdk.AuthConsumerScopeAdmin), r.GET(api.getAdminServiceHandler, service.OverrideAuth(api.authMaintainerMiddleware)), r.DELETE(api.deleteAdminServiceHandler, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/admin/services", Scope(sdk.AuthConsumerScopeAdmin), r.GET(api.getAdminServicesHandler, service.OverrideAuth(api.authMaintainerMiddleware)))
	r.Handle("/admin/services/call", Scope(sdk.AuthConsumerScopeAdmin), r.GET(api.getAdminServiceCallHandler, service.OverrideAuth(api.authMaintainerMiddleware)), r.POST(api.postAdminServiceCallHandler, service.OverrideAuth(api.authAdminMiddleware)), r.PUT(api.putAdminServiceCallHandler, service.OverrideAuth(api.authAdminMiddleware)), r.DELETE(api.deleteAdminServiceCallHandler, service.OverrideAuth(api.authAdminMiddleware)))

	// Admin database
	r.Handle("/admin/database/signature", Scope(sdk.AuthConsumerScopeAdmin), r.GET(api.getAdminDatabaseSignatureResume, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/admin/database/signature/{entity}/roll/{pk}", Scope(sdk.AuthConsumerScopeAdmin), r.POST(api.postAdminDatabaseSignatureRollEntityByPrimaryKey, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/admin/database/signature/{entity}/{signer}", Scope(sdk.AuthConsumerScopeAdmin), r.GET(api.getAdminDatabaseSignatureTuplesBySigner, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/admin/database/encryption", Scope(sdk.AuthConsumerScopeAdmin), r.GET(api.getAdminDatabaseEncryptedEntities, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/admin/database/encryption/{entity}", Scope(sdk.AuthConsumerScopeAdmin), r.GET(api.getAdminDatabaseEncryptedTuplesByEntity, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/admin/database/encryption/{entity}/roll/{pk}", Scope(sdk.AuthConsumerScopeAdmin), r.POST(api.postAdminDatabaseRollEncryptedEntityByPrimaryKey, service.OverrideAuth(api.authAdminMiddleware)))

	// Feature flipping
	r.Handle("/admin/features", Scope(sdk.AuthConsumerScopeAdmin), r.GET(api.getAdminFeatureFlipping, service.OverrideAuth(api.authAdminMiddleware)), r.POST(api.postAdminFeatureFlipping, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/admin/features/{name}", Scope(sdk.AuthConsumerScopeAdmin), r.GET(api.getAdminFeatureFlippingByName, service.OverrideAuth(api.authAdminMiddleware)), r.PUT(api.putAdminFeatureFlipping, service.OverrideAuth(api.authAdminMiddleware)), r.DELETE(api.deleteAdminFeatureFlipping, service.OverrideAuth(api.authAdminMiddleware)))

	// Download file
	r.Handle("/download", ScopeNone(), r.GET(api.downloadsHandler))
	r.Handle("/download/plugin/{name}/binary/{os}/{arch}", ScopeNone(), r.GET(api.getGRPCluginBinaryHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/download/plugin/{name}/binary/{os}/{arch}/infos", ScopeNone(), r.GET(api.getGRPCluginBinaryInfosHandler))

	r.Handle("/download/{name}/{os}/{arch}", ScopeNone(), r.GET(api.downloadHandler, service.OverrideAuth(service.NoAuthMiddleware)))

	// feature
	r.Handle("/feature/enabled/{name}", ScopeNone(), r.POST(api.isFeatureEnabledHandler))

	// Group
	r.Handle("/group", Scope(sdk.AuthConsumerScopeGroup), r.GET(api.getGroupsHandler), r.POST(api.postGroupHandler))
	r.Handle("/group/{permGroupName}", Scope(sdk.AuthConsumerScopeGroup), r.GET(api.getGroupHandler), r.PUT(api.putGroupHandler), r.DELETE(api.deleteGroupHandler))
	r.Handle("/group/{permGroupName}/user", Scope(sdk.AuthConsumerScopeGroup), r.POST(api.postGroupUserHandler))
	r.Handle("/group/{permGroupName}/user/{username}", Scope(sdk.AuthConsumerScopeGroup), r.PUT(api.putGroupUserHandler), r.DELETE(api.deleteGroupUserHandler))

	// Hooks
	r.Handle("/hook/{uuid}/workflow/{workflowID}/vcsevent/{vcsServer}", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getHookPollingVCSEvents))

	// Integration
	r.Handle("/integration/models", ScopeNone(), r.GET(api.getIntegrationModelsHandler), r.POST(api.postIntegrationModelHandler, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/integration/models/{name}", ScopeNone(), r.GET(api.getIntegrationModelHandler), r.PUT(api.putIntegrationModelHandler, service.OverrideAuth(api.authAdminMiddleware)), r.DELETE(api.deleteIntegrationModelHandler, service.OverrideAuth(api.authAdminMiddleware)))

	// Broadcast
	r.Handle("/broadcast", ScopeNone(), r.POST(api.addBroadcastHandler, service.OverrideAuth(api.authAdminMiddleware)), r.GET(api.getBroadcastsHandler))
	r.Handle("/broadcast/{id}", ScopeNone(), r.GET(api.getBroadcastHandler), r.PUT(api.updateBroadcastHandler, service.OverrideAuth(api.authAdminMiddleware)), r.DELETE(api.deleteBroadcastHandler, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/broadcast/{id}/mark", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postMarkAsReadBroadcastHandler))

	// Overall health
	r.Handle("/mon/status", ScopeNone(), r.GET(api.statusHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/version", ScopeNone(), r.GET(service.VersionHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/db/migrate", ScopeNone(), r.GET(api.getMonDBStatusMigrateHandler, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/mon/metrics", ScopeNone(), r.GET(service.GetPrometheustMetricsHandler(api), service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/metrics/all", ScopeNone(), r.GET(service.GetMetricsHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.HandlePrefix("/mon/metrics/detail/", ScopeNone(), r.GET(service.GetMetricHandler("/mon/metrics/detail/"), service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/errors/{uuid}", ScopeNone(), r.GET(api.getErrorHandler, service.OverrideAuth(api.authAdminMiddleware)))

	r.Handle("/help", ScopeNone(), r.GET(api.getHelpHandler, service.OverrideAuth(service.NoAuthMiddleware)))

	r.Handle("/ui/navbar", ScopeNone(), r.GET(api.getNavbarHandler))
	r.Handle("/ui/project/{permProjectKey}/application/{applicationName}/overview", ScopeNone(), r.GET(api.getApplicationOverviewHandler))

	// Import As Code
	r.Handle("/import/{permProjectKey}", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postImportAsCodeHandler))
	r.Handle("/import/{permProjectKey}/{uuid}/perform", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postPerformImportAsCodeHandler))

	// Bookmarks
	r.Handle("/bookmarks", ScopeNone(), r.GET(api.getBookmarksHandler))

	// Project
	r.Handle("/project", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getProjectsHandler), r.POST(api.postProjectHandler))
	r.Handle("/project/{permProjectKey}", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getProjectHandler), r.PUT(api.updateProjectHandler), r.DELETE(api.deleteProjectHandler))
	r.Handle("/project/{permProjectKey}/labels", Scope(sdk.AuthConsumerScopeProject), r.PUT(api.putProjectLabelsHandler))
	r.Handle("/project/{permProjectKey}/group", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postGroupInProjectHandler))
	r.Handle("/project/{permProjectKey}/group/import", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postImportGroupsInProjectHandler))
	r.Handle("/project/{permProjectKey}/group/{groupName}", Scope(sdk.AuthConsumerScopeProject), r.PUT(api.putGroupRoleOnProjectHandler), r.DELETE(api.deleteGroupFromProjectHandler))
	r.Handle("/project/{permProjectKey}/variable", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getVariablesInProjectHandler))
	r.Handle("/project/{permProjectKey}/encrypt", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postEncryptVariableHandler))
	r.Handle("/project/{permProjectKey}/encrypt/list", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getListEncryptVariableHandler))
	r.Handle("/project/{permProjectKey}/variable/audit", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getVariablesAuditInProjectnHandler))
	r.Handle("/project/{permProjectKey}/variable/{name}", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getVariableInProjectHandler), r.POST(api.addVariableInProjectHandler), r.PUT(api.updateVariableInProjectHandler), r.DELETE(api.deleteVariableFromProjectHandler))
	r.Handle("/project/{permProjectKey}/variable/{name}/audit", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getVariableAuditInProjectHandler))
	r.Handle("/project/{permProjectKey}/applications", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getApplicationsHandler), r.POST(api.addApplicationHandler))
	r.Handle("/project/{permProjectKey}/integrations", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getProjectIntegrationsHandler), r.POST(api.postProjectIntegrationHandler))
	r.Handle("/project/{permProjectKey}/integrations/{integrationName}", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getProjectIntegrationHandler), r.PUT(api.putProjectIntegrationHandler), r.DELETE(api.deleteProjectIntegrationHandler))
	r.Handle("/project/{permProjectKey}/notifications", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getProjectNotificationsHandler, DEPRECATED))
	r.Handle("/project/{permProjectKey}/keys", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getKeysInProjectHandler), r.POST(api.addKeyInProjectHandler))
	r.Handle("/project/{permProjectKey}/keys/{name}", Scope(sdk.AuthConsumerScopeProject), r.DELETE(api.deleteKeyInProjectHandler))

	// Import Application
	r.Handle("/project/{permProjectKey}/import/application", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postApplicationImportHandler))
	// Export Application
	r.Handle("/project/{permProjectKey}/export/application/{applicationName}", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getApplicationExportHandler))

	// Application
	r.Handle("/project/{permProjectKey}/application/{applicationName}", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getApplicationHandler), r.PUT(api.updateApplicationHandler), r.DELETE(api.deleteApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/ascode", Scope(sdk.AuthConsumerScopeProject), r.PUT(api.updateAsCodeApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/metrics/{metricName}", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getApplicationMetricHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/keys", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getKeysInApplicationHandler), r.POST(api.addKeyInApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/keys/{name}", Scope(sdk.AuthConsumerScopeProject), r.DELETE(api.deleteKeyInApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/vcsinfos", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getApplicationVCSInfosHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/clone", Scope(sdk.AuthConsumerScopeProject), r.POST(api.cloneApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/variable", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getVariablesInApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/variable/audit", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getVariablesAuditInApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/variable/{name}", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getVariableInApplicationHandler), r.POST(api.addVariableInApplicationHandler), r.PUT(api.updateVariableInApplicationHandler), r.DELETE(api.deleteVariableFromApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/variable/{name}/audit", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getVariableAuditInApplicationHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/vulnerability/{id}", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postVulnerabilityHandler))
	// Application deployment
	r.Handle("/project/{permProjectKey}/application/{applicationName}/deployment/config/{integration}", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postApplicationDeploymentStrategyConfigHandler), r.GET(api.getApplicationDeploymentStrategyConfigHandler), r.DELETE(api.deleteApplicationDeploymentStrategyConfigHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/deployment/config", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getApplicationDeploymentStrategiesConfigHandler))
	r.Handle("/project/{permProjectKey}/application/{applicationName}/metadata/{metadata}", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postApplicationMetadataHandler))

	// Pipeline
	r.Handle("/project/{permProjectKey}/pipeline", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getPipelinesHandler), r.POST(api.addPipelineHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/parameter", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getParametersInPipelineHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/parameter/{name}", Scope(sdk.AuthConsumerScopeProject), r.POST(api.addParameterInPipelineHandler), r.PUT(api.updateParameterInPipelineHandler), r.DELETE(api.deleteParameterFromPipelineHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getPipelineHandler), r.PUT(api.updatePipelineHandler), r.DELETE(api.deletePipelineHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/ascode", Scope(sdk.AuthConsumerScopeProject), r.PUT(api.updateAsCodePipelineHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/rollback/{auditID}", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postPipelineRollbackHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/audits", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getPipelineAuditHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/stage", Scope(sdk.AuthConsumerScopeProject), r.POST(api.addStageHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/stage/condition", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getStageConditionsHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/stage/move", Scope(sdk.AuthConsumerScopeProject), r.POST(api.moveStageHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/stage/{stageID}", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getStageHandler), r.PUT(api.updateStageHandler), r.DELETE(api.deleteStageHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/stage/{stageID}/job", Scope(sdk.AuthConsumerScopeProject), r.POST(api.addJobToStageHandler))
	r.Handle("/project/{permProjectKey}/pipeline/{pipelineKey}/stage/{stageID}/job/{jobID}", Scope(sdk.AuthConsumerScopeProject), r.PUT(api.updateJobHandler), r.DELETE(api.deleteJobHandler))

	// Preview pipeline
	r.Handle("/project/{permProjectKey}/preview/pipeline", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postPipelinePreviewHandler))
	// Import pipeline
	r.Handle("/project/{permProjectKey}/import/pipeline", Scope(sdk.AuthConsumerScopeProject), r.POST(api.importPipelineHandler))
	// Import pipeline (ONLY USE FOR UI)
	r.Handle("/project/{permProjectKey}/import/pipeline/{pipelineKey}", Scope(sdk.AuthConsumerScopeProject), r.PUT(api.putImportPipelineHandler))
	// Export pipeline
	r.Handle("/project/{permProjectKey}/export/pipeline/{pipelineKey}", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getPipelineExportHandler))

	// Workflows
	r.Handle("/workflow/artifact/{hash}", ScopeNone(), r.GET(api.downloadworkflowArtifactDirectHandler, service.OverrideAuth(service.NoAuthMiddleware)))

	r.Handle("/project/{key}/type/{type}/access", ScopeNone(), r.GET(api.getProjectAccessHandler))
	r.Handle("/project/{permProjectKey}/workflows", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postWorkflowHandler), r.GET(api.getWorkflowsHandler))
	r.Handle("/project/{permProjectKey}/workflows/runs/nodes/ids", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getWorkflowsRunsAndNodesIDshandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getWorkflowHandler), r.PUT(api.putWorkflowHandler), r.DELETE(api.deleteWorkflowHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/retention/maxruns", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postWorkflowMaxRunHandler, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/retention/dryrun", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postWorkflowRetentionPolicyDryRun))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/retention/suggest", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getRetentionPolicySuggestionHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/eventsintegration/{integrationID}", Scope(sdk.AuthConsumerScopeProject), r.DELETE(api.deleteWorkflowEventsIntegrationHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/icon", Scope(sdk.AuthConsumerScopeProject), r.PUT(api.putWorkflowIconHandler), r.DELETE(api.deleteWorkflowIconHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/ascode", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postWorkflowAsCodeHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/ascode/events/resync", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postWorkflowAsCodeEventsResyncHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/label", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postWorkflowLabelHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/label/{labelID}", Scope(sdk.AuthConsumerScopeProject), r.DELETE(api.deleteWorkflowLabelHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/rollback/{auditID}", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postWorkflowRollbackHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/notifications/conditions", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getWorkflowNotificationsConditionsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/groups", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postWorkflowGroupHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/groups/{groupName}", Scope(sdk.AuthConsumerScopeProject), r.PUT(api.putWorkflowGroupHandler), r.DELETE(api.deleteWorkflowGroupHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/hooks/{uuid}", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getWorkflowHookHandler))
	r.Handle("/project/{key}/workflow/{permWorkflowName}/node/{nodeID}/hook/model", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getWorkflowHookModelsHandler))
	r.Handle("/project/{key}/workflow/{permWorkflowName}/node/{nodeID}/outgoinghook/model", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getWorkflowOutgoingHookModelsHandler))

	// Outgoing hook model
	r.Handle("/workflow/outgoinghook/model", ScopeNone(), r.GET(api.getWorkflowOutgoingHookModelsHandler))

	// Preview workflows
	r.Handle("/project/{permProjectKey}/preview/workflows", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postWorkflowPreviewHandler))
	// Import workflows
	r.Handle("/project/{permProjectKey}/import/workflows", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postWorkflowImportHandler))
	// Import workflows (ONLY USE FOR UI EDIT AS CODE)
	r.Handle("/project/{key}/import/workflows/{permWorkflowName}", Scope(sdk.AuthConsumerScopeProject), r.PUT(api.putWorkflowImportHandler))
	// Export workflows
	r.Handle("/project/{key}/export/workflows/{permWorkflowName}", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getWorkflowExportHandler))
	// Pull workflows
	r.Handle("/project/{key}/pull/workflows/{permWorkflowName}", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getWorkflowPullHandler))
	// Push workflows
	r.Handle("/project/{permProjectKey}/push/workflows", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postWorkflowPushHandler))

	// Workflows run
	r.Handle("/project/{permProjectKey}/runs", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getWorkflowAllRunsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/artifact/{artifactId}", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getDownloadArtifactHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getWorkflowRunsHandler), r.POSTEXECUTE(api.postWorkflowRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/branch/{branch}", Scope(sdk.AuthConsumerScopeRun), r.DELETE(api.deleteWorkflowRunsBranchHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/latest", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getLatestWorkflowRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/tags", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getWorkflowRunTagsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/num", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getWorkflowRunNumHandler), r.POST(api.postWorkflowRunNumHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getWorkflowRunHandler), r.DELETE(api.deleteWorkflowRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/stop", Scope(sdk.AuthConsumerScopeRun), r.POSTEXECUTE(api.stopWorkflowRunHandler, MaintenanceAware()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/vcs/resync", Scope(sdk.AuthConsumerScopeRun), r.POSTEXECUTE(api.postResyncVCSWorkflowRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/artifacts", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getWorkflowRunArtifactsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/artifacts/check", Scope(sdk.AuthConsumerScopeRunExecution), r.POST(api.workflowRunArtifactCheckUploadHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/artifacts/links", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getWorkflowRunArtifactLinksHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/results", Scope(sdk.AuthConsumerScopeRunExecution), r.GET(api.getWorkflowRunResultsHandler), r.POST(api.postWorkflowRunResultsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getWorkflowNodeRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/results", Scope(sdk.AuthConsumerScopeRunExecution), r.GET(api.getWorkflowNodeRunResultsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/stop", Scope(sdk.AuthConsumerScopeRun), r.POSTEXECUTE(api.stopWorkflowNodeRunHandler, MaintenanceAware()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeID}/history", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getWorkflowNodeRunHistoryHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/{nodeName}/commits", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getWorkflowCommitsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/job/{runJobID}/info", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getWorkflowNodeRunJobSpawnInfosHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/nodes/{nodeRunID}/job/{runJobID}/service/{serviceName}/link", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getWorkflowNodeRunJobServiceLinkHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/nodes/{nodeRunID}/job/{runJobID}/service/{serviceName}/log", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getWorkflowNodeRunJobServiceLogHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/nodes/{nodeRunID}/job/{runJobID}/links", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getWorkflowNodeRunJobStepLinksHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/nodes/{nodeRunID}/job/{runJobID}/step/{stepOrder}/link", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getWorkflowNodeRunJobStepLinkHandler))
	r.Handle("/project/{key}/workflows/{workflowName}/type/{type}/access", ScopeNone(), r.GET(api.getWorkflowAccessHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/nodes/{nodeRunID}/job/{runJobID}/step/{stepOrder}/log", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getWorkflowNodeRunJobStepLogHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/node/{nodeID}/triggers/condition", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getWorkflowTriggerConditionHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/hook/triggers/condition", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getWorkflowTriggerHookConditionHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/triggers/condition", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getWorkflowTriggerConditionHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/release", Scope(sdk.AuthConsumerScopeRun), r.POST(api.releaseApplicationWorkflowHandler, MaintenanceAware()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/hooks/{hookRunID}/callback", Scope(sdk.AuthConsumerScopeRun), r.POST(api.postWorkflowJobHookCallbackHandler, MaintenanceAware()))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/hooks/{hookRunID}/details", Scope(sdk.AuthConsumerScopeRun), r.GET(api.getWorkflowJobHookDetailsHandler))

	// Environment
	r.Handle("/project/{permProjectKey}/environment", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getEnvironmentsHandler), r.POST(api.addEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/import", Scope(sdk.AuthConsumerScopeProject), r.POST(api.importNewEnvironmentHandler, DEPRECATED))
	r.Handle("/project/{permProjectKey}/environment/import/{environmentName}", Scope(sdk.AuthConsumerScopeProject), r.POST(api.importIntoEnvironmentHandler, DEPRECATED))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getEnvironmentHandler), r.PUT(api.updateEnvironmentHandler), r.DELETE(api.deleteEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/ascode", Scope(sdk.AuthConsumerScopeProject), r.PUT(api.updateAsCodeEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/usage", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getEnvironmentUsageHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/keys", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getKeysInEnvironmentHandler), r.POST(api.addKeyInEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/keys/{name}", Scope(sdk.AuthConsumerScopeProject), r.DELETE(api.deleteKeyInEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/clone/{cloneName}", Scope(sdk.AuthConsumerScopeProject), r.POST(api.cloneEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/variable", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getVariablesInEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/variable/{name}", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getVariableInEnvironmentHandler), r.POST(api.addVariableInEnvironmentHandler), r.PUT(api.updateVariableInEnvironmentHandler), r.DELETE(api.deleteVariableFromEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/{environmentName}/variable/{name}/audit", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getVariableAuditInEnvironmentHandler))

	// Import Environment
	r.Handle("/project/{permProjectKey}/import/environment", Scope(sdk.AuthConsumerScopeProject), r.POST(api.postEnvironmentImportHandler))
	// Export Environment
	r.Handle("/project/{permProjectKey}/export/environment/{environmentName}", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getEnvironmentExportHandler))

	// Project storage
	r.Handle("/project/{permProjectKey}/storage/{integrationName}", Scope(sdk.AuthConsumerScopeRunExecution), r.GET(api.getArtifactsStoreHandler))
	r.Handle("/project/{permProjectKey}/storage/{integrationName}/artifact/{ref}", Scope(sdk.AuthConsumerScopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobArtifactHandler, MaintenanceAware()))
	r.Handle("/project/{permProjectKey}/storage/{integrationName}/artifact/{ref}/url", Scope(sdk.AuthConsumerScopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobArtifacWithTempURLHandler, MaintenanceAware()))
	r.Handle("/project/{permProjectKey}/storage/{integrationName}/artifact/{ref}/url/callback", Scope(sdk.AuthConsumerScopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobArtifactWithTempURLCallbackHandler, MaintenanceAware()))
	r.Handle("/project/{permProjectKey}/storage/{integrationName}/staticfiles/{name}", Scope(sdk.AuthConsumerScopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobStaticFilesHandler, MaintenanceAware()))

	// Cache
	r.Handle("/project/{permProjectKey}/storage/{integrationName}/cache/{tag}", Scope(sdk.AuthConsumerScopeRunExecution), r.POSTEXECUTE(api.postPushCacheHandler, MaintenanceAware()), r.GET(api.getPullCacheHandler))
	r.Handle("/project/{permProjectKey}/storage/{integrationName}/cache/{tag}/url", Scope(sdk.AuthConsumerScopeRunExecution), r.POSTEXECUTE(api.postPushCacheWithTempURLHandler, MaintenanceAware()), r.GET(api.getPullCacheWithTempURLHandler))

	//Workflow queue
	r.Handle("/queue/workflows", Scope(sdk.AuthConsumerScopeRun, sdk.AuthConsumerScopeRunExecution), r.GET(api.getWorkflowJobQueueHandler, MaintenanceAware()))
	r.Handle("/queue/workflows/count", Scope(sdk.AuthConsumerScopeRun), r.GET(api.countWorkflowJobQueueHandler, MaintenanceAware()))
	r.Handle("/queue/workflows/{id}/take", Scope(sdk.AuthConsumerScopeRunExecution), r.POST(api.postTakeWorkflowJobHandler, MaintenanceAware()))
	r.Handle("/queue/workflows/{permJobID}/cache/{tag}/links", Scope(sdk.AuthConsumerScopeRunExecution), r.GET(api.getWorkerCacheLinkHandler, MaintenanceAware()))
	r.Handle("/queue/workflows/{permJobID}/book", Scope(sdk.AuthConsumerScopeRunExecution), r.POST(api.postBookWorkflowJobHandler, MaintenanceAware()), r.DELETE(api.deleteBookWorkflowJobHandler, MaintenanceAware()))
	r.Handle("/queue/workflows/{permJobID}/infos", Scope(sdk.AuthConsumerScopeRunExecution), r.GET(api.getWorkflowJobHandler, MaintenanceAware()))
	r.Handle("/queue/workflows/{permJobID}/vulnerability", Scope(sdk.AuthConsumerScopeRunExecution), r.POSTEXECUTE(api.postVulnerabilityReportHandler, MaintenanceAware()))
	r.Handle("/queue/workflows/{permJobID}/spawn/infos", Scope(sdk.AuthConsumerScopeRunExecution), r.POST(api.postSpawnInfosWorkflowJobHandler, MaintenanceAware()))
	r.Handle("/queue/workflows/{permJobID}/result", Scope(sdk.AuthConsumerScopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobResultHandler, MaintenanceAware()))
	r.Handle("/queue/workflows/{jobID}/log", Scope(sdk.AuthConsumerScopeRunExecution, sdk.AuthConsumerScopeService), r.POSTEXECUTE(api.postWorkflowJobLogsHandler, MaintenanceAware()))
	r.Handle("/queue/workflows/log/service", Scope(sdk.AuthConsumerScopeRunExecution, sdk.AuthConsumerScopeService), r.POSTEXECUTE(api.postWorkflowJobServiceLogsHandler, MaintenanceAware()))
	r.Handle("/queue/workflows/{permJobID}/coverage", Scope(sdk.AuthConsumerScopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobCoverageResultsHandler, MaintenanceAware()))
	r.Handle("/queue/workflows/{permJobID}/test", Scope(sdk.AuthConsumerScopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobTestsResultsHandler, MaintenanceAware()))
	r.Handle("/queue/workflows/{permJobID}/tag", Scope(sdk.AuthConsumerScopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobTagsHandler, MaintenanceAware()))
	r.Handle("/queue/workflows/{permJobID}/step", Scope(sdk.AuthConsumerScopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobStepStatusHandler, MaintenanceAware()))
	r.Handle("/queue/workflows/{permJobID}/version", Scope(sdk.AuthConsumerScopeRunExecution), r.POSTEXECUTE(api.postWorkflowJobSetVersionHandler, MaintenanceAware()))

	r.Handle("/variable/type", ScopeNone(), r.GET(api.getVariableTypeHandler))
	r.Handle("/parameter/type", ScopeNone(), r.GET(api.getParameterTypeHandler))
	r.Handle("/notification/type", ScopeNone(), r.GET(api.getUserNotificationTypeHandler))
	r.Handle("/notification/state", ScopeNone(), r.GET(api.getUserNotificationStateValueHandler))

	// RepositoriesManager
	r.Handle("/repositories_manager", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getRepositoriesManagerHandler))
	r.Handle("/repositories_manager/oauth2/callback", Scope(sdk.AuthConsumerScopeProject), r.GET(api.repositoriesManagerOAuthCallbackHandler, service.OverrideAuth(service.NoAuthMiddleware)))

	// RepositoriesManager for projects
	r.Handle("/project/{permProjectKey}/repositories_manager", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getRepositoriesManagerForProjectHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/authorize", Scope(sdk.AuthConsumerScopeProject), r.POST(api.repositoriesManagerAuthorizeHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/authorize/callback", Scope(sdk.AuthConsumerScopeProject), r.POST(api.repositoriesManagerAuthorizeCallbackHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/authorize/basicauth", Scope(sdk.AuthConsumerScopeProject), r.POST(api.repositoriesManagerAuthorizeBasicHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}", Scope(sdk.AuthConsumerScopeProject), r.DELETE(api.deleteRepositoriesManagerHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/repo", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getRepoFromRepositoriesManagerHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/repos", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getReposFromRepositoriesManagerHandler))

	// RepositoriesManager for applications
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/applications", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getRepositoriesManagerLinkedApplicationsHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/application/{applicationName}/attach", Scope(sdk.AuthConsumerScopeProject), r.POST(api.attachRepositoriesManagerHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/application/{applicationName}/detach", Scope(sdk.AuthConsumerScopeProject), r.POST(api.detachRepositoriesManagerHandler))

	// Suggest
	r.Handle("/suggest/variable/{permProjectKey}", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getVariablesHandler))

	//Requirements
	r.Handle("/requirement/types", ScopeNone(), r.GET(api.getRequirementTypesHandler))
	r.Handle("/requirement/types/{type}", ScopeNone(), r.GET(api.getRequirementTypeValuesHandler))

	// config
	r.Handle("/config/user", ScopeNone(), r.GET(api.ConfigUserHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/config/vcs", ScopeNone(), r.GET(api.ConfigVCShandler))
	r.Handle("/config/cdn", ScopeNone(), r.GET(api.ConfigCDNHandler))
	r.Handle("/config/api", ScopeNone(), r.GET(api.ConfigAPIHandler))

	// Users
	r.Handle("/user", Scope(sdk.AuthConsumerScopeUser), r.GET(api.getUsersHandler))
	r.Handle("/user/favorite", Scope(sdk.AuthConsumerScopeUser), r.POST(api.postUserFavoriteHandler))
	r.Handle("/user/schema", Scope(sdk.AuthConsumerScopeUser), r.GET(api.getUserJSONSchema))
	r.Handle("/user/timeline", Scope(sdk.AuthConsumerScopeUser), r.GET(api.getTimelineHandler))
	r.Handle("/user/timeline/filter", Scope(sdk.AuthConsumerScopeUser), r.GET(api.getTimelineFilterHandler), r.POST(api.postTimelineFilterHandler))
	r.Handle("/user/{permUsernamePublic}", Scope(sdk.AuthConsumerScopeUser), r.GET(api.getUserHandler), r.PUT(api.putUserHandler), r.DELETE(api.deleteUserHandler))
	r.Handle("/user/{permUsernamePublic}/group", Scope(sdk.AuthConsumerScopeUser), r.GET(api.getUserGroupsHandler))
	r.Handle("/user/{permUsername}/contact", Scope(sdk.AuthConsumerScopeUser), r.GET(api.getUserContactsHandler))
	r.Handle("/user/{permUsername}/auth/consumer", Scope(sdk.AuthConsumerScopeAccessToken), r.GET(api.getConsumersByUserHandler), r.POST(api.postConsumerByUserHandler))
	r.Handle("/user/{permUsername}/auth/consumer/{permConsumerID}", Scope(sdk.AuthConsumerScopeAccessToken), r.DELETE(api.deleteConsumerByUserHandler))
	r.Handle("/user/{permUsername}/auth/consumer/{permConsumerID}/regen", Scope(sdk.AuthConsumerScopeAccessToken), r.POST(api.postConsumerRegenByUserHandler))
	r.Handle("/user/{permUsername}/auth/session", Scope(sdk.AuthConsumerScopeAccessToken), r.GET(api.getSessionsByUserHandler))
	r.Handle("/user/{permUsername}/auth/session/{permSessionID}", Scope(sdk.AuthConsumerScopeAccessToken), r.DELETE(api.deleteSessionByUserHandler))

	// Workers
	r.Handle("/worker", Scope(sdk.AuthConsumerScopeAdmin, sdk.AuthConsumerScopeWorker, sdk.AuthConsumerScopeHatchery), r.GET(api.getWorkersHandler))
	r.Handle("/worker/refresh", Scope(sdk.AuthConsumerScopeWorker), r.POST(api.postRefreshWorkerHandler, MaintenanceAware()))
	r.Handle("/worker/waiting", Scope(sdk.AuthConsumerScopeWorker), r.POST(api.workerWaitingHandler, MaintenanceAware()))

	// Worker models
	r.Handle("/worker/model", Scope(sdk.AuthConsumerScopeWorkerModel), r.POST(api.postWorkerModelHandler), r.GET(api.getWorkerModelsHandler))
	r.Handle("/worker/model/enabled", Scope(sdk.AuthConsumerScopeWorkerModel), r.GET(api.getWorkerModelsEnabledHandler))
	r.Handle("/worker/model/type", Scope(sdk.AuthConsumerScopeWorkerModel), r.GET(api.getWorkerModelTypesHandler))
	r.Handle("/worker/model/capability/type", Scope(sdk.AuthConsumerScopeWorkerModel), r.GET(api.getRequirementTypesHandler))
	r.Handle("/worker/model/pattern", Scope(sdk.AuthConsumerScopeWorkerModel), r.POST(api.postAddWorkerModelPatternHandler, service.OverrideAuth(api.authAdminMiddleware)), r.GET(api.getWorkerModelPatternsHandler))
	r.Handle("/worker/model/pattern/{type}/{name}", Scope(sdk.AuthConsumerScopeWorkerModel), r.GET(api.getWorkerModelPatternHandler), r.PUT(api.putWorkerModelPatternHandler, service.OverrideAuth(api.authAdminMiddleware)), r.DELETE(api.deleteWorkerModelPatternHandler, service.OverrideAuth(api.authAdminMiddleware)))
	r.Handle("/worker/model/import", Scope(sdk.AuthConsumerScopeWorkerModel), r.POST(api.postWorkerModelImportHandler))
	r.Handle("/worker/model/{permGroupName}/{permModelName}", Scope(sdk.AuthConsumerScopeWorkerModel), r.GET(api.getWorkerModelHandler), r.PUT(api.putWorkerModelHandler), r.DELETE(api.deleteWorkerModelHandler))
	r.Handle("/worker/model/{permGroupName}/{permModelName}/secret", Scope(sdk.AuthConsumerScopeWorkerModel), r.GET(api.getWorkerModelSecretHandler))
	r.Handle("/worker/model/{permGroupName}/{permModelName}/export", Scope(sdk.AuthConsumerScopeWorkerModel), r.GET(api.getWorkerModelExportHandler))
	r.Handle("/worker/model/{permGroupName}/{permModelName}/usage", Scope(sdk.AuthConsumerScopeWorkerModel), r.GET(api.getWorkerModelUsageHandler))
	r.Handle("/worker/model/{permGroupName}/{permModelName}/book", Scope(sdk.AuthConsumerScopeWorkerModel), r.PUT(api.putBookWorkerModelHandler, MaintenanceAware()))
	r.Handle("/worker/model/{permGroupName}/{permModelName}/error", Scope(sdk.AuthConsumerScopeWorkerModel), r.PUT(api.putSpawnErrorWorkerModelHandler, MaintenanceAware()))

	r.Handle("/worker/{id}/disable", Scope(sdk.AuthConsumerScopeAdmin, sdk.AuthConsumerScopeHatchery), r.POST(api.disableWorkerHandler, MaintenanceAware()))
	r.Handle("/worker/{name}", Scope(sdk.AuthConsumerScopeWorker), r.GET(api.getWorkerHandler))

	r.Handle("/project/{permProjectKey}/worker/model", Scope(sdk.AuthConsumerScopeWorkerModel), r.GET(api.getWorkerModelsForProjectHandler))
	r.Handle("/group/{permGroupName}/worker/model", Scope(sdk.AuthConsumerScopeWorkerModel), r.GET(api.getWorkerModelsForGroupHandler))

	// Workflows

	r.Handle("/workflow/search", Scope(sdk.AuthConsumerScopeProject), r.GET(api.getSearchWorkflowHandler))
	r.Handle("/workflow/hook", Scope(sdk.AuthConsumerScopeHooks), r.GET(api.getWorkflowHooksHandler))
	r.Handle("/workflow/hook/model/{model}", ScopeNone(), r.GET(api.getWorkflowHookModelHandler), r.POST(api.postWorkflowHookModelHandler, service.OverrideAuth(api.authAdminMiddleware)), r.PUT(api.putWorkflowHookModelHandler, service.OverrideAuth(api.authAdminMiddleware)))

	// SSE
	r.Handle("/ws", ScopeNone(), r.GET(api.getWebsocketHandler))

	// Engine µServices
	r.Handle("/services/register", Scope(sdk.AuthConsumerScopeService), r.POST(api.postServiceRegisterHandler, MaintenanceAware()))
	r.Handle("/services/heartbeat", Scope(sdk.AuthConsumerScopeService), r.POST(api.postServiceHearbeatHandler))
	r.Handle("/services/{type}", Scope(sdk.AuthConsumerScopeService), r.GET(api.getServiceHandler))

	// Templates
	r.Handle("/template", Scope(sdk.AuthConsumerScopeTemplate), r.GET(api.getTemplatesHandler), r.POST(api.postTemplateHandler))
	r.Handle("/template/push", Scope(sdk.AuthConsumerScopeTemplate), r.POST(api.postTemplatePushHandler))
	r.Handle("/template/{permGroupName}/{permTemplateSlug}", Scope(sdk.AuthConsumerScopeTemplate), r.GET(api.getTemplateHandler), r.PUT(api.putTemplateHandler), r.DELETE(api.deleteTemplateHandler))
	r.Handle("/template/{permGroupName}/{permTemplateSlug}/pull", Scope(sdk.AuthConsumerScopeTemplate), r.POST(api.postTemplatePullHandler))
	r.Handle("/template/{permGroupName}/{permTemplateSlug}/audit", Scope(sdk.AuthConsumerScopeTemplate), r.GET(api.getTemplateAuditsHandler))
	r.Handle("/template/{groupName}/{templateSlug}/apply", Scope(sdk.AuthConsumerScopeTemplate), r.POST(api.postTemplateApplyHandler))
	r.Handle("/template/{groupName}/{templateSlug}/bulk", Scope(sdk.AuthConsumerScopeTemplate), r.POST(api.postTemplateBulkHandler))
	r.Handle("/template/{groupName}/{templateSlug}/bulk/{bulkID}", Scope(sdk.AuthConsumerScopeTemplate), r.GET(api.getTemplateBulkHandler))
	r.Handle("/template/{groupName}/{templateSlug}/instance", Scope(sdk.AuthConsumerScopeTemplate), r.GET(api.getTemplateInstancesHandler))
	r.Handle("/template/{groupName}/{templateSlug}/instance/{instanceID}", Scope(sdk.AuthConsumerScopeTemplate), r.DELETE(api.deleteTemplateInstanceHandler))
	r.Handle("/template/{groupName}/{templateSlug}/usage", Scope(sdk.AuthConsumerScopeTemplate), r.GET(api.getTemplateUsageHandler))

	//Not Found handler
	r.Mux.NotFoundHandler = http.HandlerFunc(NotFoundHandler)

	r.computeScopeDetails()
}
