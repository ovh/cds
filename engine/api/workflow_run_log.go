package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/mitchellh/hashstructure"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getWorkflowNodeRunJobServiceLogDeprecatedHandler() service.Handler {
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

func (api *API) getWorkflowNodeRunJobStepDeprecatedHandler() service.Handler {
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

func (api *API) getWorkflowNodeRunJobServiceLogHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		workflowName := vars["permWorkflowName"]
		serviceName := vars["serviceName"]

		enabled := featureflipping.IsEnabled(ctx, gorpmapping.Mapper, api.mustDB(), "cdn-job-logs", map[string]string{
			"project_key": projectKey,
		})
		if !enabled {
			return service.WriteJSON(w, sdk.CDNLogAccess{}, http.StatusOK)
		}

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

		// Try to load node run from given data
		nodeRun, err := workflow.LoadNodeRun(api.mustDB(), projectKey, workflowName, nodeRunID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if err != nil {
			return sdk.WrapError(err, "cannot find nodeRun %d for workflow %s in project %s", nodeRunID, workflowName, projectKey)
		}

		// Find job in nodeRun
		var runJob sdk.WorkflowNodeJobRun
	stageLoop:
		for _, s := range nodeRun.Stages {
			for _, rj := range s.RunJobs {
				if rj.ID != runJobID {
					continue
				}
				runJob = rj
				break stageLoop
			}
		}

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

		apiRef := sdk.CDNLogAPIRef{
			ProjectKey:             projectKey,
			WorkflowName:           workflowName,
			WorkflowID:             nodeRun.WorkflowID,
			RunID:                  nodeRun.WorkflowRunID,
			NodeRunName:            nodeRun.WorkflowNodeName,
			NodeRunID:              nodeRun.ID,
			NodeRunJobName:         runJob.Job.Action.Name,
			NodeRunJobID:           runJob.ID,
			RequirementServiceID:   req.ID,
			RequirementServiceName: req.Name,
		}
		apiRefHashU, err := hashstructure.Hash(apiRef, nil)
		if err != nil {
			return sdk.WithStack(err)
		}
		apiRefHash := strconv.FormatUint(apiRefHashU, 10)

		srvs, err := services.LoadAllByType(ctx, api.mustDB(), sdk.TypeCDN)
		if err != nil {
			return err
		}
		if len(srvs) == 0 {
			return sdk.WrapError(sdk.ErrNotFound, "no cdn service found")
		}
		_, code, err := services.NewClient(api.mustDB(), srvs).DoJSONRequest(ctx, http.MethodGet, "/item/service-log/"+apiRefHash, nil, nil)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		if code != http.StatusOK {
			return service.WriteJSON(w, sdk.CDNLogAccess{}, http.StatusOK)
		}

		tokenRaw, err := authentication.SignJWS(sdk.CDNAuthToken{APIRefHash: apiRefHash}, time.Hour)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, sdk.CDNLogAccess{
			Exists:       true,
			Token:        tokenRaw,
			DownloadPath: fmt.Sprintf("/item/service-log/%s/download", apiRefHash),
			CDNURL:       httpURL,
		}, http.StatusOK)
	}
}

func (api *API) getWorkflowNodeRunJobStepHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		workflowName := vars["permWorkflowName"]

		enabled := featureflipping.IsEnabled(ctx, gorpmapping.Mapper, api.mustDB(), "cdn-job-logs", map[string]string{
			"project_key": projectKey,
		})
		if !enabled {
			return service.WriteJSON(w, sdk.CDNLogAccess{}, http.StatusOK)
		}

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

		httpURL, err := services.GetCDNPublicHTTPAdress(ctx, api.mustDB())
		if err != nil {
			return err
		}

		// Try to load node run from given data
		nodeRun, err := workflow.LoadNodeRun(api.mustDB(), projectKey, workflowName, nodeRunID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if err != nil {
			return sdk.WrapError(err, "cannot find nodeRun %d for workflow %s in project %s", nodeRunID, workflowName, projectKey)
		}

		// Find job/step in nodeRun
		var stepStatus string
		var runJob sdk.WorkflowNodeJobRun
	stageLoop:
		for _, s := range nodeRun.Stages {
			for _, rj := range s.RunJobs {
				if rj.ID != runJobID {
					continue
				}
				runJob = rj
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

		apiRef := sdk.CDNLogAPIRef{
			ProjectKey:     projectKey,
			WorkflowName:   workflowName,
			WorkflowID:     nodeRun.WorkflowID,
			RunID:          nodeRun.WorkflowRunID,
			NodeRunName:    nodeRun.WorkflowNodeName,
			NodeRunID:      nodeRun.ID,
			NodeRunJobName: runJob.Job.Action.Name,
			NodeRunJobID:   runJob.ID,
			StepName:       runJob.Job.Action.Actions[stepOrder].Name,
			StepOrder:      stepOrder,
		}
		apiRefHashU, err := hashstructure.Hash(apiRef, nil)
		if err != nil {
			return sdk.WithStack(err)
		}
		apiRefHash := strconv.FormatUint(apiRefHashU, 10)

		srvs, err := services.LoadAllByType(ctx, api.mustDB(), sdk.TypeCDN)
		if err != nil {
			return err
		}
		if len(srvs) == 0 {
			return sdk.WrapError(sdk.ErrNotFound, "no cdn service found")
		}
		_, code, err := services.NewClient(api.mustDB(), srvs).DoJSONRequest(ctx, http.MethodGet, "/item/step-log/"+apiRefHash, nil, nil)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		if code != http.StatusOK {
			return service.WriteJSON(w, sdk.CDNLogAccess{}, http.StatusOK)
		}

		tokenRaw, err := authentication.SignJWS(sdk.CDNAuthToken{APIRefHash: apiRefHash}, time.Hour)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, sdk.CDNLogAccess{
			Exists:       true,
			Token:        tokenRaw,
			DownloadPath: fmt.Sprintf("/item/step-log/%s/download", apiRefHash),
			CDNURL:       httpURL,
		}, http.StatusOK)
	}
}
