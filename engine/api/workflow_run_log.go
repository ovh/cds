package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/mitchellh/hashstructure"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getWorkflowNodeRunJobServiceLogHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		runJobID, err := requestVarInt(r, "runJobID")
		if err != nil {
			return sdk.WrapError(err, "invalid run job id")
		}
		serviceName := vars["serviceName"]

		logsService, err := workflow.LoadServiceLog(api.mustDB(), runJobID, serviceName)
		if err != nil {
			return sdk.WrapError(err, "cannot load service log for node run job id %d and name %s", runJobID, serviceName)
		}

		ls := &sdk.ServiceLog{}
		if logsService != nil {
			ls = logsService
		}

		return service.WriteJSON(w, ls, http.StatusOK)
	}
}

func (api *API) getWorkflowNodeRunJobStepLogHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		workflowName := vars["permWorkflowName"]

		nodeRunID, err := requestVarInt(r, "nodeRunID")
		if err != nil {
			return sdk.NewErrorFrom(err, "invalid node run id")
		}
		runJobID, err := requestVarInt(r, "runJobID")
		if err != nil {
			return sdk.NewErrorFrom(err, "invalid node job id")
		}
		stepOrder, err := requestVarInt(r, "stepOrder")
		if err != nil {
			return sdk.NewErrorFrom(err, "invalid step order")
		}

		// Check nodeRunID is link to workflow
		nodeRun, err := workflow.LoadNodeRun(api.mustDB(), projectKey, workflowName, nodeRunID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if err != nil {
			return sdk.WrapError(err, "cannot find nodeRun %d for workflow %s in project %s", nodeRunID, workflowName, projectKey)
		}

		var stepStatus string
		// Find job/step in nodeRun
	stageLoop:
		for _, s := range nodeRun.Stages {
			for _, rj := range s.RunJobs {
				if rj.ID != runJobID {
					continue
				}
				ss := rj.Job.StepStatus
				for _, sss := range ss {
					if int64(sss.StepOrder) == stepOrder {
						stepStatus = sss.Status
						break
					}
				}
				break stageLoop
			}
		}

		if stepStatus == "" {
			return sdk.WrapError(sdk.ErrStepNotFound, "cannot find step %d on job %d in nodeRun %d for workflow %s in project %s",
				stepOrder, runJobID, nodeRunID, workflowName, projectKey)
		}

		logs, err := workflow.LoadStepLogs(api.mustDB(), runJobID, stepOrder)
		if err != nil {
			return sdk.WrapError(err, "cannot load log for runJob %d on step %d", runJobID, stepOrder)
		}

		ls := &sdk.Log{}
		if logs != nil {
			ls = logs
		}
		result := &sdk.BuildState{
			Status:   stepStatus,
			StepLogs: *ls,
		}

		return service.WriteJSON(w, result, http.StatusOK)
	}
}

func (api *API) getWorkflowNodeRunJobServiceLinkHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return api.getWorkflowNodeRunJobLogLinkHandler(ctx, w, r, sdk.CDNTypeItemServiceLog)
	}
}

func (api *API) getWorkflowNodeRunJobStepLinkHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return api.getWorkflowNodeRunJobLogLinkHandler(ctx, w, r, sdk.CDNTypeItemStepLog)
	}
}

func (api *API) getWorkflowNodeRunJobLogLinkHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, itemType sdk.CDNItemType) error {
	vars := mux.Vars(r)

	projectKey := vars["key"]
	enabled := featureflipping.IsEnabled(ctx, gorpmapping.Mapper, api.mustDB(), "cdn-job-logs", map[string]string{
		"project_key": projectKey,
	})
	if !enabled {
		return sdk.NewErrorFrom(sdk.ErrNotFound, "cdn is not enable for project %s", projectKey)
	}

	workflowName := vars["permWorkflowName"]
	nodeRunID, err := requestVarInt(r, "nodeRunID")
	if err != nil {
		return sdk.NewErrorFrom(err, "invalid node run id")
	}
	runJobID, err := requestVarInt(r, "runJobID")
	if err != nil {
		return sdk.NewErrorFrom(err, "invalid node job id")
	}

	httpURL, err := services.GetCDNPublicHTTPAdress(ctx, api.mustDB())
	if err != nil {
		return err
	}

	nodeRun, err := workflow.LoadNodeRun(api.mustDB(), projectKey, workflowName, nodeRunID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
	if err != nil {
		return sdk.WrapError(err, "cannot find nodeRun %d for workflow %s in project %s", nodeRunID, workflowName, projectKey)
	}
	var runJob *sdk.WorkflowNodeJobRun
	for _, s := range nodeRun.Stages {
		for _, rj := range s.RunJobs {
			if rj.ID == runJobID {
				runJob = &rj
				break
			}
		}
		if runJob != nil {
			break
		}
	}
	if runJob == nil {
		return sdk.NewErrorFrom(sdk.ErrNotFound, "cannot find run job for id %d", runJobID)
	}

	apiRef := sdk.CDNLogAPIRef{
		ProjectKey:     projectKey,
		WorkflowName:   workflowName,
		WorkflowID:     nodeRun.WorkflowID,
		RunID:          nodeRun.WorkflowRunID,
		NodeRunName:    nodeRun.WorkflowNodeName,
		NodeRunID:      nodeRun.ID,
		NodeRunJobName: runJob.Job.Action.Name,
		NodeRunJobID:   runJob.ID,
	}

	if itemType == sdk.CDNTypeItemServiceLog {
		serviceName := vars["serviceName"]
		var req *sdk.Requirement
		for _, r := range runJob.Job.Action.Requirements {
			if r.Type == sdk.ServiceRequirement && r.Name == serviceName {
				req = &r
				break
			}
		}
		if req == nil {
			return sdk.NewErrorFrom(sdk.ErrNotFound, "cannot find logs for service with name %s", serviceName)
		}
		apiRef.RequirementServiceID = req.ID
		apiRef.RequirementServiceName = req.Name
	} else {
		stepOrder, err := requestVarInt(r, "stepOrder")
		if err != nil {
			return sdk.NewErrorFrom(err, "invalid step order")
		}
		var ss *sdk.StepStatus
		for _, s := range runJob.Job.StepStatus {
			if int64(s.StepOrder) == stepOrder {
				ss = &s
				break
			}
		}
		if ss == nil {
			return sdk.WrapError(sdk.ErrStepNotFound, "cannot find step %d on job %d in nodeRun %d for workflow %s in project %s",
				stepOrder, runJobID, nodeRunID, workflowName, projectKey)
		}
		apiRef.StepName = runJob.Job.Action.Actions[int64(ss.StepOrder)].Name
		if runJob.Job.Action.Actions[int64(ss.StepOrder)].StepName != "" {
			apiRef.StepName = runJob.Job.Action.Actions[int64(ss.StepOrder)].StepName
		}
		apiRef.StepOrder = int64(ss.StepOrder)
	}

	apiRefHashU, err := hashstructure.Hash(apiRef, nil)
	if err != nil {
		return sdk.WithStack(err)
	}
	apiRefHash := strconv.FormatUint(apiRefHashU, 10)

	return service.WriteJSON(w, sdk.CDNLogLink{
		CDNURL:   httpURL,
		ItemType: itemType,
		APIRef:   apiRefHash,
	}, http.StatusOK)
}

func (api *API) getWorkflowLogAccessHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		projectKey := vars["key"]
		enabled := featureflipping.IsEnabled(ctx, gorpmapping.Mapper, api.mustDB(), "cdn-job-logs", map[string]string{
			"project_key": projectKey,
		})
		if !enabled {
			return sdk.WrapError(sdk.ErrForbidden, "cdn is not enabled for project %s", projectKey)
		}

		if !isCDN(ctx) {
			return sdk.WrapError(sdk.ErrForbidden, "only CDN can call this route")
		}

		sessionID := r.Header.Get("X-CDS-Session-ID")
		if sessionID == "" {
			return sdk.WrapError(sdk.ErrForbidden, "missing session id header")
		}

		workflowName := vars["workflowName"]

		exists, err := workflow.Exists(api.mustDB(), projectKey, workflowName)
		if err != nil {
			return err
		}
		if !exists {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		session, err := authentication.LoadSessionByID(ctx, api.mustDBWithCtx(ctx), sessionID)
		if err != nil {
			return err
		}
		consumer, err := authentication.LoadConsumerByID(ctx, api.mustDB(), session.ConsumerID,
			authentication.LoadConsumerOptions.WithAuthentifiedUser)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
		}
		if consumer.Disabled {
			return sdk.WrapError(sdk.ErrUnauthorized, "consumer (%s) is disabled", consumer.ID)
		}

		maintainerOrAdmin := consumer.Maintainer() || consumer.Admin()

		perms, err := permission.LoadWorkflowMaxLevelPermission(ctx, api.mustDB(), projectKey, []string{workflowName}, consumer.GetGroupIDs())
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
		}
		maxLevelPermission := perms.Level(workflowName)
		if maxLevelPermission < sdk.PermissionRead && !maintainerOrAdmin {
			return sdk.WrapError(sdk.ErrUnauthorized, "not authorized for workflow %s/%s", projectKey, workflowName)
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}
