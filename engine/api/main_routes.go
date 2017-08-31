package api

import (
	"net/http"
	"path"
)

func (r *Router) Init() {
	r.Handle("/login", r.POST(LoginUser, Auth(false)))

	// Action
	r.Handle("/action", r.GET(getActionsHandler))
	r.Handle("/action/import", r.POST(importActionHandler, NeedAdmin(true)))
	r.Handle("/action/requirement", r.GET(getActionsRequirements, Auth(false)))
	r.Handle("/action/{permActionName}", r.GET(getActionHandler), r.POST(addActionHandler), r.PUT(updateActionHandler), r.DELETE(deleteActionHandler))
	r.Handle("/action/{actionName}/using", r.GET(getPipelinesUsingActionHandler, NeedAdmin(true)))
	r.Handle("/action/{actionID}/audit", r.GET(getActionAuditHandler, NeedAdmin(true)))

	// Admin
	r.Handle("/admin/warning", r.DELETE(adminTruncateWarningsHandler, NeedAdmin(true)))
	r.Handle("/admin/maintenance", r.POST(postAdminMaintenanceHandler, NeedAdmin(true)), r.GET(getAdminMaintenanceHandler, NeedAdmin(true)), r.DELETE(deleteAdminMaintenanceHandler, NeedAdmin(true)))

	// Action plugin
	r.Handle("/plugin", r.POST(addPluginHandler, NeedAdmin(true)), r.PUT(updatePluginHandler, NeedAdmin(true)))
	r.Handle("/plugin/{name}", r.DELETE(deletePluginHandler, NeedAdmin(true)))
	r.Handle("/plugin/download/{name}", r.GET(downloadPluginHandler))

	// Download file
	r.ServeAbsoluteFile("/download/cli/x86_64", path.Join(r.Cfg.Directories.Download, "cds"), "cds")
	r.ServeAbsoluteFile("/download/worker/x86_64", path.Join(r.Cfg.Directories.Download, "worker"), "worker")
	r.ServeAbsoluteFile("/download/worker/windows_x86_64", path.Join(r.Cfg.Directories.Download, "worker.exe"), "worker.exe")
	r.ServeAbsoluteFile("/download/hatchery/x86_64", path.Join(r.Cfg.Directories.Download, "hatchery", "x86_64"), "hatchery")

	// Group
	r.Handle("/group", r.GET(getGroups), r.POST(addGroupHandler))
	r.Handle("/group/public", r.GET(getPublicGroups))
	r.Handle("/group/{permGroupName}", r.GET(getGroupHandler), r.PUT(updateGroupHandler), r.DELETE(deleteGroupHandler))
	r.Handle("/group/{permGroupName}/user", r.POST(addUserInGroup))
	r.Handle("/group/{permGroupName}/user/{user}", r.DELETE(removeUserFromGroupHandler))
	r.Handle("/group/{permGroupName}/user/{user}/admin", r.POST(setUserGroupAdminHandler), r.DELETE(removeUserGroupAdminHandler))
	r.Handle("/group/{permGroupName}/token/{expiration}", r.POST(generateTokenHandler))

	// Hatchery
	r.Handle("/hatchery", r.POST(registerHatchery, Auth(false)))
	r.Handle("/hatchery/{id}", r.PUT(refreshHatcheryHandler))

	// Hooks
	r.Handle("/hook", r.POST(receiveHook, Auth(false) /* Public handler called by third parties */))

	// Overall health
	r.Handle("/mon/status", r.GET(statusHandler, Auth(false)))
	r.Handle("/mon/smtp/ping", r.GET(smtpPingHandler, Auth(true)))
	r.Handle("/mon/version", r.GET(getVersionHandler, Auth(false)))
	r.Handle("/mon/stats", r.GET(getStats, Auth(false)))
	r.Handle("/mon/models", r.GET(getWorkerModelsStatsHandler, Auth(false)))
	r.Handle("/mon/building", r.GET(getBuildingPipelines))
	r.Handle("/mon/building/{hash}", r.GET(getPipelineBuildingCommit))
	r.Handle("/mon/warning", r.GET(getUserWarnings))
	r.Handle("/mon/lastupdates", r.GET(getUserLastUpdates))

	// Project
	r.Handle("/project", r.GET(getProjectsHandler), r.POST(addProjectHandler))
	r.Handle("/project/{permProjectKey}", r.GET(getProjectHandler), r.PUT(updateProjectHandler), r.DELETE(deleteProjectHandler))
	r.Handle("/project/{permProjectKey}/group", r.POST(addGroupInProject), r.PUT(updateGroupsInProject, DEPRECATED))
	r.Handle("/project/{permProjectKey}/group/{group}", r.PUT(updateGroupRoleOnProjectHandler), r.DELETE(deleteGroupFromProjectHandler))
	r.Handle("/project/{permProjectKey}/variable", r.GET(getVariablesInProjectHandler), r.PUT(updateVariablesInProjectHandler, DEPRECATED))
	r.Handle("/project/{key}/variable/audit", r.GET(getVariablesAuditInProjectnHandler))
	r.Handle("/project/{key}/variable/audit/{auditID}", r.PUT(restoreProjectVariableAuditHandler, DEPRECATED))
	r.Handle("/project/{permProjectKey}/variable/{name}", r.GET(getVariableInProjectHandler, DEPRECATED), r.POST(addVariableInProjectHandler), r.PUT(updateVariableInProjectHandler), r.DELETE(deleteVariableFromProjectHandler))
	r.Handle("/project/{permProjectKey}/variable/{name}/audit", r.GET(getVariableAuditInProjectHandler))
	r.Handle("/project/{permProjectKey}/applications", r.GET(getApplicationsHandler), r.POST(addApplicationHandler))
	r.Handle("/project/{permProjectKey}/notifications", r.GET(getProjectNotificationsHandler))
	r.Handle("/project/{permProjectKey}/keys", r.GET(getKeysInProjectHandler), r.POST(addKeyInProjectHandler))
	r.Handle("/project/{permProjectKey}/keys/{name}", r.DELETE(deleteKeyInProjectHandler))

	// Application
	r.Handle("/project/{key}/application/{permApplicationName}", r.GET(getApplicationHandler), r.PUT(updateApplicationHandler), r.DELETE(deleteApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/keys", r.GET(getKeysInApplicationHandler), r.POST(addKeyInApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/keys/{name}", r.DELETE(deleteKeyInApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/branches", r.GET(getApplicationBranchHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/version", r.GET(getApplicationBranchVersionHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/clone", r.POST(cloneApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/group", r.POST(addGroupInApplicationHandler), r.PUT(updateGroupsInApplicationHandler, DEPRECATED))
	r.Handle("/project/{key}/application/{permApplicationName}/group/{group}", r.PUT(updateGroupRoleOnApplicationHandler), r.DELETE(deleteGroupFromApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/history/branch", r.GET(getPipelineBuildBranchHistoryHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/history/env/deploy", r.GET(getApplicationDeployHistoryHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/notifications", r.POST(addNotificationsHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline", r.GET(getPipelinesInApplicationHandler), r.PUT(updatePipelinesToApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/attach", r.POST(attachPipelinesToApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}", r.POST(attachPipelineToApplicationHandler, DEPRECATED), r.PUT(updatePipelineToApplicationHandler, DEPRECATED), r.DELETE(removePipelineFromApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/notification", r.GET(getUserNotificationApplicationPipelineHandler), r.PUT(updateUserNotificationApplicationPipelineHandler), r.DELETE(deleteUserNotificationApplicationPipelineHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/scheduler", r.GET(getSchedulerApplicationPipelineHandler), r.POST(addSchedulerApplicationPipelineHandler), r.PUT(updateSchedulerApplicationPipelineHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/scheduler/{id}", r.DELETE(deleteSchedulerApplicationPipelineHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/tree", r.GET(getApplicationTreeHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/tree/status", r.GET(getApplicationTreeStatusHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/variable", r.GET(getVariablesInApplicationHandler), r.PUT(updateVariablesInApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/variable/audit", r.GET(getVariablesAuditInApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/variable/audit/{auditID}", r.PUT(restoreAuditHandler, DEPRECATED))
	r.Handle("/project/{key}/application/{permApplicationName}/variable/{name}", r.GET(getVariableInApplicationHandler), r.POST(addVariableInApplicationHandler), r.PUT(updateVariableInApplicationHandler), r.DELETE(deleteVariableFromApplicationHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/variable/{name}/audit", r.GET(getVariableAuditInApplicationHandler))

	// Pipeline
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/history", r.GET(getPipelineHistoryHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/log", r.GET(getBuildLogsHandler))
	r.Handle("/project/{key}/application/{app}/pipeline/{permPipelineKey}/build/{build}/test", r.POSTEXECUTE(addBuildTestResultsHandler), r.GET(getBuildTestResultsHandler))
	r.Handle("/project/{key}/application/{app}/pipeline/{permPipelineKey}/build/{build}/variable", r.POSTEXECUTE(addBuildVariableHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/action/{actionID}/step/{stepOrder}/log", r.GET(getStepBuildLogsHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/action/{actionID}/log", r.GET(getPipelineBuildJobLogsHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}", r.GET(getBuildStateHandler), r.DELETE(deleteBuildHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/triggered", r.GET(getPipelineBuildTriggeredHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/stop", r.POSTEXECUTE(stopPipelineBuildHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/restart", r.POSTEXECUTE(restartPipelineBuildHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/commits", r.GET(getPipelineBuildCommitsHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/commits", r.GET(getPipelineCommitsHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/run", r.POSTEXECUTE(runPipelineHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/runwithlastparent", r.POSTEXECUTE(runPipelineWithLastParentHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/rollback", r.POSTEXECUTE(rollbackPipelineHandler))

	r.Handle("/project/{permProjectKey}/pipeline", r.GET(getPipelinesHandler), r.POST(addPipeline))
	r.Handle("/project/{permProjectKey}/import/pipeline", r.POST(importPipelineHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/application", r.GET(getApplicationUsingPipelineHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/group", r.POST(addGroupInPipelineHandler), r.PUT(updateGroupsOnPipelineHandler, DEPRECATED))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/group/{group}", r.PUT(updateGroupRoleOnPipelineHandler), r.DELETE(deleteGroupFromPipelineHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/parameter", r.GET(getParametersInPipelineHandler), r.PUT(updateParametersInPipelineHandler, DEPRECATED))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/parameter/{name}", r.POST(addParameterInPipelineHandler), r.PUT(updateParameterInPipelineHandler), r.DELETE(deleteParameterFromPipelineHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}", r.GET(getPipelineHandler), r.PUT(updatePipelineHandler), r.DELETE(deletePipeline))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage", r.POST(addStageHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/move", r.POST(moveStageHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}", r.GET(getStageHandler), r.PUT(updateStageHandler), r.DELETE(deleteStageHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/job", r.POST(addJobToStageHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/job/{jobID}", r.PUT(updateJobHandler), r.DELETE(deleteJobHandler))

	// Workflows
	r.Handle("/project/{permProjectKey}/workflows", r.POST(postWorkflowHandler), r.GET(getWorkflowsHandler))
	r.Handle("/project/{permProjectKey}/workflows/{workflowName}", r.GET(getWorkflowHandler), r.PUT(putWorkflowHandler), r.DELETE(deleteWorkflowHandler))
	// Workflows run
	r.Handle("/project/{permProjectKey}/workflows/{workflowName}/runs", r.GET(getWorkflowRunsHandler), r.POST(postWorkflowRunHandler))
	r.Handle("/project/{permProjectKey}/workflows/{workflowName}/runs/latest", r.GET(getLatestWorkflowRunHandler))
	r.Handle("/project/{permProjectKey}/workflows/{workflowName}/runs/{number}", r.GET(getWorkflowRunHandler))
	r.Handle("/project/{permProjectKey}/workflows/{workflowName}/runs/{number}/artifacts", r.GET(getWorkflowRunArtifactsHandler))
	r.Handle("/project/{permProjectKey}/workflows/{workflowName}/runs/{number}/nodes/{id}", r.GET(getWorkflowNodeRunHandler))
	r.Handle("/project/{permProjectKey}/workflows/{workflowName}/runs/{number}/nodes/{id}/job/{runJobId}/step/{stepOrder}", r.GET(getWorkflowNodeRunJobStepHandler))
	r.Handle("/project/{permProjectKey}/workflows/{workflowName}/runs/{number}/nodes/{id}/artifacts", r.GET(getWorkflowNodeRunArtifactsHandler))
	r.Handle("/project/{permProjectKey}/workflows/{workflowName}/artifact/{artifactId}", r.GET(getDownloadArtifactHandler))
	r.Handle("/project/{permProjectKey}/workflows/{workflowName}/node/{nodeID}/triggers/condition", r.GET(getWorkflowTriggerConditionHandler))
	r.Handle("/project/{permProjectKey}/workflows/{workflowName}/join/{joinID}/triggers/condition", r.GET(getWorkflowTriggerJoinConditionHandler))

	// DEPRECATED
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/action/{jobID}", r.PUT(updatePipelineActionHandler, DEPRECATED), r.DELETE(deleteJobHandler))

	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/joined", r.POST(addJobToPipelineHandler))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/joined/{actionID}", r.GET(getJoinedAction), r.PUT(updateJoinedAction), r.DELETE(deleteJoinedAction))
	r.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/joined/{actionID}/audit", r.GET(getJoinedActionAuditHandler))

	// Triggers
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/trigger", r.GET(getTriggersHandler), r.POST(addTriggerHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/trigger/source", r.GET(getTriggersAsSourceHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/trigger/{id}", r.GET(getTriggerHandler), r.DELETE(deleteTriggerHandler), r.PUT(updateTriggerHandler))

	// Environment
	r.Handle("/project/{permProjectKey}/environment", r.GET(getEnvironmentsHandler), r.POST(addEnvironmentHandler), r.PUT(updateEnvironmentsHandler, DEPRECATED))
	r.Handle("/project/{permProjectKey}/environment/import", r.POST(importNewEnvironmentHandler))
	r.Handle("/project/{permProjectKey}/environment/import/{permEnvironmentName}", r.POST(importIntoEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}", r.GET(getEnvironmentHandler), r.PUT(updateEnvironmentHandler), r.DELETE(deleteEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/keys", r.GET(getKeysInEnvironmentHandler), r.POST(addKeyInEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/keys/{name}", r.DELETE(deleteKeyInEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/clone/{cloneName}", r.POST(cloneEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/audit", r.GET(getEnvironmentsAuditHandler, DEPRECATED))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/audit/{auditID}", r.PUT(restoreEnvironmentAuditHandler, DEPRECATED))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/group", r.POST(addGroupInEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/groups", r.POST(addGroupsInEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/group/{group}", r.PUT(updateGroupRoleOnEnvironmentHandler), r.DELETE(deleteGroupFromEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/variable", r.GET(getVariablesInEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/variable/{name}", r.GET(getVariableInEnvironmentHandler), r.POST(addVariableInEnvironmentHandler), r.PUT(updateVariableInEnvironmentHandler), r.DELETE(deleteVariableFromEnvironmentHandler))
	r.Handle("/project/{key}/environment/{permEnvironmentName}/variable/{name}/audit", r.GET(getVariableAuditInEnvironmentHandler))

	// Artifacts
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/artifact/{tag}", r.GET(listArtifactsHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/{buildNumber}/artifact", r.GET(listArtifactsBuildHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/{buildNumber}/artifact/{tag}", r.POSTEXECUTE(uploadArtifactHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/artifact/download/{id}", r.GET(downloadArtifactHandler))
	r.Handle("/artifact/{hash}", r.GET(downloadArtifactDirectHandler, Auth(false)))

	// Hooks
	r.Handle("/project/{key}/application/{permApplicationName}/hook", r.GET(getApplicationHooksHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/hook", r.POST(addHook), r.GET(getHooks))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/hook/{id}", r.PUT(updateHookHandler), r.DELETE(deleteHook))

	// Pollers
	r.Handle("/project/{key}/application/{permApplicationName}/polling", r.GET(getApplicationPollersHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/polling", r.POST(addPollerHandler), r.GET(getPollersHandler), r.PUT(updatePollerHandler), r.DELETE(deletePollerHandler))

	// Build queue
	r.Handle("/queue", r.GET(getQueueHandler))
	r.Handle("/queue/{id}/take", r.POST(takePipelineBuildJobHandler))
	r.Handle("/queue/{id}/book", r.POST(bookPipelineBuildJobHandler, NeedHatchery()))
	r.Handle("/queue/{id}/spawn/infos", r.POST(addSpawnInfosPipelineBuildJobHandler, NeedWorker(), NeedHatchery()))
	r.Handle("/queue/{id}/result", r.POST(addQueueResultHandler))
	r.Handle("/queue/{id}/infos", r.GET(getPipelineBuildJobHandler))
	r.Handle("/build/{id}/log", r.POST(addBuildLogHandler))
	r.Handle("/build/{id}/step", r.POST(updateStepStatusHandler))

	//Workflow queue
	r.Handle("/queue/workflows", r.GET(getWorkflowJobQueueHandler))
	r.Handle("/queue/workflows/requirements/errors", r.POST(postWorkflowJobRequirementsErrorHandler, NeedWorker()))
	r.Handle("/queue/workflows/{id}/take", r.POST(postTakeWorkflowJobHandler, NeedWorker()))
	r.Handle("/queue/workflows/{id}/book", r.POST(postBookWorkflowJobHandler, NeedHatchery()))
	r.Handle("/queue/workflows/{id}/infos", r.GET(getWorkflowJobHandler, NeedWorker()))
	r.Handle("/queue/workflows/{id}/spawn/infos", r.POST(postSpawnInfosWorkflowJobHandler, NeedHatchery()))
	r.Handle("/queue/workflows/{permID}/result", r.POSTEXECUTE(postWorkflowJobResultHandler, NeedWorker()))
	r.Handle("/queue/workflows/{permID}/log", r.POSTEXECUTE(postWorkflowJobLogsHandler, NeedWorker()))
	r.Handle("/queue/workflows/{permID}/test", r.POSTEXECUTE(postWorkflowJobTestsResultsHandler, NeedWorker()))
	r.Handle("/queue/workflows/{permID}/variable", r.POSTEXECUTE(postWorkflowJobVariableHandler, NeedWorker()))
	r.Handle("/queue/workflows/{permID}/step", r.POSTEXECUTE(postWorkflowJobStepStatusHandler, NeedWorker()))
	r.Handle("/queue/workflows/{permID}/artifact/{tag}", r.POSTEXECUTE(postWorkflowJobArtifactHandler, NeedWorker()))

	r.Handle("/variable/type", r.GET(getVariableTypeHandler))
	r.Handle("/parameter/type", r.GET(getParameterTypeHandler))
	r.Handle("/pipeline/type", r.GET(getPipelineTypeHandler))
	r.Handle("/notification/type", r.GET(getUserNotificationTypeHandler))
	r.Handle("/notification/state", r.GET(getUserNotificationStateValueHandler))

	// RepositoriesManager
	r.Handle("/repositories_manager", r.GET(getRepositoriesManagerHandler))
	r.Handle("/repositories_manager/add", r.POST(addRepositoriesManagerHandler, NeedAdmin(true)))
	r.Handle("/repositories_manager/oauth2/callback", r.GET(repositoriesManagerOAuthCallbackHandler, Auth(false)))
	// RepositoriesManager for projects
	r.Handle("/project/{permProjectKey}/repositories_manager", r.GET(getRepositoriesManagerForProjectHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/authorize", r.POST(repositoriesManagerAuthorize))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/authorize/callback", r.POST(repositoriesManagerAuthorizeCallback))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}", r.DELETE(deleteRepositoriesManagerHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/repo", r.GET(getRepoFromRepositoriesManagerHandler))
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/repos", r.GET(getReposFromRepositoriesManagerHandler))

	// RepositoriesManager for applications
	r.Handle("/project/{permProjectKey}/repositories_manager/{name}/application", r.POST(addApplicationFromRepositoriesManagerHandler))
	r.Handle("/project/{key}/repositories_manager/{name}/application/{permApplicationName}/attach", r.POST(attachRepositoriesManager))
	r.Handle("/project/{key}/repositories_manager/{name}/application/{permApplicationName}/detach", r.POST(detachRepositoriesManager))
	r.Handle("/project/{key}/application/{permApplicationName}/repositories_manager/{name}/hook", r.POST(addHookOnRepositoriesManagerHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/repositories_manager/hook/{hookId}", r.DELETE(deleteHookOnRepositoriesManagerHandler))

	// Suggest
	r.Handle("/suggest/variable/{permProjectKey}", r.GET(getVariablesHandler))

	// Templates
	r.Handle("/template", r.GET(getTemplatesHandler, Auth(false)))
	r.Handle("/template/add", r.POST(addTemplateHandler, NeedAdmin(true)))
	r.Handle("/template/build", r.GET(getBuildTemplatesHandler, Auth(false)))
	r.Handle("/template/deploy", r.GET(getDeployTemplatesHandler, Auth(false)))
	r.Handle("/template/{id}", r.PUT(updateTemplateHandler, NeedAdmin(true)), r.DELETE(deleteTemplateHandler, NeedAdmin(true)))
	r.Handle("/project/{permProjectKey}/template", r.POST(applyTemplateHandler))
	r.Handle("/project/{key}/application/{permApplicationName}/template", r.POST(applyTemplateOnApplicationHandler))

	// UI
	r.Handle("/config/user", r.GET(ConfigUserHandler, Auth(true)))

	// Users
	r.Handle("/user", r.GET(GetUsers))
	r.Handle("/user/signup", r.POST(AddUser, Auth(false)))
	r.Handle("/user/import", r.POST(importUsersHandler, NeedAdmin(true)))
	r.Handle("/user/{username}", r.GET(GetUserHandler, NeedUsernameOrAdmin(true)), r.PUT(UpdateUserHandler, NeedUsernameOrAdmin(true)), r.DELETE(DeleteUserHandler, NeedUsernameOrAdmin(true)))
	r.Handle("/user/{username}/groups", r.GET(getUserGroupsHandler, NeedUsernameOrAdmin(true)))
	r.Handle("/user/{username}/confirm/{token}", r.GET(ConfirmUser, Auth(false)))
	r.Handle("/user/{username}/reset", r.POST(ResetUser, Auth(false)))
	r.Handle("/auth/mode", r.GET(AuthModeHandler, Auth(false)))

	// Workers
	r.Handle("/worker", r.GET(getWorkersHandler, Auth(false)), r.POST(registerWorkerHandler, Auth(false)))
	r.Handle("/worker/refresh", r.POST(refreshWorkerHandler))
	r.Handle("/worker/checking", r.POST(workerCheckingHandler))
	r.Handle("/worker/waiting", r.POST(workerWaitingHandler))
	r.Handle("/worker/unregister", r.POST(unregisterWorkerHandler))
	r.Handle("/worker/{id}/disable", r.POST(disableWorkerHandler))

	// Worker models
	r.Handle("/worker/model", r.POST(addWorkerModel), r.GET(getWorkerModels))
	r.Handle("/worker/model/error/{permModelID}", r.PUT(spawnErrorWorkerModelHandler, NeedHatchery()))
	r.Handle("/worker/model/enabled", r.GET(getWorkerModelsEnabled))
	r.Handle("/worker/model/type", r.GET(getWorkerModelTypes))
	r.Handle("/worker/model/communication", r.GET(getWorkerModelCommunications))
	r.Handle("/worker/model/{permModelID}", r.PUT(updateWorkerModel), r.DELETE(deleteWorkerModel))
	r.Handle("/worker/model/capability/type", r.GET(getWorkerModelCapaTypes))

	// Workflows
	r.Handle("/workflow/hook", r.GET(getWorkflowHookModelsHandler))
	r.Handle("/workflow/hook/{model}", r.GET(getWorkflowHookModelHandler), r.POST(postWorkflowHookModelHandler, NeedAdmin(true)), r.PUT(putWorkflowHookModelHandler, NeedAdmin(true)))

	// SSE
	r.Handle("/mon/lastupdates/events", r.GET(lastUpdateBroker.ServeHTTP))

	//Not Found handler
	r.Mux.NotFoundHandler = http.HandlerFunc(notFoundHandler)
}
