package main

import (
	"net/http"
	"path"

	"github.com/spf13/viper"
)

func (router *Router) init() {
	router.Handle("/login", POST(LoginUser, Auth(false)))

	// Action
	router.Handle("/action", GET(getActionsHandler))
	router.Handle("/action/import", POST(importActionHandler, NeedAdmin(true)))
	router.Handle("/action/requirement", GET(getActionsRequirements, Auth(false)))
	router.Handle("/action/{permActionName}", GET(getActionHandler), POST(addActionHandler), PUT(updateActionHandler), DELETE(deleteActionHandler))
	router.Handle("/action/{actionName}/using", GET(getPipelinesUsingActionHandler, NeedAdmin(true)))
	router.Handle("/action/{actionID}/audit", GET(getActionAuditHandler, NeedAdmin(true)))

	// Admin
	router.Handle("/admin/warning", DELETE(adminTruncateWarningsHandler, NeedAdmin(true)))
	router.Handle("/admin/maintenance", POST(postAdminMaintenanceHandler, NeedAdmin(true)), GET(getAdminMaintenanceHandler, NeedAdmin(true)), DELETE(deleteAdminMaintenanceHandler, NeedAdmin(true)))

	// Action plugin
	router.Handle("/plugin", POST(addPluginHandler, NeedAdmin(true)), PUT(updatePluginHandler, NeedAdmin(true)))
	router.Handle("/plugin/{name}", DELETE(deletePluginHandler, NeedAdmin(true)))
	router.Handle("/plugin/download/{name}", GET(downloadPluginHandler))

	// Download file
	router.ServeAbsoluteFile("/download/cli/x86_64", path.Join(viper.GetString(viperDownloadDirectory), "cds"), "cds")
	router.ServeAbsoluteFile("/download/worker/x86_64", path.Join(viper.GetString(viperDownloadDirectory), "worker"), "worker")
	router.ServeAbsoluteFile("/download/worker/windows_x86_64", path.Join(viper.GetString(viperDownloadDirectory), "worker.exe"), "worker.exe")
	router.ServeAbsoluteFile("/download/hatchery/x86_64", path.Join(viper.GetString(viperDownloadDirectory), "hatchery", "x86_64"), "hatchery")

	// Group
	router.Handle("/group", GET(getGroups), POST(addGroupHandler))
	router.Handle("/group/public", GET(getPublicGroups))
	router.Handle("/group/{permGroupName}", GET(getGroupHandler), PUT(updateGroupHandler), DELETE(deleteGroupHandler))
	router.Handle("/group/{permGroupName}/user", POST(addUserInGroup))
	router.Handle("/group/{permGroupName}/user/{user}", DELETE(removeUserFromGroupHandler))
	router.Handle("/group/{permGroupName}/user/{user}/admin", POST(setUserGroupAdminHandler), DELETE(removeUserGroupAdminHandler))
	router.Handle("/group/{permGroupName}/token/{expiration}", POST(generateTokenHandler))

	// Hatchery
	router.Handle("/hatchery", POST(registerHatchery, Auth(false)))
	router.Handle("/hatchery/{id}", PUT(refreshHatcheryHandler))

	// Hooks
	router.Handle("/hook", POST(receiveHook, Auth(false) /* Public handler called by third parties */))

	// Overall health
	router.Handle("/mon/status", GET(statusHandler, Auth(false)))
	router.Handle("/mon/smtp/ping", GET(smtpPingHandler, Auth(true)))
	router.Handle("/mon/version", GET(getVersionHandler, Auth(false)))
	router.Handle("/mon/stats", GET(getStats, Auth(false)))
	router.Handle("/mon/models", GET(getWorkerModelsStatsHandler, Auth(false)))
	router.Handle("/mon/building", GET(getBuildingPipelines))
	router.Handle("/mon/building/{hash}", GET(getPipelineBuildingCommit))
	router.Handle("/mon/warning", GET(getUserWarnings))
	router.Handle("/mon/lastupdates", GET(getUserLastUpdates))

	// Project
	router.Handle("/project", GET(getProjectsHandler), POST(addProjectHandler))
	router.Handle("/project/{permProjectKey}", GET(getProjectHandler), PUT(updateProjectHandler), DELETE(deleteProjectHandler))
	router.Handle("/project/{permProjectKey}/group", POST(addGroupInProject), PUT(updateGroupsInProject, DEPRECATED))
	router.Handle("/project/{permProjectKey}/group/{group}", PUT(updateGroupRoleOnProjectHandler), DELETE(deleteGroupFromProjectHandler))
	router.Handle("/project/{permProjectKey}/variable", GET(getVariablesInProjectHandler), PUT(updateVariablesInProjectHandler, DEPRECATED))
	router.Handle("/project/{key}/variable/audit", GET(getVariablesAuditInProjectnHandler))
	router.Handle("/project/{key}/variable/audit/{auditID}", PUT(restoreProjectVariableAuditHandler, DEPRECATED))
	router.Handle("/project/{permProjectKey}/variable/{name}", GET(getVariableInProjectHandler, DEPRECATED), POST(addVariableInProjectHandler), PUT(updateVariableInProjectHandler), DELETE(deleteVariableFromProjectHandler))
	router.Handle("/project/{permProjectKey}/variable/{name}/audit", GET(getVariableAuditInProjectHandler))
	router.Handle("/project/{permProjectKey}/applications", GET(getApplicationsHandler), POST(addApplicationHandler))
	router.Handle("/project/{permProjectKey}/notifications", GET(getProjectNotificationsHandler))
	router.Handle("/project/{permProjectKey}/keys", GET(getKeysInProjectHandler), POST(addKeyInProjectHandler))
	router.Handle("/project/{permProjectKey}/keys/{name}", DELETE(deleteKeyInProjectHandler))

	// Application
	router.Handle("/project/{key}/application/{permApplicationName}", GET(getApplicationHandler), PUT(updateApplicationHandler), DELETE(deleteApplicationHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/keys", GET(getKeysInApplicationHandler), POST(addKeyInApplicationHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/keys/{name}", DELETE(deleteKeyInApplicationHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/branches", GET(getApplicationBranchHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/version", GET(getApplicationBranchVersionHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/clone", POST(cloneApplicationHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/group", POST(addGroupInApplicationHandler), PUT(updateGroupsInApplicationHandler, DEPRECATED))
	router.Handle("/project/{key}/application/{permApplicationName}/group/{group}", PUT(updateGroupRoleOnApplicationHandler), DELETE(deleteGroupFromApplicationHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/history/branch", GET(getPipelineBuildBranchHistoryHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/history/env/deploy", GET(getApplicationDeployHistoryHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/notifications", POST(addNotificationsHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline", GET(getPipelinesInApplicationHandler), PUT(updatePipelinesToApplicationHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/attach", POST(attachPipelinesToApplicationHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}", POST(attachPipelineToApplicationHandler, DEPRECATED), PUT(updatePipelineToApplicationHandler, DEPRECATED), DELETE(removePipelineFromApplicationHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/notification", GET(getUserNotificationApplicationPipelineHandler), PUT(updateUserNotificationApplicationPipelineHandler), DELETE(deleteUserNotificationApplicationPipelineHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/scheduler", GET(getSchedulerApplicationPipelineHandler), POST(addSchedulerApplicationPipelineHandler), PUT(updateSchedulerApplicationPipelineHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/scheduler/{id}", DELETE(deleteSchedulerApplicationPipelineHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/tree", GET(getApplicationTreeHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/tree/status", GET(getApplicationTreeStatusHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/variable", GET(getVariablesInApplicationHandler), PUT(updateVariablesInApplicationHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/variable/audit", GET(getVariablesAuditInApplicationHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/variable/audit/{auditID}", PUT(restoreAuditHandler, DEPRECATED))
	router.Handle("/project/{key}/application/{permApplicationName}/variable/{name}", GET(getVariableInApplicationHandler), POST(addVariableInApplicationHandler), PUT(updateVariableInApplicationHandler), DELETE(deleteVariableFromApplicationHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/variable/{name}/audit", GET(getVariableAuditInApplicationHandler))

	// Pipeline
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/history", GET(getPipelineHistoryHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/log", GET(getBuildLogsHandler))
	router.Handle("/project/{key}/application/{app}/pipeline/{permPipelineKey}/build/{build}/test", POSTEXECUTE(addBuildTestResultsHandler), GET(getBuildTestResultsHandler))
	router.Handle("/project/{key}/application/{app}/pipeline/{permPipelineKey}/build/{build}/variable", POSTEXECUTE(addBuildVariableHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/action/{actionID}/step/{stepOrder}/log", GET(getStepBuildLogsHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/action/{actionID}/log", GET(getPipelineBuildJobLogsHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}", GET(getBuildStateHandler), DELETE(deleteBuildHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/triggered", GET(getPipelineBuildTriggeredHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/stop", POSTEXECUTE(stopPipelineBuildHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/restart", POSTEXECUTE(restartPipelineBuildHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/build/{build}/commits", GET(getPipelineBuildCommitsHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/commits", GET(getPipelineCommitsHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/run", POSTEXECUTE(runPipelineHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/runwithlastparent", POSTEXECUTE(runPipelineWithLastParentHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/rollback", POSTEXECUTE(rollbackPipelineHandler))

	router.Handle("/project/{permProjectKey}/pipeline", GET(getPipelinesHandler), POST(addPipeline))
	router.Handle("/project/{permProjectKey}/import/pipeline", POST(importPipelineHandler))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/application", GET(getApplicationUsingPipelineHandler))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/group", POST(addGroupInPipelineHandler), PUT(updateGroupsOnPipelineHandler, DEPRECATED))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/group/{group}", PUT(updateGroupRoleOnPipelineHandler), DELETE(deleteGroupFromPipelineHandler))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/parameter", GET(getParametersInPipelineHandler), PUT(updateParametersInPipelineHandler, DEPRECATED))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/parameter/{name}", POST(addParameterInPipelineHandler), PUT(updateParameterInPipelineHandler), DELETE(deleteParameterFromPipelineHandler))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}", GET(getPipelineHandler), PUT(updatePipelineHandler), DELETE(deletePipeline))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/stage", POST(addStageHandler))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/move", POST(moveStageHandler))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}", GET(getStageHandler), PUT(updateStageHandler), DELETE(deleteStageHandler))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/job", POST(addJobToStageHandler))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/job/{jobID}", PUT(updateJobHandler), DELETE(deleteJobHandler))

	// Workflows
	router.Handle("/project/{permProjectKey}/workflows", POST(postWorkflowHandler), GET(getWorkflowsHandler))
	router.Handle("/project/{permProjectKey}/workflows/{workflowName}", GET(getWorkflowHandler), PUT(putWorkflowHandler), DELETE(deleteWorkflowHandler))
	// Workflows run
	router.Handle("/project/{permProjectKey}/workflows/{workflowName}/runs", GET(getWorkflowRunsHandler), POST(postWorkflowRunHandler))
	router.Handle("/project/{permProjectKey}/workflows/{workflowName}/runs/latest", GET(getLatestWorkflowRunHandler))
	router.Handle("/project/{permProjectKey}/workflows/{workflowName}/runs/{number}", GET(getWorkflowRunHandler))
	router.Handle("/project/{permProjectKey}/workflows/{workflowName}/runs/{number}/artifacts", GET(getWorkflowRunArtifactsHandler))
	router.Handle("/project/{permProjectKey}/workflows/{workflowName}/runs/{number}/nodes/{id}", GET(getWorkflowNodeRunHandler))
	router.Handle("/project/{permProjectKey}/workflows/{workflowName}/runs/{number}/nodes/{id}/job/{runJobId}/step/{stepOrder}", GET(getWorkflowNodeRunJobStepHandler))
	router.Handle("/project/{permProjectKey}/workflows/{workflowName}/runs/{number}/nodes/{id}/artifacts", GET(getWorkflowNodeRunArtifactsHandler))
	router.Handle("/project/{permProjectKey}/workflows/{workflowName}/artifact/{artifactId}", GET(getDownloadArtifactHandler))
	router.Handle("/project/{permProjectKey}/workflows/{workflowName}/node/{nodeID}/triggers/condition", GET(getWorkflowTriggerConditionHandler))
	router.Handle("/project/{permProjectKey}/workflows/{workflowName}/join/{joinID}/triggers/condition", GET(getWorkflowTriggerJoinConditionHandler))

	// DEPRECATED
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/action/{jobID}", PUT(updatePipelineActionHandler, DEPRECATED), DELETE(deleteJobHandler))

	router.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/joined", POST(addJobToPipelineHandler))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/joined/{actionID}", GET(getJoinedAction), PUT(updateJoinedAction), DELETE(deleteJoinedAction))
	router.Handle("/project/{key}/pipeline/{permPipelineKey}/stage/{stageID}/joined/{actionID}/audit", GET(getJoinedActionAudithandler))

	// Triggers
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/trigger", GET(getTriggersHandler), POST(addTriggerHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/trigger/source", GET(getTriggersAsSourceHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/trigger/{id}", GET(getTriggerHandler), DELETE(deleteTriggerHandler), PUT(updateTriggerHandler))

	// Environment
	router.Handle("/project/{permProjectKey}/environment", GET(getEnvironmentsHandler), POST(addEnvironmentHandler), PUT(updateEnvironmentsHandler, DEPRECATED))
	router.Handle("/project/{permProjectKey}/environment/import", POST(importNewEnvironmentHandler))
	router.Handle("/project/{permProjectKey}/environment/import/{permEnvironmentName}", POST(importIntoEnvironmentHandler))
	router.Handle("/project/{key}/environment/{permEnvironmentName}", GET(getEnvironmentHandler), PUT(updateEnvironmentHandler), DELETE(deleteEnvironmentHandler))
	router.Handle("/project/{key}/environment/{permEnvironmentName}/keys", GET(getKeysInEnvironmentHandler), POST(addKeyInEnvironmentHandler))
	router.Handle("/project/{key}/environment/{permEnvironmentName}/keys/{name}", DELETE(deleteKeyInEnvironmentHandler))
	router.Handle("/project/{key}/environment/{permEnvironmentName}/clone/{cloneName}", POST(cloneEnvironmentHandler))
	router.Handle("/project/{key}/environment/{permEnvironmentName}/audit", GET(getEnvironmentsAuditHandler, DEPRECATED))
	router.Handle("/project/{key}/environment/{permEnvironmentName}/audit/{auditID}", PUT(restoreEnvironmentAuditHandler, DEPRECATED))
	router.Handle("/project/{key}/environment/{permEnvironmentName}/group", POST(addGroupInEnvironmentHandler))
	router.Handle("/project/{key}/environment/{permEnvironmentName}/groups", POST(addGroupsInEnvironmentHandler))
	router.Handle("/project/{key}/environment/{permEnvironmentName}/group/{group}", PUT(updateGroupRoleOnEnvironmentHandler), DELETE(deleteGroupFromEnvironmentHandler))
	router.Handle("/project/{key}/environment/{permEnvironmentName}/variable", GET(getVariablesInEnvironmentHandler))
	router.Handle("/project/{key}/environment/{permEnvironmentName}/variable/{name}", GET(getVariableInEnvironmentHandler), POST(addVariableInEnvironmentHandler), PUT(updateVariableInEnvironmentHandler), DELETE(deleteVariableFromEnvironmentHandler))
	router.Handle("/project/{key}/environment/{permEnvironmentName}/variable/{name}/audit", GET(getVariableAuditInEnvironmentHandler))

	// Artifacts
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/artifact/{tag}", GET(listArtifactsHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/{buildNumber}/artifact", GET(listArtifactsBuildHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/{buildNumber}/artifact/{tag}", POSTEXECUTE(uploadArtifactHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/artifact/download/{id}", GET(downloadArtifactHandler))
	router.Handle("/artifact/{hash}", GET(downloadArtifactDirectHandler, Auth(false)))

	// Hooks
	router.Handle("/project/{key}/application/{permApplicationName}/hook", GET(getApplicationHooksHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/hook", POST(addHook), GET(getHooks))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/hook/{id}", PUT(updateHookHandler), DELETE(deleteHook))

	// Pollers
	router.Handle("/project/{key}/application/{permApplicationName}/polling", GET(getApplicationPollersHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/pipeline/{permPipelineKey}/polling", POST(addPollerHandler), GET(getPollersHandler), PUT(updatePollerHandler), DELETE(deletePollerHandler))

	// Build queue
	router.Handle("/queue", GET(getQueueHandler))
	router.Handle("/queue/{id}/take", POST(takePipelineBuildJobHandler))
	router.Handle("/queue/{id}/book", POST(bookPipelineBuildJobHandler, NeedHatchery()))
	router.Handle("/queue/{id}/spawn/infos", POST(addSpawnInfosPipelineBuildJobHandler, NeedWorker(), NeedHatchery()))
	router.Handle("/queue/{id}/result", POST(addQueueResultHandler))
	router.Handle("/queue/{id}/infos", GET(getPipelineBuildJobHandler))
	router.Handle("/build/{id}/log", POST(addBuildLogHandler))
	router.Handle("/build/{id}/step", POST(updateStepStatusHandler))

	//Workflow queue
	router.Handle("/queue/workflows", GET(getWorkflowJobQueueHandler))
	router.Handle("/queue/workflows/requirements/errors", POST(postWorkflowJobRequirementsErrorHandler, NeedWorker()))
	router.Handle("/queue/workflows/{id}/take", POST(postTakeWorkflowJobHandler, NeedWorker()))
	router.Handle("/queue/workflows/{id}/book", POST(postBookWorkflowJobHandler, NeedHatchery()))
	router.Handle("/queue/workflows/{id}/infos", GET(getWorkflowJobHandler, NeedWorker()))
	router.Handle("/queue/workflows/{id}/spawn/infos", POST(postSpawnInfosWorkflowJobHandler, NeedHatchery()))
	router.Handle("/queue/workflows/{permID}/result", POSTEXECUTE(postWorkflowJobResultHandler, NeedWorker()))
	router.Handle("/queue/workflows/{permID}/log", POSTEXECUTE(postWorkflowJobLogsHandler, NeedWorker()))
	router.Handle("/queue/workflows/{permID}/test", POSTEXECUTE(postWorkflowJobTestsResultsHandler, NeedWorker()))
	router.Handle("/queue/workflows/{permID}/variable", POSTEXECUTE(postWorkflowJobVariableHandler, NeedWorker()))
	router.Handle("/queue/workflows/{permID}/step", POSTEXECUTE(postWorkflowJobStepStatusHandler, NeedWorker()))
	router.Handle("/queue/workflows/{permID}/artifact/{tag}", POSTEXECUTE(postWorkflowJobArtifactHandler, NeedWorker()))

	router.Handle("/variable/type", GET(getVariableTypeHandler))
	router.Handle("/parameter/type", GET(getParameterTypeHandler))
	router.Handle("/pipeline/type", GET(getPipelineTypeHandler))
	router.Handle("/notification/type", GET(getUserNotificationTypeHandler))
	router.Handle("/notification/state", GET(getUserNotificationStateValueHandler))

	// RepositoriesManager
	router.Handle("/repositories_manager", GET(getRepositoriesManagerHandler))
	router.Handle("/repositories_manager/add", POST(addRepositoriesManagerHandler, NeedAdmin(true)))
	router.Handle("/repositories_manager/oauth2/callback", GET(repositoriesManagerOAuthCallbackHandler, Auth(false)))
	// RepositoriesManager for projects
	router.Handle("/project/{permProjectKey}/repositories_manager", GET(getRepositoriesManagerForProjectHandler))
	router.Handle("/project/{permProjectKey}/repositories_manager/{name}/authorize", POST(repositoriesManagerAuthorize))
	router.Handle("/project/{permProjectKey}/repositories_manager/{name}/authorize/callback", POST(repositoriesManagerAuthorizeCallback))
	router.Handle("/project/{permProjectKey}/repositories_manager/{name}", DELETE(deleteRepositoriesManagerHandler))
	router.Handle("/project/{permProjectKey}/repositories_manager/{name}/repo", GET(getRepoFromRepositoriesManagerHandler))
	router.Handle("/project/{permProjectKey}/repositories_manager/{name}/repos", GET(getReposFromRepositoriesManagerHandler))

	// RepositoriesManager for applications
	router.Handle("/project/{permProjectKey}/repositories_manager/{name}/application", POST(addApplicationFromRepositoriesManagerHandler))
	router.Handle("/project/{key}/repositories_manager/{name}/application/{permApplicationName}/attach", POST(attachRepositoriesManager))
	router.Handle("/project/{key}/repositories_manager/{name}/application/{permApplicationName}/detach", POST(detachRepositoriesManager))
	router.Handle("/project/{key}/application/{permApplicationName}/repositories_manager", GET(getRepositoriesManagerForApplicationsHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/repositories_manager/{name}/hook", POST(addHookOnRepositoriesManagerHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/repositories_manager/hook/{hookId}", DELETE(deleteHookOnRepositoriesManagerHandler))

	// Suggest
	router.Handle("/suggest/variable/{permProjectKey}", GET(getVariablesHandler))

	// Templates
	router.Handle("/template", GET(getTemplatesHandler, Auth(false)))
	router.Handle("/template/add", POST(addTemplateHandler, NeedAdmin(true)))
	router.Handle("/template/build", GET(getBuildTemplatesHandler, Auth(false)))
	router.Handle("/template/deploy", GET(getDeployTemplatesHandler, Auth(false)))
	router.Handle("/template/{id}", PUT(updateTemplateHandler, NeedAdmin(true)), DELETE(deleteTemplateHandler, NeedAdmin(true)))
	router.Handle("/project/{permProjectKey}/template", POST(applyTemplateHandler))
	router.Handle("/project/{key}/application/{permApplicationName}/template", POST(applyTemplateOnApplicationHandler))

	// UI
	router.Handle("/config/user", GET(ConfigUserHandler, Auth(true)))

	// Users
	router.Handle("/user", GET(GetUsers))
	router.Handle("/user/signup", POST(AddUser, Auth(false)))
	router.Handle("/user/import", POST(importUsersHandler, NeedAdmin(true)))
	router.Handle("/user/{username}", GET(GetUserHandler, NeedUsernameOrAdmin(true)), PUT(UpdateUserHandler, NeedUsernameOrAdmin(true)), DELETE(DeleteUserHandler, NeedUsernameOrAdmin(true)))
	router.Handle("/user/{username}/groups", GET(getUserGroupsHandler, NeedUsernameOrAdmin(true)))
	router.Handle("/user/{username}/confirm/{token}", GET(ConfirmUser, Auth(false)))
	router.Handle("/user/{username}/reset", POST(ResetUser, Auth(false)))
	router.Handle("/auth/mode", GET(AuthModeHandler, Auth(false)))

	// Workers
	router.Handle("/worker", GET(getWorkersHandler, Auth(false)), POST(registerWorkerHandler, Auth(false)))
	router.Handle("/worker/refresh", POST(refreshWorkerHandler))
	router.Handle("/worker/checking", POST(workerCheckingHandler))
	router.Handle("/worker/waiting", POST(workerWaitingHandler))
	router.Handle("/worker/unregister", POST(unregisterWorkerHandler))
	router.Handle("/worker/{id}/disable", POST(disableWorkerHandler))

	// Worker models
	router.Handle("/worker/model", POST(addWorkerModel), GET(getWorkerModels))
	router.Handle("/worker/model/error/{permModelID}", PUT(spawnErrorWorkerModelHandler, NeedHatchery()))
	router.Handle("/worker/model/enabled", GET(getWorkerModelsEnabled))
	router.Handle("/worker/model/type", GET(getWorkerModelTypes))
	router.Handle("/worker/model/communication", GET(getWorkerModelCommunications))
	router.Handle("/worker/model/{permModelID}", PUT(updateWorkerModel), DELETE(deleteWorkerModel))
	router.Handle("/worker/model/capability/type", GET(getWorkerModelCapaTypes))

	// Workflows
	router.Handle("/workflow/hook", GET(getWorkflowHookModelsHandler))
	router.Handle("/workflow/hook/{model}", GET(getWorkflowHookModelHandler), POST(postWorkflowHookModelHandler, NeedAdmin(true)), PUT(putWorkflowHookModelHandler, NeedAdmin(true)))

	// SSE
	router.Handle("/mon/lastupdates/events", GET(lastUpdateBroker.ServeHTTP))

	//Not Found handler
	router.mux.NotFoundHandler = http.HandlerFunc(notFoundHandler)
}
