package api

import (
	"net/http"
	"path"
	"sync"
)

// InitRouter initializes the router and all the routes
func (api *API) InitRouter() {
	api.Router.URL = api.Config.URL.API
	api.Router.SetHeaderFunc = DefaultHeaders
	api.Router.Middlewares = append(api.Router.Middlewares, api.authMiddleware)
	api.Router.PostMiddlewares = append(api.Router.PostMiddlewares, api.deletePermissionMiddleware)
	api.lastUpdateBroker = &lastUpdateBroker{
		make(map[string]*lastUpdateBrokerSubscribe),
		make(chan *lastUpdateBrokerSubscribe),
		make(chan string),
		&sync.Mutex{},
	}
	api.lastUpdateBroker.Init(api.Router.Background, api.DBConnectionFactory.GetDBMap, api.Cache)

	r := api.Router
	r.Handle("/login", r.POST(api.loginUserHandler, Auth(false)))

	// Action
	r.Handle("/action", r.GET(api.getActionsHandler))
	r.Handle("/action/import", r.POST(api.importActionHandler, NeedAdmin(true)))
	r.Handle("/action/requirement", r.GET(api.getActionsRequirements, Auth(false)))
	r.Handle("/action/{permActionName}", r.GET(api.getActionHandler), r.POST(api.addActionHandler), r.PUT(api.updateActionHandler), r.DELETE(api.deleteActionHandler))
	r.Handle("/action/{actionName}/using", r.GET(api.getPipelinesUsingActionHandler, NeedAdmin(true)))
	r.Handle("/action/{actionID}/audit", r.GET(api.getActionAuditHandler, NeedAdmin(true)))

	// Admin
	r.Handle("/admin/warning", r.DELETE(api.adminTruncateWarningsHandler, NeedAdmin(true)))
	r.Handle("/admin/maintenance", r.POST(api.postAdminMaintenanceHandler, NeedAdmin(true)), r.GET(api.getAdminMaintenanceHandler, NeedAdmin(true)), r.DELETE(api.deleteAdminMaintenanceHandler, NeedAdmin(true)))
	r.Handle("/admin/debug", r.GET(api.getProfileIndexHandler, Auth(false)))
	r.Handle("/admin/debug/trace", r.POST(api.getTraceHandler, NeedAdmin(true)))
	r.Handle("/admin/debug/cpu", r.POST(api.getCPUProfileHandler, NeedAdmin(true)))
	r.Handle("/admin/debug/{name}", r.POST(api.getProfileHandler, NeedAdmin(true)))

	// Action plugin
	r.Handle("/plugin", r.POST(api.addPluginHandler, NeedAdmin(true)), r.PUT(api.updatePluginHandler, NeedAdmin(true)))
	r.Handle("/plugin/{name}", r.DELETE(api.deletePluginHandler, NeedAdmin(true)))
	r.Handle("/plugin/download/{name}", r.GET(api.downloadPluginHandler))

	// Download file
	r.Handle("/download", r.GET(api.downloadsHandler))
	r.Handle("/download/{name}/{os}/{arch}", r.GET(api.downloadHandler, Auth(false)))

	r.ServeAbsoluteFile("/download/cli/x86_64", path.Join(api.Config.Directories.Download, "cds-linux-amd64"), "cds")
	r.ServeAbsoluteFile("/download/worker/x86_64", path.Join(api.Config.Directories.Download, "cds-worker-linux-amd64"), "worker")
	r.ServeAbsoluteFile("/download/worker/windows_x86_64", path.Join(api.Config.Directories.Download, "cds-worker-windows-amd64"), "worker.exe")
	r.ServeAbsoluteFile("/download/worker/i386", path.Join(api.Config.Directories.Download, "cds-worker-linux-386"), "worker")
	r.ServeAbsoluteFile("/download/worker/i686", path.Join(api.Config.Directories.Download, "cds-worker-linux-386"), "worker")

	r.ServeAbsoluteFile("/download/cdsctl-windows-amd64", path.Join(api.Config.Directories.Download, "cdsctl-windows-amd64"), "cdsctl-windows-amd64")
	r.ServeAbsoluteFile("/download/cdsctl-linux-amd64", path.Join(api.Config.Directories.Download, "cdsctl-linux-amd64"), "cdsctl-linux-amd64")
	r.ServeAbsoluteFile("/download/cdsctl-freebsd-amd64", path.Join(api.Config.Directories.Download, "cdsctl-freebsd-amd64"), "cdsctl-freebsd-amd64")
	r.ServeAbsoluteFile("/download/cdsctl-darwin-amd64", path.Join(api.Config.Directories.Download, "cdsctl-darwin-amd64"), "cdsctl-darwin-amd64")

	r.ServeAbsoluteFile("/download/cds-engine-darwin-amd64", path.Join(api.Config.Directories.Download, "cds-engine-darwin-amd64"), "cds-engine-darwin-amd64")
	r.ServeAbsoluteFile("/download/cds-engine-windows-amd64", path.Join(api.Config.Directories.Download, "cds-engine-windows-amd64"), "cds-engine-windows-amd64")
	r.ServeAbsoluteFile("/download/cds-engine-linux-amd64", path.Join(api.Config.Directories.Download, "cds-engine-linux-amd64"), "cds-engine-linux-amd64")

	r.ServeAbsoluteFile("/download/cds-worker-windows-amd64", path.Join(api.Config.Directories.Download, "cds-worker-windows-amd64"), "cds-worker-windows-amd64")
	r.ServeAbsoluteFile("/download/cds-worker-linux-amd64", path.Join(api.Config.Directories.Download, "cds-worker-linux-amd64"), "cds-worker-linux-amd64")
	r.ServeAbsoluteFile("/download/cds-worker-linux-386", path.Join(api.Config.Directories.Download, "cds-worker-linux-386"), "cds-worker-linux-386")
	r.ServeAbsoluteFile("/download/cds-worker-freebsd-amd64", path.Join(api.Config.Directories.Download, "cds-worker-freebsd-amd64"), "cds-worker-freebsd-amd64")
	r.ServeAbsoluteFile("/download/cds-worker-darwin-amd64", path.Join(api.Config.Directories.Download, "cds-worker-darwin-amd64"), "cds-worker-darwin-amd64")

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
	r.Handle("/hatchery", r.POST(api.registerHatcheryHandler, Auth(false)))
	r.Handle("/hatchery/{id}", r.PUT(api.refreshHatcheryHandler))

	// Hooks
	r.Handle("/hook", r.POST(api.receiveHookHandler, Auth(false) /* Public handler called by third parties */))

	// Overall health
	r.Handle("/mon/status", r.GET(api.statusHandler, Auth(false)))
	r.Handle("/mon/smtp/ping", r.GET(api.smtpPingHandler, Auth(true)))
	r.Handle("/mon/version", r.GET(VersionHandler, Auth(false)))
	r.Handle("/mon/stats", r.GET(api.getStatsHandler, Auth(false)))
	r.Handle("/mon/db/migrate", r.GET(api.getMonDBStatusMigrateHandler, NeedAdmin(true)))
	r.Handle("/mon/db/times", r.GET(api.getMonDBTimesDBHandler, NeedAdmin(true)))
	r.Handle("/mon/building", r.GET(api.getBuildingPipelinesHandler))
	r.Handle("/mon/building/{hash}", r.GET(api.getPipelineBuildingCommitHandler))
	r.Handle("/mon/warning", r.GET(api.getUserWarningsHandler))
	r.Handle("/mon/metrics", r.GET(api.getMetricsHandler, Auth(false)))

	// Specific web ui routes
	r.Handle("/ui/navbar", r.GET(api.getUINavbarHandler))

	// Project
	r.Handle("/project", r.GET(api.getProjectsHandler), r.POST(api.addProjectHandler))
	r.Handle("/project/{permProjectKey}", r.GET(api.getProjectHandler), r.PUT(api.updateProjectHandler), r.DELETE(api.deleteProjectHandler))
	r.Handle("/project/{permProjectKey}/group", r.POST(api.addGroupInProjectHandler), r.PUT(api.updateGroupsInProjectHandler, DEPRECATED))
	r.Handle("/project/{permProjectKey}/group/import", r.POST(api.importGroupsInProjectHandler))
	r.Handle("/project/{permProjectKey}/group/{group}", r.PUT(api.updateGroupRoleOnProjectHandler), r.DELETE(api.deleteGroupFromProjectHandler))
	r.Handle("/project/{permProjectKey}/variable", r.GET(api.getVariablesInProjectHandler), r.PUT(api.updateVariablesInProjectHandler, DEPRECATED))
	r.Handle("/project/{permProjectKey}/encrypt", r.POST(api.postEncryptVariableHandler))
	r.Handle("/project/{key}/variable/audit", r.GET(api.getVariablesAuditInProjectnHandler))
	r.Handle("/project/{key}/variable/audit/{auditID}", r.PUT(api.restoreProjectVariableAuditHandler, DEPRECATED))
	r.Handle("/project/{permProjectKey}/variable/{name}", r.GET(api.getVariableInProjectHandler, DEPRECATED), r.POST(api.addVariableInProjectHandler), r.PUT(api.updateVariableInProjectHandler), r.DELETE(api.deleteVariableFromProjectHandler))
	r.Handle("/project/{permProjectKey}/variable/{name}/audit", r.GET(api.getVariableAuditInProjectHandler))
	r.Handle("/project/{permProjectKey}/applications", r.GET(api.getApplicationsHandler), r.POST(api.addApplicationHandler))
	r.Handle("/project/{permProjectKey}/notifications", r.GET(api.getProjectNotificationsHandler))
	r.Handle("/project/{permProjectKey}/keys", r.GET(api.getKeysInProjectHandler), r.POST(api.addKeyInProjectHandler))
	r.Handle("/project/{permProjectKey}/keys/{name}", r.DELETE(api.deleteKeyInProjectHandler))
	// Import Application
	r.Handle("/project/{permProjectKey}/import/application", r.POST(api.postApplicationImportHandler))
	// Export Application
	r.Handle("/project/{key}/export/application/{permApplicationName}", r.GET(api.getApplicationExportHandler))

	// Application
	r.Handle("/project/{key}/application/{permApplicationName}", r.GET(api.getApplicationHandler), r.PUT(api.updateApplicationHandler), r.DELETE(api.deleteApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/keys", r.GET(api.getKeysInApplicationHandler), r.POST(api.addKeyInApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/keys/{name}", r.DELETE(api.deleteKeyInApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/branches", r.GET(api.getApplicationBranchHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/remotes", r.GET(api.getApplicationRemoteHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/version", r.GET(api.getApplicationBranchVersionHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/clone", r.POST(api.cloneApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/group", r.POST(api.addGroupInApplicationHandler), r.PUT(api.updateGroupsInApplicationHandler, DEPRECATED))
	r.Handle("/project/{key}/application/{permApplicationName}/group/import", r.POST(api.importGroupsInApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/group/{group}", r.PUT(api.updateGroupRoleOnApplicationHandler), r.DELETE(api.deleteGroupFromApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/history/branch", r.GET(api.getPipelineBuildBranchHistoryHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/history/env/deploy", r.GET(api.getApplicationDeployHistoryHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/notifications", r.POST(api.addNotificationsHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline", r.GET(api.getPipelinesInApplicationHandler), r.PUT(api.updatePipelinesToApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/attach", r.POST(api.attachPipelinesToApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}", r.POST(api.attachPipelineToApplicationHandler, DEPRECATED), r.PUT(api.updatePipelineToApplicationHandler, DEPRECATED), r.DELETE(api.removePipelineFromApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/notification", r.GET(api.getUserNotificationApplicationPipelineHandler), r.PUT(api.updateUserNotificationApplicationPipelineHandler), r.DELETE(api.deleteUserNotificationApplicationPipelineHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/scheduler", r.GET(api.getSchedulerApplicationPipelineHandler), r.POST(api.addSchedulerApplicationPipelineHandler), r.PUT(api.updateSchedulerApplicationPipelineHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/scheduler/{id}", r.DELETE(api.deleteSchedulerApplicationPipelineHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/tree", r.GET(api.getApplicationTreeHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/tree/status", r.GET(api.getApplicationTreeStatusHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/variable", r.GET(api.getVariablesInApplicationHandler), r.PUT(api.updateVariablesInApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/variable/audit", r.GET(api.getVariablesAuditInApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/variable/audit/{auditID}", r.PUT(api.restoreAuditHandler, DEPRECATED))
	r.Handle("/project/{key}/application/{permApplicationName}/variable/{name}", r.GET(api.getVariableInApplicationHandler), r.POST(api.addVariableInApplicationHandler), r.PUT(api.updateVariableInApplicationHandler), r.DELETE(api.deleteVariableFromApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/variable/{name}/audit", r.GET(api.getVariableAuditInApplicationHandler))

	// Application workflow migration
	r.Handle("/project/{key}/application/{permApplicationName}/workflow/migrate", r.POST(api.migrationApplicationWorkflowHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/workflow/clean", r.POST(api.migrationApplicationWorkflowCleanHandler))

	// Pipeline Build
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/history", r.GET(api.getPipelineHistoryHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/log", r.GET(api.getBuildLogsHandler))
	r.Handle("/project/{key}/application/{app}/pipeline/{permPipelineKey}/build/{build}/test", r.POSTEXECUTE(api.addBuildTestResultsHandler), r.GET(api.getBuildTestResultsHandler))
	r.Handle("/project/{key}/application/{app}/pipeline/{permPipelineKey}/build/{build}/variable", r.POSTEXECUTE(api.addBuildVariableHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/action/{actionID}/step/{stepOrder}/log", r.GET(api.getStepBuildLogsHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/action/{actionID}/log", r.GET(api.getPipelineBuildJobLogsHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}", r.GET(api.getBuildStateHandler), r.DELETE(api.deleteBuildHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/triggered", r.GET(api.getPipelineBuildTriggeredHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/stop", r.POSTEXECUTE(api.stopPipelineBuildHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/restart", r.POSTEXECUTE(api.restartPipelineBuildHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/commits", r.GET(api.getPipelineBuildCommitsHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/commits", r.GET(api.getPipelineCommitsHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/run", r.POSTEXECUTE(api.runPipelineHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/runwithlastparent", r.POSTEXECUTE(api.runPipelineWithLastParentHandler))

	// Pipeline
	r.Handle("/project/{permProjectKey}/pipeline", r.GET(api.getPipelinesHandler), r.POST(api.addPipelineHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/application", r.GET(api.getApplicationUsingPipelineHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/group", r.POST(api.addGroupInPipelineHandler), r.PUT(api.updateGroupsOnPipelineHandler, DEPRECATED))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/group/import", r.POST(api.importGroupsInPipelineHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/group/{group}", r.PUT(api.updateGroupRoleOnPipelineHandler), r.DELETE(api.deleteGroupFromPipelineHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/parameter", r.GET(api.getParametersInPipelineHandler), r.PUT(api.updateParametersInPipelineHandler, DEPRECATED))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/parameter/{name}", r.POST(api.addParameterInPipelineHandler), r.PUT(api.updateParameterInPipelineHandler), r.DELETE(api.deleteParameterFromPipelineHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}", r.GET(api.getPipelineHandler), r.PUT(api.updatePipelineHandler), r.DELETE(api.deletePipelineHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/audits", r.GET(api.getPipelineAuditHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage", r.POST(api.addStageHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/move", r.POST(api.moveStageHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}", r.GET(api.getStageHandler), r.PUT(api.updateStageHandler), r.DELETE(api.deleteStageHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/job", r.POST(api.addJobToStageHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/job/{jobID}", r.PUT(api.updateJobHandler), r.DELETE(api.deleteJobHandler))

	// Import pipeline
	r.Handle("/project/{permProjectKey}/import/pipeline", r.POST(api.importPipelineHandler))
	// Export pipeline
	r.Handle("/project/{key}/export/pipeline/{permPipelineKey}", r.GET(api.getPipelineExportHandler))

	// Workflows
	r.Handle("/workflow/artifact/{hash}", r.GET(api.downloadworkflowArtifactDirectHandler, Auth(false)))

	r.Handle("/project/{permProjectKey}/workflows", r.POST(api.postWorkflowHandler), r.GET(api.getWorkflowsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}", r.GET(api.getWorkflowHandler), r.PUT(api.putWorkflowHandler), r.DELETE(api.deleteWorkflowHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/groups", r.POST(api.postWorkflowGroupHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/groups/{groupName}", r.PUT(api.putWorkflowGroupHandler), r.DELETE(api.deleteWorkflowGroupHandler))
	r.Handle("/project/{key}/workflow/{permWorkflowName}/node/{nodeID}/hook/model", r.GET(api.getWorkflowHookModelsHandler))

	// Import workflows
	r.Handle("/project/{permProjectKey}/import/workflows", r.POST(api.postWorkflowImportHandler))
	// Export workflows
	r.Handle("/project/{key}/export/workflows/{permWorkflowName}", r.GET(api.getWorkflowExportHandler))
	// Pull workflows
	r.Handle("/project/{key}/pull/workflows/{permWorkflowName}", r.GET(api.getWorkflowPullHandler))
	// Push workflows
	r.Handle("/project/{permProjectKey}/push/workflows", r.POST(api.postWorkflowPushHandler))

	// Workflows run
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs", r.GET(api.getWorkflowRunsHandler), r.POSTEXECUTE(api.postWorkflowRunHandler, AllowServices(true)))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/latest", r.GET(api.getLatestWorkflowRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/tags", r.GET(api.getWorkflowRunTagsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/num", r.GET(api.getWorkflowRunNumHandler), r.POST(api.postWorkflowRunNumHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}", r.GET(api.getWorkflowRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/stop", r.POST(api.stopWorkflowRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/resync", r.POST(api.resyncWorkflowRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/artifacts", r.GET(api.getWorkflowRunArtifactsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}", r.GET(api.getWorkflowNodeRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/stop", r.POST(api.stopWorkflowNodeRunHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeID}/history", r.GET(api.getWorkflowNodeRunHistoryHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/{nodeName}/commits", r.GET(api.getWorkflowCommitsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/job/{runJobId}/step/{stepOrder}", r.GET(api.getWorkflowNodeRunJobStepHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/artifacts", r.GET(api.getWorkflowNodeRunArtifactsHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/artifact/{artifactId}", r.GET(api.getDownloadArtifactHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/node/{nodeID}/triggers/condition", r.GET(api.getWorkflowTriggerConditionHandler))
	r.Handle("/project/{key}/workflows/{permWorkflowName}/runs/{number}/nodes/{nodeRunID}/release", r.POST(api.releaseApplicationWorkflowHandler))

	// DEPRECATED
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/action/{jobID}", r.PUT(api.updatePipelineActionHandler, DEPRECATED), r.DELETE(api.deleteJobHandler))

	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/joined", r.POST(api.addJobToPipelineHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/joined/{actionID}", r.GET(api.getJoinedActionHandler), r.PUT(api.updateJoinedActionHandler), r.DELETE(api.deleteJoinedActionHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/joined/{actionID}/audit", r.GET(api.getJoinedActionAuditHandler))

	// Triggers
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/trigger", r.GET(api.getTriggersHandler), r.POST(api.addTriggerHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/trigger/source", r.GET(api.getTriggersAsSourceHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/trigger/{id}", r.GET(api.getTriggerHandler), r.DELETE(api.deleteTriggerHandler), r.PUT(api.updateTriggerHandler))

	// Environment
	r.Handle("/project/{permProjectKey}/environment", r.GET(api.getEnvironmentsHandler), r.POST(api.addEnvironmentHandler), r.PUT(api.updateEnvironmentsHandler, DEPRECATED))
	r.Handle("/project/{permProjectKey}/environment/import", r.POST(api.importNewEnvironmentHandler))
	r.Handle("/project/{key}/environment/import/{permEnvironmentName}", r.POST(api.importIntoEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}", r.GET(api.getEnvironmentHandler), r.PUT(api.updateEnvironmentHandler), r.DELETE(api.deleteEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/usage", r.GET(api.getEnvironmentUsageHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/keys", r.GET(api.getKeysInEnvironmentHandler), r.POST(api.addKeyInEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/keys/{name}", r.DELETE(api.deleteKeyInEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/clone/{cloneName}", r.POST(api.cloneEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/audit", r.GET(api.getEnvironmentsAuditHandler, DEPRECATED))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/audit/{auditID}", r.PUT(api.restoreEnvironmentAuditHandler, DEPRECATED))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/group", r.POST(api.addGroupInEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/groups", r.POST(api.addGroupsInEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/group/import", r.POST(api.importGroupsInEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/group/{group}", r.PUT(api.updateGroupRoleOnEnvironmentHandler), r.DELETE(api.deleteGroupFromEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/variable", r.GET(api.getVariablesInEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/variable/{name}", r.GET(api.getVariableInEnvironmentHandler), r.POST(api.addVariableInEnvironmentHandler), r.PUT(api.updateVariableInEnvironmentHandler), r.DELETE(api.deleteVariableFromEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/variable/{name}/audit", r.GET(api.getVariableAuditInEnvironmentHandler))

	// Import Environment
	r.Handle("/project/{permProjectKey}/import/environment", r.POST(api.postEnvironmentImportHandler))
	// Export Environment
	r.Handle("/project/{key}/export/environment/{permEnvironmentName}", r.GET(api.getEnvironmentExportHandler))

	// Artifacts
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/artifact/{tag}", r.GET(api.listArtifactsHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/{buildNumber}/artifact", r.GET(api.listArtifactsBuildHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/{buildNumber}/artifact/{tag}/url", r.POSTEXECUTE(api.postArtifactWithTempURLHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/{buildNumber}/artifact/{tag}/url/callback", r.POSTEXECUTE(api.postArtifactWithTempURLCallbackHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/{buildNumber}/artifact/{tag}", r.POSTEXECUTE(api.uploadArtifactHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/artifact/download/{id}", r.GET(api.downloadArtifactHandler))
	r.Handle("/artifact/store", r.GET(api.getArtifactsStoreHandler, Auth(false)))
	r.Handle("/artifact/{hash}", r.GET(api.downloadArtifactDirectHandler, Auth(false)))

	// Hooks
	r.Handle("/project/{key}/application/{permApplicationName}/hook", r.GET(api.getApplicationHooksHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/hook", r.POST(api.addHookHandler), r.GET(api.getHooksHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/hook/{id}", r.PUT(api.updateHookHandler), r.DELETE(api.deleteHookHandler))

	// Pollers
	r.Handle("/project/{key}/application/{permApplicationName}/polling", r.GET(api.getApplicationPollersHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/polling", r.POST(api.addPollerHandler), r.GET(api.getPollersHandler), r.PUT(api.updatePollerHandler), r.DELETE(api.deletePollerHandler))

	// Build queue
	r.Handle("/queue", r.GET(api.getQueueHandler))
	r.Handle("/queue/{id}/take", r.POST(api.takePipelineBuildJobHandler))
	r.Handle("/queue/{id}/book", r.POST(api.bookPipelineBuildJobHandler, NeedHatchery()))
	r.Handle("/queue/{id}/spawn/infos", r.POST(api.addSpawnInfosPipelineBuildJobHandler, NeedWorker(), NeedHatchery()))
	r.Handle("/queue/{id}/result", r.POST(api.addQueueResultHandler))
	r.Handle("/queue/{id}/infos", r.GET(api.getPipelineBuildJobHandler))
	r.Handle("/build/{id}/log", r.POST(api.addBuildLogHandler))
	r.Handle("/build/{id}/step", r.POST(api.updateStepStatusHandler))

	//Workflow queue
	r.Handle("/queue/workflows", r.GET(api.getWorkflowJobQueueHandler))
	r.Handle("/queue/workflows/count", r.GET(api.countWorkflowJobQueueHandler))
	r.Handle("/queue/workflows/requirements/errors", r.POST(api.postWorkflowJobRequirementsErrorHandler, NeedWorker()))
	r.Handle("/queue/workflows/{id}/take", r.POST(api.postTakeWorkflowJobHandler, NeedWorker()))
	r.Handle("/queue/workflows/{id}/book", r.POST(api.postBookWorkflowJobHandler, NeedHatchery()))
	r.Handle("/queue/workflows/{id}/infos", r.GET(api.getWorkflowJobHandler, NeedWorker()))
	r.Handle("/queue/workflows/{id}/spawn/infos", r.POST(r.Asynchronous(api.postSpawnInfosWorkflowJobHandler, 3), NeedHatchery()))
	r.Handle("/queue/workflows/{permID}/result", r.POSTEXECUTE(api.postWorkflowJobResultHandler, NeedWorker()))
	r.Handle("/queue/workflows/{permID}/log", r.POSTEXECUTE(r.Asynchronous(api.postWorkflowJobLogsHandler, 5), NeedWorker()))
	r.Handle("/queue/workflows/{permID}/test", r.POSTEXECUTE(api.postWorkflowJobTestsResultsHandler, NeedWorker()))
	r.Handle("/queue/workflows/{permID}/tag", r.POSTEXECUTE(api.postWorkflowJobTagsHandler, NeedWorker()))
	r.Handle("/queue/workflows/{permID}/variable", r.POSTEXECUTE(api.postWorkflowJobVariableHandler, NeedWorker()))
	r.Handle("/queue/workflows/{permID}/step", r.POSTEXECUTE(api.postWorkflowJobStepStatusHandler, NeedWorker()))
	r.Handle("/queue/workflows/{permID}/artifact/{tag}", r.POSTEXECUTE(api.postWorkflowJobArtifactHandler, NeedWorker()))
	r.Handle("/queue/workflows/{permID}/artifact/{tag}/url", r.POSTEXECUTE(api.postWorkflowJobArtifacWithTempURLHandler, NeedWorker()))
	r.Handle("/queue/workflows/{permID}/artifact/{tag}/url/callback", r.POSTEXECUTE(api.postWorkflowJobArtifactWithTempURLCallbackHandler, NeedWorker()))

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
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/application", r.POST(api.addApplicationFromRepositoriesManagerHandler))
	r.Handle("/project/{key}/repositories_manager/{name}/application/{permApplicationName}/attach", r.POST(api.attachRepositoriesManagerHandler))
	r.Handle("/project/{key}/repositories_manager/{name}/application/{permApplicationName}/detach", r.POST(api.detachRepositoriesManagerHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/repositories_manager/{name}/hook", r.POST(api.addHookOnRepositoriesManagerHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/repositories_manager/hook/{hookId}", r.DELETE(api.deleteHookOnRepositoriesManagerHandler))

	// Suggest
	r.Handle("/suggest/variable/{permProjectKey}", r.GET(api.getVariablesHandler))

	//Requirements
	r.Handle("/requirement/types", r.GET(api.getRequirementTypesHandler))
	r.Handle("/requirement/types/{type}", r.GET(api.getRequirementTypeValuesHandler))

	//Requirements
	r.Handle("/requirement/types", r.GET(api.getRequirementTypesHandler))
	r.Handle("/requirement/types/{type}", r.GET(api.getRequirementTypeValuesHandler))

	// UI
	r.Handle("/config/user", r.GET(api.ConfigUserHandler, Auth(true)))

	// Users
	r.Handle("/user", r.GET(api.getUsersHandler))
	r.Handle("/user/tokens", r.GET(api.getUserTokenListHandler))
	r.Handle("/user/signup", r.POST(api.addUserHandler, Auth(false)))
	r.Handle("/user/import", r.POST(api.importUsersHandler, NeedAdmin(true)))
	r.Handle("/user/{username}", r.GET(api.getUserHandler, NeedUsernameOrAdmin(true)), r.PUT(api.updateUserHandler, NeedUsernameOrAdmin(true)), r.DELETE(api.deleteUserHandler, NeedUsernameOrAdmin(true)))
	r.Handle("/user/{username}/groups", r.GET(api.getUserGroupsHandler, NeedUsernameOrAdmin(true)))
	r.Handle("/user/{username}/confirm/{token}", r.GET(api.confirmUserHandler, Auth(false)))
	r.Handle("/user/{username}/reset", r.POST(api.resetUserHandler, Auth(false)))
	r.Handle("/auth/mode", r.GET(api.authModeHandler, Auth(false)))

	// Workers
	r.Handle("/worker", r.GET(api.getWorkersHandler, Auth(false)), r.POST(api.registerWorkerHandler, Auth(false)))
	r.Handle("/worker/refresh", r.POST(api.refreshWorkerHandler))
	r.Handle("/worker/checking", r.POST(api.workerCheckingHandler))
	r.Handle("/worker/waiting", r.POST(api.workerWaitingHandler))
	r.Handle("/worker/unregister", r.POST(api.unregisterWorkerHandler))
	r.Handle("/worker/{id}/disable", r.POST(api.disableWorkerHandler))

	// Worker models
	r.Handle("/worker/model", r.POST(api.addWorkerModelHandler), r.GET(api.getWorkerModelsHandler))
	r.Handle("/worker/model/book/{permModelID}", r.PUT(api.bookWorkerModelHandler, NeedHatchery()))
	r.Handle("/worker/model/error/{permModelID}", r.PUT(api.spawnErrorWorkerModelHandler, NeedHatchery()))
	r.Handle("/worker/model/enabled", r.GET(api.getWorkerModelsEnabledHandler))
	r.Handle("/worker/model/type", r.GET(api.getWorkerModelTypesHandler))
	r.Handle("/worker/model/communication", r.GET(api.getWorkerModelCommunicationsHandler))
	r.Handle("/worker/model/{permModelID}", r.PUT(api.updateWorkerModelHandler), r.DELETE(api.deleteWorkerModelHandler))
	r.Handle("/worker/model/capability/type", r.GET(api.getRequirementTypesHandler))

	// Workflows
	r.Handle("/workflow/hook", r.GET(api.getWorkflowHooksHandler, NeedService()))
	r.Handle("/workflow/hook/model/{model}", r.GET(api.getWorkflowHookModelHandler), r.POST(api.postWorkflowHookModelHandler, NeedAdmin(true)), r.PUT(api.putWorkflowHookModelHandler, NeedAdmin(true)))

	// SSE
	r.Handle("/mon/lastupdates/events", r.GET(api.lastUpdateBroker.ServeHTTP))

	// Engine µServices
	r.Handle("/services/register", r.POST(api.postServiceRegisterHandler, Auth(false)))

	//Not Found handler
	r.Mux.NotFoundHandler = http.HandlerFunc(notFoundHandler)
}
