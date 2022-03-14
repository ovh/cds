package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/mitchellh/hashstructure"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getWorkflowNodeRunJobServiceLinkHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return api.getWorkflowNodeRunJobLogLinkHandler(ctx, w, r, sdk.CDNTypeItemServiceLog)
	}
}

func (api *API) getWorkflowNodeRunJobStepLinksHandler() service.Handler {
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

		jobRun, err := workflow.LoadRunByID(ctx, api.mustDB(), nodeRun.WorkflowRunID, workflow.LoadRunOptions{
			DisableDetailledNodeRun: true,
		})
		if err != nil {
			return err
		}

		refs := make([]sdk.CDNLogAPIRef, 0)
		apiRef := sdk.CDNLogAPIRef{
			ProjectKey:     projectKey,
			WorkflowName:   jobRun.Workflow.Name,
			WorkflowID:     jobRun.WorkflowID,
			RunID:          jobRun.ID,
			NodeRunName:    nodeRun.WorkflowNodeName,
			NodeRunID:      nodeRun.ID,
			NodeRunJobName: runJob.Job.Action.Name,
			NodeRunJobID:   runJob.ID,
		}

		for _, s := range runJob.Job.StepStatus {
			ref := apiRef
			ref.StepName = runJob.Job.Action.Actions[int64(s.StepOrder)].Name
			if runJob.Job.Action.Actions[int64(s.StepOrder)].StepName != "" {
				ref.StepName = runJob.Job.Action.Actions[int64(s.StepOrder)].StepName
			}
			ref.StepOrder = int64(s.StepOrder)
			refs = append(refs, ref)
		}

		datas := make([]sdk.CDNLogLinkData, 0, len(refs))

		for _, r := range refs {
			apiRefHashU, err := hashstructure.Hash(r, nil)
			if err != nil {
				return sdk.WithStack(err)
			}
			apiRefHash := strconv.FormatUint(apiRefHashU, 10)
			datas = append(datas, sdk.CDNLogLinkData{
				APIRef:    apiRefHash,
				StepOrder: r.StepOrder,
			})
		}

		return service.WriteJSON(w, sdk.CDNLogLinks{
			CDNURL:   httpURL,
			ItemType: sdk.CDNTypeItemStepLog,
			Data:     datas,
		}, http.StatusOK)
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

	jobRun, err := workflow.LoadRunByID(ctx, api.mustDB(), nodeRun.WorkflowRunID, workflow.LoadRunOptions{
		DisableDetailledNodeRun: true,
	})
	if err != nil {
		return err
	}

	apiRef := sdk.CDNLogAPIRef{
		ProjectKey:     projectKey,
		WorkflowName:   jobRun.Workflow.Name,
		WorkflowID:     jobRun.WorkflowID,
		RunID:          jobRun.ID,
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

func (api *API) getWorkflowAccessHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		projectKey := vars["key"]
		itemType := vars["type"]

		var enabled bool
		switch sdk.CDNItemType(itemType) {
		case sdk.CDNTypeItemStepLog, sdk.CDNTypeItemServiceLog:
			enabled = true
		case sdk.CDNTypeItemRunResult:
			enabled = true
		}

		if !enabled {
			return sdk.WrapError(sdk.ErrForbidden, "cdn is not enabled for type %s", itemType)
		}

		if !isCDN(ctx) {
			return sdk.WrapError(sdk.ErrForbidden, "only CDN can call this route")
		}

		sessionID := r.Header.Get(sdk.CDSSessionID)
		if sessionID == "" {
			return sdk.WrapError(sdk.ErrForbidden, "missing session id header")
		}

		workflowID, err := requestVarInt(r, "workflowID")
		if err != nil {
			return err
		}

		exists, err := workflow.ExistsID(ctx, api.mustDB(), projectKey, workflowID)
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

		// Add worker for consumer if exists
		worker, err := worker.LoadByConsumerID(ctx, api.mustDB(), consumer.ID)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
		}

		if worker != nil {
			jobRunID := worker.JobRunID
			if jobRunID != nil {
				nodeJobRun, err := workflow.LoadNodeJobRun(ctx, api.mustDB(), api.Cache, *jobRunID)
				if err != nil {
					return sdk.WrapError(sdk.ErrUnauthorized, "can't load node job run with id %q", *jobRunID)
				}

				nodeRun, err := workflow.LoadNodeRunByID(ctx, api.mustDB(), nodeJobRun.WorkflowNodeRunID, workflow.LoadRunOptions{})
				if err != nil {
					return sdk.WrapError(sdk.ErrUnauthorized, "can't load node run with id %q", nodeJobRun.WorkflowNodeRunID)
				}

				if nodeRun.WorkflowID == workflowID {
					return service.WriteJSON(w, nil, http.StatusOK)
				}

				daoSrc := workflow.LoadOptions{Minimal: true}.GetWorkflowDAO()
				daoSrc.Filters.WorkflowIDs = []int64{nodeRun.WorkflowID}
				workerSrcWf, err := daoSrc.Load(ctx, api.mustDB())
				if err != nil {
					return sdk.WrapError(err, "can't load worker source workflow with id %d on project %q", nodeRun.WorkflowID, projectKey)
				}

				// Allow workers to download artifact from other workflow inside the same project
				if projectKey == workerSrcWf.ProjectKey && sdk.CDNItemType(itemType) == sdk.CDNTypeItemRunResult {
					return service.WriteJSON(w, nil, http.StatusOK)
				}
			}

			return sdk.WrapError(sdk.ErrUnauthorized, "worker %q(%s) not authorized for workflow with id %d", worker.Name, worker.ID, workflowID)
		}

		maintainerOrAdmin := consumer.Maintainer() || consumer.Admin()

		perms, err := permission.LoadWorkflowMaxLevelPermissionByWorkflowIDs(ctx, api.mustDB(), []int64{workflowID}, consumer.GetGroupIDs())
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
		}
		workflowIDString := strconv.FormatInt(workflowID, 10)
		maxLevelPermission := perms.Level(workflowIDString)
		if maxLevelPermission < sdk.PermissionRead && !maintainerOrAdmin {
			return sdk.WrapError(sdk.ErrUnauthorized, "not authorized for workflow %s/%s", projectKey, workflowIDString)
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}
