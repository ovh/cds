package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/luascript"
	"github.com/ovh/cds/sdk/telemetry"
)

const (
	defaultLimit = 10
)

func (api *API) searchWorkflowRun(ctx context.Context, w http.ResponseWriter, r *http.Request, route, key, name string) error {
	// About pagination: [FR] http://blog.octo.com/designer-une-api-rest/#pagination
	var limit, offset int

	offsetS := r.FormValue("offset")
	var errAtoi error
	if offsetS != "" {
		offset, errAtoi = strconv.Atoi(offsetS)
		if errAtoi != nil {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}
	}
	limitS := r.FormValue("limit")
	if limitS != "" {
		limit, errAtoi = strconv.Atoi(limitS)
		if errAtoi != nil {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}
	}

	if offset < 0 {
		offset = 0
	}
	if limit == 0 {
		limit = defaultLimit
	}

	//Parse all form values
	mapFilters := map[string]string{}
	for k := range r.Form {
		if k != "offset" && k != "limit" && k != "workflow" {
			mapFilters[k] = r.FormValue(k)
		}
	}

	//Maximim range is set to 50
	w.Header().Add("Accept-Range", "run 50")
	runs, offset, limit, count, err := workflow.LoadRunsSummaries(ctx, api.mustDB(), key, name, offset, limit, mapFilters)
	if err != nil {
		return sdk.WrapError(err, "Unable to load workflow runs")
	}

	code := http.StatusOK

	//RFC5988: Link : <https://api.fakecompany.com/v1/orders?range=0-7>; rel="first", <https://api.fakecompany.com/v1/orders?range=40-47>; rel="prev", <https://api.fakecompany.com/v1/orders?range=56-64>; rel="next", <https://api.fakecompany.com/v1/orders?range=968-975>; rel="last"
	if len(runs) < count {
		baseLinkURL := api.Router.URL + route
		code = http.StatusPartialContent

		//First page
		firstLimit := limit - offset
		if firstLimit > count {
			firstLimit = count
		}
		firstLink := fmt.Sprintf(`<%s?offset=0&limit=%d>; rel="first"`, baseLinkURL, firstLimit)
		link := firstLink

		//Prev page
		if offset != 0 {
			prevOffset := offset - (limit - offset)
			prevLimit := offset
			if prevOffset < 0 {
				prevOffset = 0
			}
			prevLink := fmt.Sprintf(`<%s?offset=%d&limit=%d>; rel="prev"`, baseLinkURL, prevOffset, prevLimit)
			link = link + ", " + prevLink
		}

		//Next page
		if limit < count {
			nextOffset := limit
			nextLimit := limit + (limit - offset)

			if nextLimit >= count {
				nextLimit = count
			}

			nextLink := fmt.Sprintf(`<%s?offset=%d&limit=%d>; rel="next"`, baseLinkURL, nextOffset, nextLimit)
			link = link + ", " + nextLink
		}

		//Last page
		lastOffset := count - (limit - offset)
		if lastOffset < 0 {
			lastOffset = 0
		}
		lastLimit := count
		lastLink := fmt.Sprintf(`<%s?offset=%d&limit=%d>; rel="last"`, baseLinkURL, lastOffset, lastLimit)
		link = link + ", " + lastLink

		w.Header().Add("Link", link)
	}

	w.Header().Add("Content-Range", fmt.Sprintf("%d-%d/%d", offset, limit, count))

	// Return empty array instead of nil
	if runs == nil {
		runs = []sdk.WorkflowRunSummary{}
	}
	return service.WriteJSON(w, runs, code)
}

func (api *API) getWorkflowAllRunsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		name := r.FormValue("workflow")
		route := api.Router.GetRoute("GET", api.getWorkflowAllRunsHandler, map[string]string{
			"permProjectKey": key,
		})
		return api.searchWorkflowRun(ctx, w, r, route, key, name)
	}
}

func (api *API) getWorkflowRunsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowNameAdvanced"]
		route := api.Router.GetRoute("GET", api.getWorkflowRunsHandler, map[string]string{
			"key":                      key,
			"permWorkflowNameAdvanced": name,
		})
		return api.searchWorkflowRun(ctx, w, r, route, key, name)
	}
}

func (api *API) deleteWorkflowRunsBranchHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		branch := vars["branch"]

		wfIDs, err := workflow.LoadRunsIDByTag(api.mustDB(), key, name, "git.branch", branch)
		if err != nil {
			return err
		}

		if err := workflow.MarkWorkflowRunsAsDelete(api.mustDB(), wfIDs); err != nil {
			return err
		}

		workflow.CountWorkflowRunsMarkToDelete(ctx, api.mustDB(), api.Metrics.WorkflowRunsMarkToDelete)

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

// getWorkflowRunNumHandler returns the last run number for the given workflow
func (api *API) getWorkflowRunNumHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		num, err := workflow.LoadCurrentRunNum(api.mustDB(), key, name)
		if err != nil {
			return sdk.WrapError(err, "Cannot load current run num")
		}

		return service.WriteJSON(w, sdk.WorkflowRunNumber{Num: num}, http.StatusOK)
	}
}

// postWorkflowRunNumHandler updates the current run number for the given workflow
func (api *API) postWorkflowRunNumHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		m := struct {
			Num int64 `json:"num"`
		}{}

		if err := service.UnmarshalBody(r, &m); err != nil {
			return sdk.WithStack(err)
		}

		num, err := workflow.LoadCurrentRunNum(api.mustDB(), key, name)
		if err != nil {
			return sdk.WrapError(err, "cannot load current run num")
		}

		if m.Num < num {
			return sdk.WrapError(sdk.ErrWrongRequest, "cannot num must be > %d, got %d", num, m.Num)
		}

		proj, err := project.Load(ctx, api.mustDB(), key, project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load project")
		}

		options := workflow.LoadOptions{}
		wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, *proj, name, options)
		if err != nil {
			return sdk.WrapError(err, "cannot load workflow")
		}

		if num == 0 {
			err = workflow.InsertRunNum(api.mustDB(), wf, m.Num)
		} else {
			err = workflow.UpdateRunNum(api.mustDB(), wf, m.Num)
		}
		if err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, m, http.StatusOK)
	}
}

func (api *API) getLatestWorkflowRunHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		run, err := workflow.LoadLastRun(ctx, api.mustDB(), key, name, workflow.LoadRunOptions{})
		if err != nil {
			return sdk.WrapError(err, "Unable to load last workflow run")
		}
		api.setWorkflowRunURLs(run)
		run.Translate()
		return service.WriteJSON(w, run, http.StatusOK)
	}
}

func (api *API) getWorkflowRunHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowNameAdvanced"]
		if name == "" {
			name = vars["permWorkflowName"] // Useful for workflowv3 routes
		}
		number, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}

		withDetailledNodeRun := QueryString(r, "withDetails")

		isService := isService(ctx)

		// loadRun, DisableDetailledNodeRun = false for calls from CDS Service
		// as hook service. It's needed to have the buildParameters.
		run, err := workflow.LoadRun(ctx, api.mustDB(), key, name, number,
			workflow.LoadRunOptions{
				WithDeleted:             false,
				WithLightTests:          true,
				DisableDetailledNodeRun: !isService && withDetailledNodeRun != "true",
			},
		)
		if err != nil {
			return sdk.WrapError(err, "Unable to load workflow %s run number %d", name, number)
		}

		// Remove unused data
		for i := range run.WorkflowNodeRuns {
			for j := range run.WorkflowNodeRuns[i] {
				nr := &run.WorkflowNodeRuns[i][j]
				for si := range nr.Stages {
					s := &nr.Stages[si]
					for rji := range s.RunJobs {
						rj := &s.RunJobs[rji]
						rj.Parameters = nil
					}
				}
			}
		}

		api.setWorkflowRunURLs(run)
		run.Translate()

		return service.WriteJSON(w, run, http.StatusOK)
	}
}

func (api *API) deleteWorkflowRunHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if isHooks(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowNameAdvanced"]
		number, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}

		run, err := workflow.LoadRun(ctx, api.mustDB(), key, name, number,
			workflow.LoadRunOptions{
				DisableDetailledNodeRun: true,
			},
		)
		if err != nil {
			return sdk.WrapError(err, "Unable to load workflow %s run number %d", name, number)
		}

		if err := workflow.MarkWorkflowRunsAsDelete(api.mustDB(), []int64{run.ID}); err != nil {
			return sdk.WrapError(err, "cannot mark workflow run %d as delete", run.ID)
		}
		run.ToDelete = true

		workflow.CountWorkflowRunsMarkToDelete(ctx, api.mustDB(), api.Metrics.WorkflowRunsMarkToDelete)
		event.PublishWorkflowRun(ctx, *run, key)

		return service.WriteJSON(w, nil, http.StatusAccepted)
	}
}

func (api *API) stopWorkflowRunHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		number, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}

		consumer := getUserConsumer(ctx)

		// This POST exec handler should not be called by workers
		if consumer.AuthConsumerUser.Worker != nil {
			return sdk.WrapError(sdk.ErrForbidden, "not authorized for worker")
		}

		run, err := workflow.LoadRun(ctx, api.mustDB(), key, name, number, workflow.LoadRunOptions{
			WithDeleted: true,
		})
		if err != nil {
			return sdk.WrapError(err, "unable to load last workflow run")
		}

		proj, err := project.Load(ctx, api.mustDB(), key)
		if err != nil {
			return sdk.WrapError(err, "unable to load project")
		}

		report, err := api.stopWorkflowRun(ctx, proj, run, 0)
		if err != nil {
			return sdk.WrapError(err, "unable to stop workflow")
		}

		go api.WorkflowSendEvent(context.Background(), *proj, report)

		go func(ID int64) {
			wRun, err := workflow.LoadRunByID(ctx, api.mustDB(), ID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
			if err != nil {
				log.Error(ctx, "workflow.stopWorkflowNodeRun> Cannot load run for resync commit status %v", err)
				return
			}
			// The function could be called with nil project so we need to test if project is not nil
			if sdk.StatusIsTerminated(wRun.Status) && proj != nil {
				wRun.LastExecution = time.Now()
				if err := workflow.ResyncCommitStatus(context.Background(), api.mustDB(), api.Cache, *proj, wRun, api.Config.URL.UI); err != nil {
					log.Error(ctx, "workflow.UpdateNodeJobRunStatus> %v", err)
				}
			}
		}(run.ID)

		workflowRuns := report.WorkflowRuns()
		if len(workflowRuns) > 0 {
			telemetry.Current(ctx,
				telemetry.Tag(telemetry.TagProjectKey, proj.Key),
				telemetry.Tag(telemetry.TagWorkflow, workflowRuns[0].Workflow.Name),
			)
			if workflowRuns[0].Status == sdk.StatusFail {
				telemetry.Record(api.Router.Background, api.Metrics.WorkflowRunFailed, 1)
			}
		}

		return service.WriteJSON(w, run, http.StatusOK)
	}
}

func (api *API) stopWorkflowRun(ctx context.Context, p *sdk.Project, run *sdk.WorkflowRun, parentWorkflowRunID int64) (*workflow.ProcessorReport, error) {
	ident := getUserConsumer(ctx)
	report := new(workflow.ProcessorReport)

	tx, err := api.mustDB().Begin()
	if err != nil {
		return nil, sdk.WrapError(err, "unable to create transaction")
	}
	defer tx.Rollback() //nolint

	spwnMsg := sdk.SpawnMsgNew(*sdk.MsgWorkflowNodeStop, ident.GetUsername())
	workflow.AddWorkflowRunInfo(run, spwnMsg)

	for _, wn := range run.WorkflowNodeRuns {
		for _, wnr := range wn {
			if wnr.SubNumber != run.LastSubNumber || (wnr.Status == sdk.StatusSuccess ||
				wnr.Status == sdk.StatusFail || wnr.Status == sdk.StatusSkipped) {
				log.Debug(ctx, "stopWorkflowRun> cannot stop this workflow node run with current status %s", wnr.Status)
				continue
			}

			r1, err := workflow.StopWorkflowNodeRun(ctx, api.mustDB, api.Cache, *p, *run, wnr, sdk.SpawnInfo{
				Message: spwnMsg,
			})
			if err != nil {
				return nil, sdk.WrapError(err, "unable to stop workflow node run %d", wnr.ID)
			}
			report.Merge(ctx, r1)
			wnr.Status = sdk.StatusStopped

			// If it's a outgoing hook, we stop the child
			if wnr.OutgoingHook != nil {
				if run.Workflow.OutGoingHookModels == nil {
					run.Workflow.OutGoingHookModels = make(map[int64]sdk.WorkflowHookModel)
				}
				model, has := run.Workflow.OutGoingHookModels[wnr.OutgoingHook.HookModelID]
				if !has {
					m, errM := workflow.LoadOutgoingHookModelByID(api.mustDB(), wnr.OutgoingHook.HookModelID)
					if errM != nil {
						log.Error(ctx, "stopWorkflowRun> Unable to load outgoing hook model: %v", errM)
						continue
					}
					model = *m
					run.Workflow.OutGoingHookModels[wnr.OutgoingHook.HookModelID] = *m
				}
				if model.Name == sdk.WorkflowModelName && wnr.Callback != nil && wnr.Callback.WorkflowRunNumber != nil {
					//Stop trigggered workflow
					targetProject := wnr.OutgoingHook.Config[sdk.HookConfigTargetProject].Value
					targetWorkflow := wnr.OutgoingHook.Config[sdk.HookConfigTargetWorkflow].Value

					targetRun, errL := workflow.LoadRun(ctx, api.mustDB(), targetProject, targetWorkflow, *wnr.Callback.WorkflowRunNumber, workflow.LoadRunOptions{})
					if errL != nil {
						log.Error(ctx, "stopWorkflowRun> Unable to load last workflow run: %v", errL)
						continue
					}

					targetProj, errP := project.Load(ctx, api.mustDB(), targetProject)
					if errP != nil {
						log.Error(ctx, "stopWorkflowRun> Unable to load project %v", errP)
						continue
					}

					r2, err := api.stopWorkflowRun(ctx, targetProj, targetRun, run.ID)
					if err != nil {
						log.Error(ctx, "stopWorkflowRun> Unable to stop workflow %v", err)
						continue
					}
					report.Merge(ctx, r2)
				}
			}
		}
	}

	run.LastExecution = time.Now()
	run.Status = sdk.StatusStopped
	if errU := workflow.UpdateWorkflowRun(ctx, tx, run); errU != nil {
		return nil, sdk.WrapError(errU, "Unable to update workflow run %d", run.ID)
	}
	report.Add(ctx, *run)

	if err := tx.Commit(); err != nil {
		return nil, sdk.WithStack(err)
	}

	if parentWorkflowRunID == 0 {
		if err := api.updateParentWorkflowRun(ctx, run); err != nil {
			return nil, sdk.WithStack(err)
		}
	}

	return report, nil
}

func (api *API) updateParentWorkflowRun(ctx context.Context, run *sdk.WorkflowRun) error {
	if !run.HasParentWorkflow() {
		return nil
	}

	tx, err := api.mustDB().Begin()
	if err != nil {
		return sdk.WrapError(err, "unable to start transaction")
	}
	defer tx.Rollback() //nolint

	parentProj, err := project.Load(context.Background(),
		tx, run.RootRun().HookEvent.ParentWorkflow.Key,
		project.LoadOptions.WithVariables,
		project.LoadOptions.WithIntegrations,
		project.LoadOptions.WithApplicationVariables,
		project.LoadOptions.WithApplicationWithDeploymentStrategies,
		project.LoadOptions.WithGroups,
	)
	if err != nil {
		return sdk.WrapError(err, "cannot load project")
	}

	parentWR, err := workflow.LoadRun(ctx,
		tx,
		run.RootRun().HookEvent.ParentWorkflow.Key,
		run.RootRun().HookEvent.ParentWorkflow.Name,
		run.RootRun().HookEvent.ParentWorkflow.Run,
		workflow.LoadRunOptions{
			DisableDetailledNodeRun: false,
		})
	if err != nil {
		return sdk.WrapError(err, "unable to load parent run: %v", run.RootRun().HookEvent)
	}

	report, err := workflow.UpdateParentWorkflowRun(ctx, tx, api.Cache, run, *parentProj, parentWR)
	if err != nil {
		return sdk.WithStack(err)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	go api.WorkflowSendEvent(context.Background(), *parentProj, report)

	// Recursively update the parent run
	return api.updateParentWorkflowRun(ctx, parentWR)
}

func (api *API) getWorkflowNodeRunHistoryHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		number, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}
		nodeID, err := requestVarInt(r, "nodeID")
		if err != nil {
			return err
		}

		run, errR := workflow.LoadRun(ctx, api.mustDB(), key, name, number, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if errR != nil {
			return sdk.WrapError(errR, "getWorkflowNodeRunHistoryHandler")
		}

		nodeRuns, ok := run.WorkflowNodeRuns[nodeID]
		if !ok {
			return sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "getWorkflowNodeRunHistoryHandler")
		}
		return service.WriteJSON(w, nodeRuns, http.StatusOK)
	}
}

func (api *API) getWorkflowCommitsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		nodeName := vars["nodeName"]
		remote := FormString(r, "remote")
		branch := FormString(r, "branch")
		hash := FormString(r, "hash")
		number, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}

		proj, errP := project.Load(ctx, api.mustDB(), key, project.LoadOptions.WithIntegrations)
		if errP != nil {
			return sdk.WrapError(errP, "getWorkflowCommitsHandler> Unable to load project %s", key)
		}

		var wf *sdk.Workflow
		wfRun, err := workflow.LoadRun(ctx, api.mustDB(), key, name, number, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if err != nil {
			wf, err = workflow.Load(ctx, api.mustDB(), api.Cache, *proj, name, workflow.LoadOptions{})
			if err != nil {
				return sdk.WrapError(err, "unable to load workflow %s", name)
			}
		} else {
			wf = &wfRun.Workflow
		}

		var app sdk.Application
		var env sdk.Environment
		var node *sdk.Node
		if wf != nil {
			node = wf.WorkflowData.NodeByName(nodeName)
			if node == nil {
				return sdk.WrapError(sdk.ErrNotFound, "unable to load workflow data node")
			}
			if node.Context != nil && node.Context.ApplicationID == 0 {
				return service.WriteJSON(w, []sdk.VCSCommit{}, http.StatusOK)
			}
			if node.Context != nil && node.Context.ApplicationID != 0 {
				app = wf.Applications[node.Context.ApplicationID]
			}
			if node.Context != nil && node.Context.EnvironmentID != 0 {
				env = wf.Environments[node.Context.EnvironmentID]
			}
		}

		wfNodeRun := &sdk.WorkflowNodeRun{}
		if branch != "" {
			wfNodeRun.VCSBranch = branch
		}
		if remote != "" {
			wfNodeRun.VCSRepository = remote
		}
		if hash != "" {
			wfNodeRun.VCSHash = hash
		} else {
			// Find hash and branch of ancestor node run
			var nodeIDsAncestors []int64
			if node != nil {
				nodeIDsAncestors = node.Ancestors(wf.WorkflowData)
			}

			if wfRun != nil && wfRun.WorkflowNodeRuns != nil {
				for _, ancestorID := range nodeIDsAncestors {
					nodeRuns, ok := wfRun.WorkflowNodeRuns[ancestorID]
					if !ok || len(nodeRuns) == 0 {
						continue
					}
					if nodeRuns[0].VCSRepository == app.RepositoryFullname {
						wfNodeRun.VCSHash = nodeRuns[0].VCSHash
						wfNodeRun.VCSBranch = nodeRuns[0].VCSBranch
						break
					}
				}
			}
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		log.Debug(ctx, "getWorkflowCommitsHandler> VCSHash: %s VCSBranch: %s", wfNodeRun.VCSHash, wfNodeRun.VCSBranch)
		commits, _, err := workflow.GetNodeRunBuildCommits(ctx, tx, api.Cache, *proj, *wf, nodeName, number, wfNodeRun, &app, &env)
		if err != nil {
			return sdk.WrapError(err, "unable to load commits")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		return service.WriteJSON(w, commits, http.StatusOK)
	}
}

func (api *API) stopWorkflowNodeRunHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		workflowName := vars["permWorkflowName"]
		workflowRunNumber, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}
		workflowNodeRunID, err := requestVarInt(r, "nodeRunID")
		if err != nil {
			return err
		}

		consumer := getUserConsumer(ctx)

		// This POST exec handler should not be called by workers
		if consumer.AuthConsumerUser.Worker != nil {
			return sdk.WrapError(sdk.ErrForbidden, "not authorized for worker")
		}

		p, err := project.Load(ctx, api.mustDB(), key, project.LoadOptions.WithVariables)
		if err != nil {
			return sdk.WrapError(err, "cannot load project")
		}

		workflowRun, err := workflow.LoadRun(ctx, api.mustDB(), p.Key, workflowName, workflowRunNumber, workflow.LoadRunOptions{
			WithDeleted: true,
		})
		if err != nil {
			return sdk.WrapError(err, "unable to load workflow run with number %d for workflow %s", workflowRunNumber, workflowName)
		}

		workflowNodeRun, err := workflow.LoadNodeRun(api.mustDB(), key, workflowName, workflowNodeRunID, workflow.LoadRunOptions{
			WithDeleted: true,
		})
		if err != nil {
			return sdk.WrapError(err, "unable to load workflow node run with id %d for workflow %s and run with number %d", workflowNodeRunID, workflowName, workflowRun.Number)
		}

		r1, err := workflow.StopWorkflowNodeRun(ctx, api.mustDB, api.Cache, *p, *workflowRun, *workflowNodeRun, sdk.SpawnInfo{
			Message: sdk.SpawnMsg{ID: sdk.MsgWorkflowNodeStop.ID, Args: []interface{}{getUserConsumer(ctx).GetUsername()}},
		})
		if err != nil {
			return sdk.WrapError(err, "unable to stop workflow node run")
		}

		api.GoRoutines.Exec(context.Background(), fmt.Sprintf("stopWorkflowNodeRunHandler-%d", workflowNodeRunID), func(ctx context.Context) {
			api.WorkflowSendEvent(context.Background(), *p, r1)
		})

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		workflowRun, err = workflow.LoadRun(ctx, tx, p.Key, workflowName, workflowRunNumber, workflow.LoadRunOptions{
			WithDeleted: true,
		})
		if err != nil {
			return sdk.WrapError(err, "unable to load workflow run with number %d for workflow %s", workflowRunNumber, workflowName)
		}

		r2, err := workflow.ResyncWorkflowRunStatus(ctx, tx, workflowRun)
		if err != nil {
			return sdk.WrapError(err, "unable to resync workflow run status")
		}

		telemetry.Current(ctx,
			telemetry.Tag(telemetry.TagProjectKey, p.Key),
			telemetry.Tag(telemetry.TagWorkflow, workflowRun.Workflow.Name),
		)
		if workflowRun.Status == sdk.StatusFail {
			telemetry.Record(api.Router.Background, api.Metrics.WorkflowRunFailed, 1)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		api.GoRoutines.Exec(context.Background(), fmt.Sprintf("stopWorkflowNodeRunHandler-%d-resync-run-%d", workflowNodeRunID, workflowRun.ID), func(ctx context.Context) {
			api.WorkflowSendEvent(context.Background(), *p, r2)
		})

		go func(ID int64) {
			wRun, err := workflow.LoadRunByID(ctx, api.mustDB(), ID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
			if err != nil {
				log.Error(ctx, "workflow.stopWorkflowNodeRun> Cannot load run for resync commit status %v", err)
				return
			}
			//The function could be called with nil project so we need to test if project is not nil
			if sdk.StatusIsTerminated(wRun.Status) && p != nil {
				wRun.LastExecution = time.Now()
				if err := workflow.ResyncCommitStatus(context.Background(), api.mustDB(), api.Cache, *p, wRun, api.Config.URL.UI); err != nil {
					log.Error(ctx, "workflow.stopWorkflowNodeRun> %v", err)
				}
			}
		}(workflowRun.ID)

		return service.WriteJSON(w, workflowNodeRun, http.StatusOK)
	}
}

func (api *API) getWorkflowNodeRunHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		id, err := requestVarInt(r, "nodeRunID")
		if err != nil {
			return err
		}
		nodeRun, err := workflow.LoadNodeRun(api.mustDB(), key, name, id, workflow.LoadRunOptions{
			WithTests: true,
		})
		if err != nil {
			return sdk.WrapError(err, "Unable to load last workflow run")
		}

		nodeRun.Translate()
		return service.WriteJSON(w, nodeRun, http.StatusOK)
	}
}

func (api *API) postWorkflowRunHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowNameAdvanced"]

		consumer := getUserConsumer(ctx)

		// This POST exec handler should not be called by workers
		if consumer.AuthConsumerUser.Worker != nil {
			return sdk.WrapError(sdk.ErrForbidden, "not authorized for worker")
		}

		telemetry.Current(ctx,
			telemetry.Tag(telemetry.TagProjectKey, key),
			telemetry.Tag(telemetry.TagWorkflow, name),
		)
		telemetry.Record(api.Router.Background, api.Metrics.WorkflowRunStarted, 1)

		// LOAD PROJECT
		_, next := telemetry.Span(ctx, "project.Load")
		p, err := project.Load(ctx, api.mustDB(), key,
			project.LoadOptions.WithVariables,
			project.LoadOptions.WithIntegrations,
		)
		next()
		if err != nil {
			return sdk.WrapError(err, "cannot load project")
		}

		opts := sdk.WorkflowRunPostHandlerOption{}
		if err := service.UnmarshalBody(r, &opts); err != nil {
			return err
		}
		opts.AuthConsumerID = getUserConsumer(ctx).ID

		// Request check
		if opts.Manual != nil && opts.Manual.OnlyFailedJobs && opts.Manual.Resync {
			return sdk.WrapError(sdk.ErrWrongRequest, "You cannot resync workflow and run only failed jobs")
		}

		// CHECK IF IT S AN EXISTING RUN
		var lastRun *sdk.WorkflowRun
		if opts.Number != nil {
			lastRun, err = workflow.LoadRun(ctx, api.mustDB(), key, name, *opts.Number, workflow.LoadRunOptions{})
			if err != nil {
				return sdk.WrapError(err, "unable to load workflow run")
			}
		}

		// To handle conditions on hooks
		if opts.Hook != nil {
			hook, err := workflow.LoadHookByUUID(api.mustDB(), opts.Hook.WorkflowNodeHookUUID)
			if err != nil {
				return sdk.WrapError(err, "cannot load hook for uuid %s", opts.Hook.WorkflowNodeHookUUID)
			}
			conditions := hook.Conditions
			params := sdk.ParametersFromMap(opts.Hook.Payload)

			var conditionsOK bool
			var conditionsError error
			if conditions.LuaScript == "" {
				conditionsOK, conditionsError = sdk.WorkflowCheckConditions(conditions.PlainConditions, params)
			} else {
				luacheck, err := luascript.NewCheck()
				if err != nil {
					return sdk.WrapError(err, "cannot check lua script")
				}
				luacheck.SetVariables(sdk.ParametersToMap(params))
				conditionsError = luacheck.Perform(conditions.LuaScript)
				conditionsOK = luacheck.Result
			}
			if conditionsError != nil {
				return sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("cannot check conditions: %v", conditionsError))
			}

			if !conditionsOK {
				return sdk.WithStack(sdk.ErrConditionsNotOk)
			}
		}

		var wf *sdk.Workflow
		// IF CONTINUE EXISTING RUN
		if lastRun != nil {
			if lastRun.ReadOnly {
				return sdk.NewErrorFrom(sdk.ErrForbidden, "this workflow execution is on read only mode, it cannot be run anymore")
			}

			if opts.Manual != nil && opts.Manual.Resync {
				log.Debug(ctx, "Resync workflow %d for run %d", lastRun.Workflow.ID, lastRun.ID)
				if err := workflow.Resync(ctx, api.mustDB(), api.Cache, *p, lastRun); err != nil {
					return err
				}
			}

			wf = &lastRun.Workflow
			// Check workflow name in case of rename
			if wf.Name != name {
				wf.Name = name
			}

			for _, id := range opts.FromNodeIDs {
				fromNode := lastRun.Workflow.WorkflowData.NodeByID(opts.FromNodeIDs[0])
				if fromNode == nil {
					return sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "unable to find node %d", id)
				}

				if !permission.AccessToWorkflowNode(ctx, api.mustDB(), &lastRun.Workflow, fromNode, *consumer, sdk.PermissionReadExecute) {
					return sdk.WrapError(sdk.ErrNoPermExecution, "not enough right on node %s", fromNode.Name)
				}
			}

			lastRun.Status = sdk.StatusWaiting
			// Workflow Run initialization
			api.GoRoutines.Exec(context.Background(), fmt.Sprintf("api.initWorkflowRun-%d", lastRun.ID), func(ctx context.Context) {
				api.initWorkflowRun(ctx, p.Key, wf, lastRun, opts)
			})

		} else {
			wf, err = workflow.Load(ctx, api.mustDB(), api.Cache, *p, name, workflow.LoadOptions{
				DeepPipeline:          true,
				WithAsCodeUpdateEvent: true,
				WithIcon:              true,
				WithIntegrations:      true,
				WithTemplate:          true,
			})
			if err != nil {
				return sdk.WrapError(err, "unable to load workflow %s", name)
			}

			// Check node permission
			if isService := isService(ctx); !isService && !permission.AccessToWorkflowNode(ctx, api.mustDB(), wf, &wf.WorkflowData.Node, *consumer, sdk.PermissionReadExecute) {
				return sdk.WrapError(sdk.ErrNoPermExecution, "not enough right on node %s", wf.WorkflowData.Node.Name)
			}

			// CREATE WORKFLOW RUN
			lastRun, err = workflow.CreateRun(api.mustDB(), wf, opts)
			if err != nil {
				return err
			}
		}

		api.setWorkflowRunURLs(lastRun)

		return service.WriteJSON(w, lastRun, http.StatusAccepted)
	}
}

func (api *API) initWorkflowRun(ctx context.Context, projKey string, wf *sdk.Workflow, wfRun *sdk.WorkflowRun, opts sdk.WorkflowRunPostHandlerOption) *workflow.ProcessorReport {
	ctx, end := telemetry.Span(ctx, "api.initWorkflowRun",
		telemetry.Tag(telemetry.TagProjectKey, projKey),
		telemetry.Tag(telemetry.TagWorkflow, wf.Name),
	)
	defer end()

	var asCodeInfosMsg []sdk.Message
	var report = new(workflow.ProcessorReport)

	c, err := authentication.LoadUserConsumerByID(ctx, api.mustDB(), opts.AuthConsumerID,
		authentication.LoadUserConsumerOptions.WithAuthentifiedUserWithContacts,
		authentication.LoadUserConsumerOptions.WithConsumerGroups)
	if err != nil {
		r := failInitWorkflowRun(ctx, api.mustDB(), wfRun, err)
		report.Merge(ctx, r)
		return report
	}

	// Add service for consumer if exists
	s, err := services.LoadByConsumerID(context.Background(), api.mustDB(), c.ID)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		r := failInitWorkflowRun(ctx, api.mustDB(), wfRun, err)
		report.Merge(ctx, r)
		return report
	}
	c.AuthConsumerUser.Service = s

	p, err := project.Load(ctx, api.mustDB(), projKey,
		project.LoadOptions.WithVariables,
		project.LoadOptions.WithKeys,
		project.LoadOptions.WithIntegrations,
		project.LoadOptions.WithGroups,
	)
	if err != nil {
		r := failInitWorkflowRun(ctx, api.mustDB(), wfRun, sdk.WrapError(err, "cannot load project for as code workflow creation"))
		report.Merge(ctx, r)
		return report
	}

	// To avoid crafting more than once workflow as code from the same repo at the same time
	// We lock the repository in redis
	if wf.FromRepository != "" {
		cackeKey := cache.Key("api", "initworkflow", "repository", wf.FromRepository)
		ok, err := api.Cache.Lock(cackeKey, time.Minute, 100, 5)
		if err != nil {
			r := failInitWorkflowRun(ctx, api.mustDB(), wfRun, sdk.WrapError(err, "unable lock repository in cache"))
			report.Merge(ctx, r)
			return report
		}
		if !ok {
			return nil
		}
		defer api.Cache.Unlock(cackeKey) // nolint
	}

	defer func() {
		go api.WorkflowSendEvent(context.Background(), *p, report)
	}()

	var workflowSecrets *workflow.PushSecrets
	if wfRun.Status == sdk.StatusPending {
		// Sync as code event to remove events in case where a PR was merged
		if len(wf.AsCodeEvent) > 0 {
			res, err := ascode.SyncEvents(ctx, api.mustDB(), api.Cache, *p, *wf, c)
			if err != nil {
				r := failInitWorkflowRun(ctx, api.mustDB(), wfRun, sdk.WrapError(err, "unable to sync as code event"))
				report.Merge(ctx, r)
				return report
			}
			if res.Merged {
				if err := workflow.UpdateFromRepository(api.mustDB(), wf.ID, res.FromRepository); err != nil {
					r := failInitWorkflowRun(ctx, api.mustDB(), wfRun, sdk.WrapError(err, "unable to sync as code event"))
					report.Merge(ctx, r)
					return report
				}
				wf.FromRepository = res.FromRepository
				event.PublishWorkflowUpdate(ctx, p.Key, *wf, *wf, c)
			}
		}

		// If the workflow is as code we need to reimport it.
		// NOTICE: Only repository webhooks and manual run will perform the repository analysis.
		workflowStartedByRepoWebHook := opts.Hook != nil && wf.WorkflowData.Node.GetHook(opts.Hook.WorkflowNodeHookUUID) != nil &&
			wf.WorkflowData.Node.GetHook(opts.Hook.WorkflowNodeHookUUID).HookModelName == sdk.RepositoryWebHookModelName

		if wf.FromRepository != "" && (workflowStartedByRepoWebHook || opts.Manual != nil) {
			log.Debug(ctx, "initWorkflowRun> rebuild workflow %s/%s from as code configuration", p.Key, wf.Name)
			p1, err := project.Load(ctx, api.mustDB(), projKey,
				project.LoadOptions.WithVariables,
				project.LoadOptions.WithGroups,
				project.LoadOptions.WithApplicationVariables,
				project.LoadOptions.WithApplicationKeys,
				project.LoadOptions.WithApplicationWithDeploymentStrategies,
				project.LoadOptions.WithEnvironments,
				project.LoadOptions.WithPipelines,
				project.LoadOptions.WithClearKeys,
				project.LoadOptions.WithClearIntegrations,
			)
			if err != nil {
				r := failInitWorkflowRun(ctx, api.mustDB(), wfRun, sdk.WrapError(err, "cannot load project for as code workflow creation"))
				report.Merge(ctx, r)
				return report
			}

			// Get workflow from repository
			log.Debug(ctx, "workflow.CreateFromRepository> %s", wf.Name)
			oldWf := *wf
			var asCodeInfosMsg []sdk.Message
			workflowSecrets, asCodeInfosMsg, err = workflow.CreateFromRepository(ctx, api.mustDB(), api.Cache, p1, wf, opts, *c, project.DecryptWithBuiltinKey, api.gpgKeyEmailAddress)
			infos := make([]sdk.SpawnMsg, len(asCodeInfosMsg))
			for i, msg := range asCodeInfosMsg {
				infos[i] = msg.ToSpawnMsg()
			}
			workflow.AddWorkflowRunInfo(wfRun, infos...)
			if err != nil {
				r1 := failInitWorkflowRun(ctx, api.mustDB(), wfRun, sdk.WrapError(err, "unable to get workflow from repository"))
				report.Merge(ctx, r1)
				return report
			}

			event.PublishWorkflowUpdate(ctx, p.Key, *wf, oldWf, c)
		} else {
			// Get all secrets for non ascode run
			workflowSecrets, err = workflow.RetrieveSecrets(ctx, api.mustDB(), *wf)
			if err != nil {
				r1 := failInitWorkflowRun(ctx, api.mustDB(), wfRun, sdk.WrapError(err, "unable to retrieve workflow secret"))
				report.Merge(ctx, r1)
				return report
			}
		}

		wfRun.Workflow = *wf

		if err := saveWorkflowRunSecrets(ctx, api.mustDB(), p.ID, *wfRun, workflowSecrets); err != nil {
			r := failInitWorkflowRun(ctx, api.mustDB(), wfRun, sdk.WrapError(err, "unable to compute workflow secrets %s/%s", p.Key, wf.Name))
			report.Merge(ctx, r)
			return report
		}
	}

	if exist := featureflipping.Exists(ctx, gorpmapping.Mapper, api.mustDB(), sdk.FeatureRegion); exist {
		if err := workflow.CheckRegion(ctx, api.mustDB(), *p, wfRun.Workflow); err != nil {
			r := failInitWorkflowRun(ctx, api.mustDB(), wfRun, err)
			report.Merge(ctx, r)
			return report
		}
	}

	tx, err := api.mustDB().Begin()
	if err != nil {
		r := failInitWorkflowRun(ctx, api.mustDB(), wfRun, sdk.WrapError(err, "unable to start workflow %s/%s", p.Key, wf.Name))
		report.Merge(ctx, r)
		return report
	}

	r, err := workflow.StartWorkflowRun(ctx, tx, api.Cache, *p, wfRun, &opts, *c, asCodeInfosMsg)
	report.Merge(ctx, r)
	if err != nil {
		_ = tx.Rollback()
		r := failInitWorkflowRun(ctx, api.mustDB(), wfRun, sdk.WrapError(err, "unable to start workflow %s/%s", p.Key, wf.Name))
		report.Merge(ctx, r)
		return report
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		r := failInitWorkflowRun(ctx, api.mustDB(), wfRun, sdk.WrapError(err, "unable to start workflow %s/%s", p.Key, wf.Name))
		report.Merge(ctx, r)
		return report
	}
	workflow.ResyncNodeRunsWithCommits(api.Router.Background, api.mustDBWithCtx(api.Router.Background), api.Cache, *p, report)

	api.initWorkflowRunPurge(ctx, wf)

	// Update parent
	for i := range report.WorkflowRuns() {
		run := &report.WorkflowRuns()[i]
		if err := api.updateParentWorkflowRun(ctx, run); err != nil {
			log.Error(ctx, "unable to update parent workflow run: %v", err)
		}
	}

	return report
}

func (api *API) initWorkflowRunPurge(ctx context.Context, wf *sdk.Workflow) {
	_, enabled := featureflipping.IsEnabled(ctx, gorpmapping.Mapper, api.mustDB(), sdk.FeaturePurgeName, map[string]string{"project_key": wf.ProjectKey})
	if !enabled {
		// Purge workflow run
		api.GoRoutines.Exec(ctx, "workflow.PurgeWorkflowRun", func(ctx context.Context) {
			tx, err := api.mustDB().Begin()
			defer tx.Rollback() // nolint
			if err != nil {
				log.Error(ctx, "workflow.PurgeWorkflowRun> error %v", err)
				return
			}
			if err := workflow.PurgeWorkflowRun(ctx, tx, *wf); err != nil {
				log.Error(ctx, "workflow.PurgeWorkflowRun> error %v", err)
				return
			}
			if err := tx.Commit(); err != nil {
				log.Error(ctx, "workflow.PurgeWorkflowRun> unable to commit transaction:  %v", err)
				return
			}
			workflow.CountWorkflowRunsMarkToDelete(ctx, api.mustDB(), api.Metrics.WorkflowRunsMarkToDelete)
		})
	}
}

func saveWorkflowRunSecrets(ctx context.Context, db *gorp.DbMap, projID int64, wr sdk.WorkflowRun, secrets *workflow.PushSecrets) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	// Get project secrets
	p, err := project.LoadByID(tx, projID, project.LoadOptions.WithVariablesWithClearPassword, project.LoadOptions.WithClearKeys)
	if err != nil {
		return err
	}

	// Create a snapshot of project secrets and keys
	pv := sdk.VariablesFilter(sdk.FromProjectVariables(p.Variables), sdk.SecretVariable, sdk.KeyVariable)
	pv = sdk.VariablesPrefix(pv, "cds.proj.")
	for _, v := range pv {
		wrSecret := sdk.WorkflowRunSecret{
			WorkflowRunID: wr.ID,
			Context:       workflow.SecretProjContext,
			Name:          v.Name,
			Type:          v.Type,
			Value:         []byte(v.Value),
		}
		if err := workflow.InsertRunSecret(ctx, tx, &wrSecret); err != nil {
			return err
		}
	}

	for _, k := range p.Keys {
		log.Debug(ctx, "checking %q (disabled:%v)", k.Name, k.Disabled)
		if k.Disabled {
			continue // skip disabled keys, so they are not usable in workers
		}
		wrSecret := sdk.WorkflowRunSecret{
			WorkflowRunID: wr.ID,
			Context:       workflow.SecretProjContext,
			Name:          fmt.Sprintf("cds.key.%s.priv", k.Name),
			Type:          string(k.Type),
			Value:         []byte(k.Private),
		}
		if err := workflow.InsertRunSecret(ctx, tx, &wrSecret); err != nil {
			return err
		}
	}

	// Find Needed Project Integrations
	ppIDs := make(map[int64]string)
	for _, n := range wr.Workflow.WorkflowData.Array() {
		if n.Context == nil || n.Context.ProjectIntegrationID == 0 {
			continue
		}
		ppIDs[n.Context.ProjectIntegrationID] = ""
	}
	for _, n := range wr.Workflow.Integrations {
		if !sdk.AllowIntegrationInVariable(n.ProjectIntegration.Model) {
			continue
		}
		ppIDs[n.ProjectIntegrationID] = ""
	}

	for ppID := range ppIDs {
		projectIntegration, err := integration.LoadProjectIntegrationByIDWithClearPassword(ctx, tx, ppID)
		if err != nil {
			return err
		}
		ppIDs[ppID] = projectIntegration.Name

		// Project integration secret variable
		for k, v := range projectIntegration.Config {
			if v.Type != sdk.SecretVariable {
				continue
			}
			wrSecret := sdk.WorkflowRunSecret{
				WorkflowRunID: wr.ID,
				Context:       fmt.Sprintf(workflow.SecretProjIntegrationContext, ppID),
				Name:          fmt.Sprintf("cds.integration.%s.%s", sdk.GetIntegrationVariablePrefix(projectIntegration.Model), k),
				Type:          v.Type,
				Value:         []byte(v.Value),
			}
			if err := workflow.InsertRunSecret(ctx, tx, &wrSecret); err != nil {
				return err
			}
		}
	}

	// Application secret
	for id, variables := range secrets.ApplicationsSecrets {
		// Filter to avoid getting cds.deployment variables
		for _, v := range variables {
			var wrSecret sdk.WorkflowRunSecret
			switch {
			case strings.HasPrefix(v.Name, "cds.app.") || strings.HasPrefix(v.Name, "cds.key.") || v.Name == "git.http.password":
				wrSecret = sdk.WorkflowRunSecret{
					WorkflowRunID: wr.ID,
					Context:       fmt.Sprintf(workflow.SecretAppContext, id),
					Name:          v.Name,
					Type:          v.Type,
					Value:         []byte(v.Value),
				}
			case strings.Contains(v.Name, ":cds.integration."):
				piName := strings.SplitN(v.Name, ":", 2)
				wrSecret = sdk.WorkflowRunSecret{
					WorkflowRunID: wr.ID,
					Context:       fmt.Sprintf(workflow.SecretApplicationIntegrationContext, id, piName[0]),
					Name:          piName[1],
					Type:          v.Type,
					Value:         []byte(v.Value),
				}
			default:
				continue
			}
			if err := workflow.InsertRunSecret(ctx, tx, &wrSecret); err != nil {
				return err
			}
		}
	}

	// Environment secret
	for id, variables := range secrets.EnvironmentdSecrets {
		for _, v := range variables {
			wrSecret := sdk.WorkflowRunSecret{
				WorkflowRunID: wr.ID,
				Context:       fmt.Sprintf(workflow.SecretEnvContext, id),
				Name:          v.Name,
				Type:          v.Type,
				Value:         []byte(v.Value),
			}
			if err := workflow.InsertRunSecret(ctx, tx, &wrSecret); err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	return nil
}

func failInitWorkflowRun(ctx context.Context, db *gorp.DbMap, wfRun *sdk.WorkflowRun, err error) *workflow.ProcessorReport {
	report := new(workflow.ProcessorReport)

	var info sdk.SpawnMsg
	if sdk.ErrorIs(err, sdk.ErrConditionsNotOk) {
		info = sdk.SpawnMsgNew(*sdk.MsgWorkflowConditionError)
		if len(wfRun.WorkflowNodeRuns) == 0 {
			wfRun.Status = sdk.StatusNeverBuilt
		}
	} else if sdk.ErrorIs(err, sdk.ErrRegionNotAllowed) {
		httpErr := sdk.ExtractHTTPError(err)
		info = sdk.SpawnMsgNew(*sdk.MsgWorkflowError, httpErr)
		ctx = sdk.ContextWithStacktrace(ctx, err)
		log.Warn(ctx, "%v", err)
	} else {
		httpErr := sdk.ExtractHTTPError(err)
		info = sdk.SpawnMsgNew(*sdk.MsgWorkflowError, httpErr)
		wfRun.Status = sdk.StatusFail
		log.ErrorWithStackTrace(ctx, err)
	}

	workflow.AddWorkflowRunInfo(wfRun, info)
	if errU := workflow.UpdateWorkflowRun(ctx, db, wfRun); errU != nil {
		log.Error(ctx, "unable to fail workflow run %v", errU)
	}
	report.Add(ctx, wfRun)
	return report
}

func (api *API) getWorkflowRunResultsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowNameAdvanced"]

		number, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}

		wr, err := workflow.LoadRun(ctx, api.mustDB(), key, name, number, workflow.LoadRunOptions{
			DisableDetailledNodeRun: true,
		})
		if err != nil {
			return sdk.WrapError(err, "unable to load workflow run for workflow %s and number %d", name, number)
		}

		results, err := workflow.LoadRunResultsByRunIDUnique(ctx, api.mustDB(), wr.ID)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, results, http.StatusOK)
	}
}

func (api *API) getWorkflowNodeRunResultsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		number, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}

		nodeRunID, err := requestVarInt(r, "nodeRunID")
		if err != nil {
			return sdk.NewErrorFrom(err, "invalid node run id")
		}

		wnr, err := workflow.LoadNodeRun(api.mustDB(), key, name, nodeRunID, workflow.LoadRunOptions{
			DisableDetailledNodeRun: true,
		})
		if err != nil {
			return sdk.WrapError(err, "unable to load workflow node run with id %d for workflow %s and run with number %d", nodeRunID, name, number)
		}

		results, err := workflow.LoadRunResultsByNodeRunID(ctx, api.mustDB(), wnr.ID)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, results, http.StatusOK)
	}
}

func (api *API) getWorkflowNodeRunJobSpawnInfosHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		workflowName := vars["permWorkflowName"]

		id, err := requestVarInt(r, "nodeRunID")
		if err != nil {
			return err
		}
		runJobID, err := requestVarInt(r, "runJobID")
		if err != nil {
			return err
		}

		nodeRun, err := workflow.LoadNodeRun(api.mustDB(), projectKey, workflowName, id, workflow.LoadRunOptions{
			DisableDetailledNodeRun: true,
		})
		if err != nil {
			return sdk.WrapError(err, "unable to load last workflow run")
		}

		spawnInfos, err := workflow.LoadNodeRunJobInfo(ctx, api.mustDB(), nodeRun.ID, runJobID)
		if err != nil {
			return sdk.WrapError(err, "cannot load spawn infos for node run job id %d", runJobID)
		}

		for ki, info := range spawnInfos {
			if _, ok := sdk.Messages[info.Message.ID]; ok {
				m := sdk.NewMessage(sdk.Messages[info.Message.ID], info.Message.Args...)
				spawnInfos[ki].UserMessage = m.String()
			}
		}
		return service.WriteJSON(w, spawnInfos, http.StatusOK)
	}
}

func (api *API) getWorkflowRunTagsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		workflowName := vars["permWorkflowName"]

		res, err := workflow.GetTagsAndValue(api.mustDB(), projectKey, workflowName)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, res, http.StatusOK)
	}
}

func (api *API) postResyncVCSWorkflowRunHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		number, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}

		consumer := getUserConsumer(ctx)

		// This POST exec handler should not be called by workers
		if consumer.AuthConsumerUser.Worker != nil {
			return sdk.WrapError(sdk.ErrForbidden, "not authorized for worker")
		}

		proj, err := project.Load(ctx, api.mustDB(), key, project.LoadOptions.WithVariables)
		if err != nil {
			return sdk.WrapError(err, "cannot load project")
		}

		wfr, err := workflow.LoadRun(ctx, api.mustDB(), key, name, number, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if err != nil {
			return sdk.WrapError(err, "cannot load workflow run")
		}

		if err := workflow.ResyncCommitStatus(ctx, api.mustDB(), api.Cache, *proj, wfr, api.Config.URL.UI); err != nil {
			return sdk.WrapError(err, "cannot resync workflow run commit status")
		}

		return nil
	}
}
