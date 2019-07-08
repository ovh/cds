package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const (
	rangeMax     = 50
	defaultLimit = 10
)

func (api *API) searchWorkflowRun(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string, route, key, name string) error {
	// About pagination: [FR] http://blog.octo.com/designer-une-api-rest/#pagination
	var limit, offset int

	offsetS := r.FormValue("offset")
	var errAtoi error
	if offsetS != "" {
		offset, errAtoi = strconv.Atoi(offsetS)
		if errAtoi != nil {
			return sdk.ErrWrongRequest
		}
	}
	limitS := r.FormValue("limit")
	if limitS != "" {
		limit, errAtoi = strconv.Atoi(limitS)
		if errAtoi != nil {
			return sdk.ErrWrongRequest
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
	runs, offset, limit, count, err := workflow.LoadRuns(api.mustDB(), key, name, offset, limit, mapFilters)
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

	for i := range runs {
		runs[i].Translate(r.Header.Get("Accept-Language"))
	}

	// Return empty array instead of nil
	if runs == nil {
		runs = []sdk.WorkflowRun{}
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
		return api.searchWorkflowRun(ctx, w, r, vars, route, key, name)
	}
}

func (api *API) getWorkflowRunsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		route := api.Router.GetRoute("GET", api.getWorkflowRunsHandler, map[string]string{
			"key":          key,
			"workflowName": name,
		})
		return api.searchWorkflowRun(ctx, w, r, vars, route, key, name)
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
			return sdk.WrapError(err, "Cannot load current run num")
		}

		if m.Num < num {
			return sdk.WrapError(sdk.ErrWrongRequest, "postWorkflowRunNumHandler> Cannot num must be > %d, got %d", num, m.Num)
		}

		proj, err := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load projet")
		}

		options := workflow.LoadOptions{}
		wf, errW := workflow.Load(ctx, api.mustDB(), api.Cache, proj, name, deprecatedGetUser(ctx), options)
		if errW != nil {
			return sdk.WrapError(errW, "postWorkflowRunNumHandler > Cannot load workflow")
		}

		var errDb error
		if num == 0 {
			errDb = workflow.InsertRunNum(api.mustDB(), wf, m.Num)
		} else {
			errDb = workflow.UpdateRunNum(api.mustDB(), wf, m.Num)
		}

		if errDb != nil {
			return sdk.WrapError(errDb, "postWorkflowRunNumHandler> ")
		}

		return service.WriteJSON(w, m, http.StatusOK)
	}
}

func (api *API) getLatestWorkflowRunHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		run, err := workflow.LoadLastRun(api.mustDB(), key, name, workflow.LoadRunOptions{WithArtifacts: true})
		if err != nil {
			return sdk.WrapError(err, "Unable to load last workflow run")
		}
		run.Translate(r.Header.Get("Accept-Language"))
		return service.WriteJSON(w, run, http.StatusOK)
	}
}

func (api *API) resyncWorkflowRunHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		number, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}

		proj, err := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load projet")
		}

		run, err := workflow.LoadRun(ctx, api.mustDB(), key, name, number, workflow.LoadRunOptions{})
		if err != nil {
			return sdk.WrapError(err, "Unable to load last workflow run [%s/%d]", name, number)
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "resyncWorkflowRunHandler> Cannot start transaction")
		}

		if err := workflow.Resync(tx, api.Cache, proj, run, deprecatedGetUser(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot resync pipelines")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}
		return service.WriteJSON(w, run, http.StatusOK)
	}
}

func (api *API) getWorkflowRunHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		number, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}

		// loadRun, DisableDetailledNodeRun = false for calls from CDS Service
		// as hook service. It's needed to have the buildParameters.
		run, err := workflow.LoadRun(ctx, api.mustDB(), key, name, number,
			workflow.LoadRunOptions{
				WithDeleted:             false,
				WithArtifacts:           true,
				WithLightTests:          true,
				DisableDetailledNodeRun: getService(ctx) == nil,
				Language:                r.Header.Get("Accept-Language"),
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

		run.Translate(r.Header.Get("Accept-Language"))

		return service.WriteJSON(w, run, http.StatusOK)
	}
}

func (api *API) deleteWorkflowRunHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
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

		run, errL := workflow.LoadRun(ctx, api.mustDB(), key, name, number, workflow.LoadRunOptions{})
		if errL != nil {
			return sdk.WrapError(errL, "stopWorkflowRunHandler> Unable to load last workflow run")
		}

		proj, errP := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "stopWorkflowRunHandler> Unable to load project")
		}

		report, err := stopWorkflowRun(ctx, api.mustDB, api.Cache, proj, run, deprecatedGetUser(ctx), 0)
		if err != nil {
			return sdk.WrapError(err, "Unable to stop workflow")
		}
		workflowRuns := report.WorkflowRuns()

		go workflow.SendEvent(api.mustDB(), proj.Key, report)

		go func(ID int64) {
			wRun, errLw := workflow.LoadRunByID(api.mustDB(), ID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
			if errLw != nil {
				log.Error("workflow.stopWorkflowNodeRun> Cannot load run for resync commit status %v", errLw)
				return
			}
			//The function could be called with nil project so we need to test if project is not nil
			if sdk.StatusIsTerminated(wRun.Status) && proj != nil {
				wRun.LastExecution = time.Now()
				if err := workflow.ResyncCommitStatus(context.Background(), api.mustDB(), api.Cache, proj, wRun); err != nil {
					log.Error("workflow.UpdateNodeJobRunStatus> %v", err)
				}
			}
		}(run.ID)

		if len(workflowRuns) > 0 {
			observability.Current(ctx,
				observability.Tag(observability.TagProjectKey, proj.Key),
				observability.Tag(observability.TagWorkflow, workflowRuns[0].Workflow.Name),
			)

			if workflowRuns[0].Status == sdk.StatusFail.String() {
				observability.Record(api.Router.Background, api.Metrics.WorkflowRunFailed, 1)
			}
		}

		return service.WriteJSON(w, run, http.StatusOK)
	}
}

func stopWorkflowRun(ctx context.Context, dbFunc func() *gorp.DbMap, store cache.Store, p *sdk.Project, run *sdk.WorkflowRun, u *sdk.User, parentWorkflowRunID int64) (*workflow.ProcessorReport, error) {
	report := new(workflow.ProcessorReport)

	tx, errTx := dbFunc().Begin()
	if errTx != nil {
		return nil, sdk.WrapError(errTx, "stopWorkflowRun> Unable to create transaction")
	}
	defer tx.Rollback() //nolint

	spwnMsg := sdk.SpawnMsg{ID: sdk.MsgWorkflowNodeStop.ID, Args: []interface{}{u.Username}}

	stopInfos := sdk.SpawnInfo{
		APITime:    time.Now(),
		RemoteTime: time.Now(),
		Message:    spwnMsg,
	}

	workflow.AddWorkflowRunInfo(run, false, spwnMsg)

	for _, wn := range run.WorkflowNodeRuns {
		for _, wnr := range wn {
			if wnr.SubNumber != run.LastSubNumber || (wnr.Status == sdk.StatusSuccess.String() ||
				wnr.Status == sdk.StatusFail.String() || wnr.Status == sdk.StatusSkipped.String()) {
				log.Debug("stopWorkflowRun> cannot stop this workflow node run with current status %s", wnr.Status)
				continue
			}

			r1, errS := workflow.StopWorkflowNodeRun(ctx, dbFunc, store, p, wnr, stopInfos)
			if errS != nil {
				return nil, sdk.WrapError(errS, "stopWorkflowRun> Unable to stop workflow node run %d", wnr.ID)
			}
			report.Merge(r1, nil) // nolint
			wnr.Status = sdk.StatusStopped.String()

			// If it's a outgoing hook, we stop the child
			if wnr.OutgoingHook != nil {
				if run.Workflow.OutGoingHookModels == nil {
					run.Workflow.OutGoingHookModels = make(map[int64]sdk.WorkflowHookModel)
				}
				model, has := run.Workflow.OutGoingHookModels[wnr.OutgoingHook.HookModelID]
				if !has {
					m, errM := workflow.LoadOutgoingHookModelByID(dbFunc(), wnr.OutgoingHook.HookModelID)
					if errM != nil {
						log.Error("stopWorkflowRun> Unable to load outgoing hook model: %v", errM)
						continue
					}
					model = *m
					run.Workflow.OutGoingHookModels[wnr.OutgoingHook.HookModelID] = *m
				}
				if model.Name == sdk.WorkflowModelName && wnr.Callback != nil && wnr.Callback.WorkflowRunNumber != nil {
					//Stop trigggered workflow
					targetProject := wnr.OutgoingHook.Config[sdk.HookConfigTargetProject].Value
					targetWorkflow := wnr.OutgoingHook.Config[sdk.HookConfigTargetWorkflow].Value

					targetRun, errL := workflow.LoadRun(ctx, dbFunc(), targetProject, targetWorkflow, *wnr.Callback.WorkflowRunNumber, workflow.LoadRunOptions{})
					if errL != nil {
						log.Error("stopWorkflowRun> Unable to load last workflow run: %v", errL)
						continue
					}

					targetProj, errP := project.Load(dbFunc(), store, targetProject, u)
					if errP != nil {
						log.Error("stopWorkflowRun> Unable to load project %v", errP)
						continue
					}

					r2, err := stopWorkflowRun(ctx, dbFunc, store, targetProj, targetRun, u, run.ID)
					if err != nil {
						log.Error("stopWorkflowRun> Unable to stop workflow %v", err)
						continue
					}
					report.Merge(r2, nil) // nolint
				}
			}
		}
	}

	run.LastExecution = time.Now()
	run.Status = sdk.StatusStopped.String()
	if errU := workflow.UpdateWorkflowRun(ctx, tx, run); errU != nil {
		return nil, sdk.WrapError(errU, "Unable to update workflow run %d", run.ID)
	}
	report.Add(*run)

	if err := tx.Commit(); err != nil {
		return nil, sdk.WrapError(err, "Cannot commit transaction")
	}

	if parentWorkflowRunID == 0 {
		if err := updateParentWorkflowRun(ctx, dbFunc, store, run); err != nil {
			return nil, sdk.WithStack(err)
		}
	}

	return report, nil
}

func updateParentWorkflowRun(ctx context.Context, dbFunc func() *gorp.DbMap, store cache.Store, run *sdk.WorkflowRun) error {
	if !run.HasParentWorkflow() {
		return nil
	}

	parentProj, err := project.Load(
		dbFunc(), store, run.RootRun().HookEvent.ParentWorkflow.Key,
		deprecatedGetUser(ctx),
		project.LoadOptions.WithVariables,
		project.LoadOptions.WithFeatures,
		project.LoadOptions.WithIntegrations,
		project.LoadOptions.WithApplicationVariables,
		project.LoadOptions.WithApplicationWithDeploymentStrategies,
	)
	if err != nil {
		return sdk.WrapError(err, "updateParentWorkflowRun> Cannot load project")
	}

	parentWR, err := workflow.LoadRun(ctx,
		dbFunc(),
		run.RootRun().HookEvent.ParentWorkflow.Key,
		run.RootRun().HookEvent.ParentWorkflow.Name,
		run.RootRun().HookEvent.ParentWorkflow.Run,
		workflow.LoadRunOptions{
			DisableDetailledNodeRun: false,
		})
	if err != nil {
		return sdk.WrapError(err, "Unable to load parent run: %v", run.RootRun().HookEvent)
	}

	if err := workflow.UpdateParentWorkflowRun(ctx, dbFunc, store, run, parentProj, parentWR); err != nil {
		return sdk.WrapError(err, "updateParentWorkflowRun")
	}

	return nil
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

// TODO Clean old workflow structure
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

		proj, errP := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.WithIntegrations)
		if errP != nil {
			return sdk.WrapError(errP, "getWorkflowCommitsHandler> Unable to load project %s", key)
		}

		var wf *sdk.Workflow
		wfRun, errW := workflow.LoadRun(ctx, api.mustDB(), key, name, number, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if errW != nil {
			wf, errW = workflow.Load(ctx, api.mustDB(), api.Cache, proj, name, deprecatedGetUser(ctx), workflow.LoadOptions{})
			if errW != nil {
				return sdk.WrapError(errW, "getWorkflowCommitsHandler> Unable to load workflow %s", name)
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
				return sdk.WrapError(sdk.ErrNotFound, "getWorkflowCommitsHandler> Unable to load workflow data node")
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

		if wfRun == nil {
			wfRun = &sdk.WorkflowRun{Number: number, Workflow: *wf}
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
				nodeIDsAncestors = node.Ancestors(wfRun.Workflow.WorkflowData)
			}

			for _, ancestorID := range nodeIDsAncestors {
				if wfRun.WorkflowNodeRuns != nil && wfRun.WorkflowNodeRuns[ancestorID][0].VCSRepository == app.RepositoryFullname {
					wfNodeRun.VCSHash = wfRun.WorkflowNodeRuns[ancestorID][0].VCSHash
					wfNodeRun.VCSBranch = wfRun.WorkflowNodeRuns[ancestorID][0].VCSBranch
					break
				}
			}
		}

		log.Debug("getWorkflowCommitsHandler> VCSHash: %s VCSBranch: %s", wfNodeRun.VCSHash, wfNodeRun.VCSBranch)
		commits, _, errC := workflow.GetNodeRunBuildCommits(ctx, api.mustDB(), api.Cache, proj, wf, nodeName, wfRun.Number, wfNodeRun, &app, &env)
		if errC != nil {
			return sdk.WrapError(errC, "getWorkflowCommitsHandler> Unable to load commits: %v", errC)
		}

		return service.WriteJSON(w, commits, http.StatusOK)
	}
}

func (api *API) stopWorkflowNodeRunHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		number, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}
		id, err := requestVarInt(r, "nodeRunID")
		if err != nil {
			return err
		}

		p, errP := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.WithVariables)
		if errP != nil {
			return sdk.WrapError(errP, "stopWorkflowNodeRunHandler> Cannot load project")
		}

		// Load node run
		nodeRun, err := workflow.LoadNodeRun(api.mustDB(), key, name, number, id, workflow.LoadRunOptions{})
		if err != nil {
			return sdk.WrapError(err, "Unable to load last workflow run")
		}

		report, err := api.stopWorkflowNodeRun(ctx, api.mustDB, api.Cache, p, nodeRun, name, deprecatedGetUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "Unable to stop workflow run")
		}

		go workflow.SendEvent(api.mustDB(), p.Key, report)

		return service.WriteJSON(w, nodeRun, http.StatusOK)
	}
}

func (api *API) stopWorkflowNodeRun(ctx context.Context, dbFunc func() *gorp.DbMap, store cache.Store, p *sdk.Project, nodeRun *sdk.WorkflowNodeRun, workflowName string, u *sdk.User) (*workflow.ProcessorReport, error) {
	tx, errTx := dbFunc().Begin()
	if errTx != nil {
		return nil, sdk.WrapError(errTx, "stopWorkflowNodeRunHandler> Unable to create transaction")
	}
	defer tx.Rollback()

	stopInfos := sdk.SpawnInfo{
		APITime:    time.Now(),
		RemoteTime: time.Now(),
		Message:    sdk.SpawnMsg{ID: sdk.MsgWorkflowNodeStop.ID, Args: []interface{}{u.Username}},
	}
	report, errS := workflow.StopWorkflowNodeRun(ctx, dbFunc, store, p, *nodeRun, stopInfos)
	if errS != nil {
		return nil, sdk.WrapError(errS, "stopWorkflowNodeRunHandler> Unable to stop workflow node run")
	}

	wr, errLw := workflow.LoadRun(ctx, tx, p.Key, workflowName, nodeRun.Number, workflow.LoadRunOptions{})
	if errLw != nil {
		return nil, sdk.WrapError(errLw, "stopWorkflowNodeRunHandler> Unable to load workflow run %s", workflowName)
	}

	r1, errR := workflow.ResyncWorkflowRunStatus(tx, wr)
	if errR != nil {
		return nil, sdk.WrapError(errR, "stopWorkflowNodeRunHandler> Unable to resync workflow run status")
	}

	_, _ = report.Merge(r1, nil)

	observability.Current(ctx,
		observability.Tag(observability.TagProjectKey, p.Key),
		observability.Tag(observability.TagWorkflow, wr.Workflow.Name),
	)
	if wr.Status == sdk.StatusFail.String() {
		observability.Record(api.Router.Background, api.Metrics.WorkflowRunFailed, 1)
	}

	if errC := tx.Commit(); errC != nil {
		return nil, sdk.WrapError(errC, "stopWorkflowNodeRunHandler> Unable to commit")
	}

	go func(ID int64) {
		wRun, errLw := workflow.LoadRunByID(api.mustDB(), ID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if errLw != nil {
			log.Error("workflow.stopWorkflowNodeRun> Cannot load run for resync commit status %v", errLw)
			return
		}
		//The function could be called with nil project so we need to test if project is not nil
		if sdk.StatusIsTerminated(wRun.Status) && p != nil {
			wRun.LastExecution = time.Now()
			if err := workflow.ResyncCommitStatus(context.Background(), api.mustDB(), api.Cache, p, wRun); err != nil {
				log.Error("workflow.stopWorkflowNodeRun> %v", err)
			}
		}
	}(wr.ID)

	return report, nil
}

func (api *API) getWorkflowNodeRunHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		number, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}
		id, err := requestVarInt(r, "nodeRunID")
		if err != nil {
			return err
		}
		run, err := workflow.LoadNodeRun(api.mustDB(), key, name, number, id, workflow.LoadRunOptions{
			WithTests:           true,
			WithArtifacts:       true,
			WithStaticFiles:     true,
			WithCoverage:        true,
			WithVulnerabilities: true,
		})
		if err != nil {
			return sdk.WrapError(err, "Unable to load last workflow run")
		}

		run.Translate(r.Header.Get("Accept-Language"))
		return service.WriteJSON(w, run, http.StatusOK)
	}
}

func (api *API) postWorkflowRunHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		u := deprecatedGetUser(ctx)

		observability.Current(ctx,
			observability.Tag(observability.TagProjectKey, key),
			observability.Tag(observability.TagWorkflow, name),
		)
		observability.Record(api.Router.Background, api.Metrics.WorkflowRunStarted, 1)

		// LOAD PROJECT
		_, next := observability.Span(ctx, "project.Load")
		p, errP := project.Load(api.mustDB(), api.Cache, key, u,
			project.LoadOptions.WithVariables,
			project.LoadOptions.WithFeatures,
			project.LoadOptions.WithIntegrations,
			project.LoadOptions.WithApplicationVariables,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
		)
		next()
		if errP != nil {
			return sdk.WrapError(errP, "postWorkflowRunHandler> Cannot load project")
		}

		// GET BODY
		opts := &sdk.WorkflowRunPostHandlerOption{}
		if err := service.UnmarshalBody(r, opts); err != nil {
			return err
		}

		// CHECK IF IT S AN EXISTING RUN
		var lastRun *sdk.WorkflowRun
		if opts.Number != nil {
			var errlr error
			lastRun, errlr = workflow.LoadRun(ctx, api.mustDB(), key, name, *opts.Number, workflow.LoadRunOptions{})
			if errlr != nil {
				return sdk.WrapError(errlr, "postWorkflowRunHandler> Unable to load workflow run")
			}
		}

		var wf *sdk.Workflow
		// IF CONTINUE EXISTING RUN
		if lastRun != nil {
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

				if !permission.AccessToWorkflowNode(&lastRun.Workflow, fromNode, u, permission.PermissionReadExecute) {
					return sdk.WrapError(sdk.ErrNoPermExecution, "not enough right on node %s", fromNode.Name)
				}
			}

			lastRun.Status = sdk.StatusWaiting.String()
		} else {
			var errWf error
			wf, errWf = workflow.Load(ctx, api.mustDB(), api.Cache, p, name, u, workflow.LoadOptions{
				DeepPipeline:          true,
				Base64Keys:            true,
				WithAsCodeUpdateEvent: true,
				WithIcon:              true,
			})
			if errWf != nil {
				return sdk.WrapError(errWf, "unable to load workflow %s", name)
			}

			// Check node permission
			if getService(ctx) == nil && !permission.AccessToWorkflowNode(wf, &wf.WorkflowData.Node, u, permission.PermissionReadExecute) {
				return sdk.WrapError(sdk.ErrNoPermExecution, "not enough right on node %s", wf.WorkflowData.Node.Name)
			}

			// CREATE WORKFLOW RUN
			var errCreateRun error
			lastRun, errCreateRun = workflow.CreateRun(api.mustDB(), wf, opts, u)
			if errCreateRun != nil {
				return errCreateRun
			}
		}

		// Workflow Run initialization
		sdk.GoRoutine(context.Background(), fmt.Sprintf("api.initWorkflowRun-%d", lastRun.ID), func(ctx context.Context) {
			api.initWorkflowRun(ctx, api.mustDB(), api.Cache, p, wf, lastRun, opts, u)
		}, api.PanicDump())

		return service.WriteJSON(w, lastRun, http.StatusAccepted)
	}
}

func (api *API) initWorkflowRun(ctx context.Context, db *gorp.DbMap, cache cache.Store, p *sdk.Project, wf *sdk.Workflow, wfRun *sdk.WorkflowRun, opts *sdk.WorkflowRunPostHandlerOption, u *sdk.User) {
	var asCodeInfosMsg []sdk.Message
	report := new(workflow.ProcessorReport)
	defer func() {
		go workflow.SendEvent(db, p.Key, report)
	}()

	// IF NEW WORKFLOW RUN
	if wfRun.Status == sdk.StatusPending.String() {
		// BECOME AS CODE ?
		if wf.FromRepository == "" && len(wf.AsCodeEvent) > 0 {
			tx, err := db.Begin()
			if err != nil {
				r1 := failInitWorkflowRun(ctx, db, wfRun, sdk.WrapError(err, "unable to start transaction"))
				report.Merge(r1, nil) // nolint
				return
			}
			if err := workflow.SyncAsCodeEvent(ctx, tx, cache, p, wf, u); err != nil {
				tx.Rollback() // nolint
				r1 := failInitWorkflowRun(ctx, db, wfRun, sdk.WrapError(err, "unable to sync as code event"))
				report.Merge(r1, nil) // nolint
				return
			}
			if err := tx.Commit(); err != nil {
				tx.Rollback() // nolint
				r1 := failInitWorkflowRun(ctx, db, wfRun, sdk.WrapError(err, "unable to commit transaction as code event"))
				report.Merge(r1, nil) // nolint
				return
			}
		}

		// IF AS CODE - REBUILD Workflow
		if wf.FromRepository != "" {
			p1, errp := project.Load(db, cache, p.Key, u,
				project.LoadOptions.WithVariables,
				project.LoadOptions.WithGroups,
				project.LoadOptions.WithApplicationVariables,
				project.LoadOptions.WithApplicationWithDeploymentStrategies,
				project.LoadOptions.WithEnvironments,
				project.LoadOptions.WithPipelines,
				project.LoadOptions.WithClearKeys,
				project.LoadOptions.WithClearIntegrations,
			)

			if errp != nil {
				r1 := failInitWorkflowRun(ctx, db, wfRun, sdk.WrapError(errp, "cannot load project for as code workflow creation"))
				report.Merge(r1, nil) // nolint
				return
			}
			// Get workflow from repository
			var errCreate error
			log.Debug("workflow.CreateFromRepository> %s", wf.Name)
			asCodeInfosMsg, errCreate = workflow.CreateFromRepository(ctx, db, cache, p1, wf, *opts, u, project.DecryptWithBuiltinKey)
			if errCreate != nil {
				infos := make([]sdk.SpawnMsg, len(asCodeInfosMsg))
				for i, msg := range asCodeInfosMsg {
					infos[i] = sdk.SpawnMsg{
						ID:   msg.ID,
						Args: msg.Args,
					}
					workflow.AddWorkflowRunInfo(wfRun, false, infos...)
				}
				r1 := failInitWorkflowRun(ctx, db, wfRun, sdk.WrapError(errCreate, "unable to get workflow from repository."))
				report.Merge(r1, nil) // nolint
				return
			}
		}
		wfRun.Workflow = *wf
	}

	r1, errS := workflow.StartWorkflowRun(ctx, db, cache, p, wfRun, opts, u, asCodeInfosMsg)
	report.Merge(r1, nil) // nolint
	if errS != nil {
		r1 := failInitWorkflowRun(ctx, db, wfRun, sdk.WrapError(errS, "unable to start workflow %s/%s", p.Key, wf.Name))
		report.Merge(r1, nil) // nolint
		return
	}

	workflow.ResyncNodeRunsWithCommits(db, cache, p, report)

	// Purge workflow run
	sdk.GoRoutine(ctx, "workflow.PurgeWorkflowRun", func(ctx context.Context) {
		if err := workflow.PurgeWorkflowRun(ctx, db, *wf, api.Metrics.WorkflowRunsMarkToDelete); err != nil {
			log.Error("workflow.PurgeWorkflowRun> error %v", err)
		}
	}, api.PanicDump())
}

func failInitWorkflowRun(ctx context.Context, db *gorp.DbMap, wfRun *sdk.WorkflowRun, err error) *workflow.ProcessorReport {
	report := new(workflow.ProcessorReport)

	var info sdk.SpawnMsg
	if sdk.ErrorIs(err, sdk.ErrConditionsNotOk) {
		info = sdk.SpawnMsg{
			ID: sdk.MsgWorkflowConditionError.ID,
		}
		if len(wfRun.WorkflowNodeRuns) == 0 {
			wfRun.Status = sdk.StatusNeverBuilt.String()
		}
	} else {
		log.Error("unable to start workflow: %v", err)
		wfRun.Status = sdk.StatusFail.String()
		info = sdk.SpawnMsg{
			ID:   sdk.MsgWorkflowError.ID,
			Args: []interface{}{sdk.Cause(err).Error()},
		}
	}

	workflow.AddWorkflowRunInfo(wfRun, !sdk.ErrorIs(err, sdk.ErrConditionsNotOk), info)
	if errU := workflow.UpdateWorkflowRun(ctx, db, wfRun); errU != nil {
		log.Error("unable to fail workflow run %v", errU)
	}
	report.Add(wfRun)
	return report
}

func (api *API) downloadworkflowArtifactDirectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		hash := vars["hash"]

		art, err := workflow.LoadWorkfowArtifactByHash(api.mustDB(), hash)
		if err != nil {
			return sdk.WrapError(err, "Could not load artifact with hash %s", hash)
		}

		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", art.Name))

		f, err := api.SharedStorage.Fetch(art)
		if err != nil {
			return sdk.WrapError(err, "Cannot fetch artifact")
		}

		if _, err := io.Copy(w, f); err != nil {
			_ = f.Close()
			return sdk.WrapError(err, "Cannot stream artifact")
		}

		if err := f.Close(); err != nil {
			return sdk.WrapError(err, "Cannot close artifact")
		}
		return nil
	}
}

func (api *API) getDownloadArtifactHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		id, errI := requestVarInt(r, "artifactId")
		if errI != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "getDownloadArtifactHandler> Invalid node job run ID")
		}

		proj, err := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load projet")
		}

		options := workflow.LoadOptions{}
		work, errW := workflow.Load(ctx, api.mustDB(), api.Cache, proj, name, deprecatedGetUser(ctx), options)
		if errW != nil {
			return sdk.WrapError(errW, "getDownloadArtifactHandler> Cannot load workflow")
		}

		art, errA := workflow.LoadArtifactByIDs(api.mustDB(), work.ID, id)
		if errA != nil {
			return sdk.WrapError(errA, "getDownloadArtifactHandler> Cannot load artifacts")
		}

		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", art.Name))

		var integrationName string
		if art.ProjectIntegrationID != nil && *art.ProjectIntegrationID > 0 {
			projectIntegration, err := integration.LoadProjectIntegrationByID(api.mustDB(), *art.ProjectIntegrationID, false)
			if err != nil {
				return sdk.WrapError(err, "Cannot load LoadProjectIntegrationByID %s/%d", proj.Key, *art.ProjectIntegrationID)
			}
			integrationName = projectIntegration.Name
		} else {
			integrationName = sdk.DefaultStorageIntegrationName
		}

		storageDriver, err := api.getStorageDriver(proj.Key, integrationName)
		if err != nil {
			return err
		}

		f, err := storageDriver.Fetch(art)
		if err != nil {
			_ = f.Close()
			return sdk.WrapError(err, "Cannot fetch artifact")
		}

		if _, err := io.Copy(w, f); err != nil {
			_ = f.Close()
			return sdk.WrapError(err, "Cannot stream artifact")
		}

		if err := f.Close(); err != nil {
			return sdk.WrapError(err, "Cannot close artifact")
		}
		return nil
	}
}

func (api *API) getWorkflowRunArtifactsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		number, errNu := requestVarInt(r, "number")
		if errNu != nil {
			return sdk.WrapError(errNu, "getWorkflowJobArtifactsHandler> Invalid node job run ID")
		}

		wr, errW := workflow.LoadRun(ctx, api.mustDB(), key, name, number, workflow.LoadRunOptions{WithArtifacts: true})
		if errW != nil {
			return errW
		}

		arts := []sdk.WorkflowNodeRunArtifact{}
		for _, runs := range wr.WorkflowNodeRuns {
			if len(runs) == 0 {
				continue
			}

			sort.Slice(runs, func(i, j int) bool {
				return runs[i].SubNumber > runs[j].SubNumber
			})

			wg := &sync.WaitGroup{}
			for i := range runs[0].Artifacts {
				wg.Add(1)
				go func(art *sdk.WorkflowNodeRunArtifact) {
					defer wg.Done()

					var integrationName string
					if art.ProjectIntegrationID != nil && *art.ProjectIntegrationID > 0 {
						projectIntegration, err := integration.LoadProjectIntegrationByID(api.mustDB(), *art.ProjectIntegrationID, false)
						if err != nil {
							log.Error("Cannot load LoadProjectIntegrationByID %s/%d: err: %v", key, *art.ProjectIntegrationID, err)
							return
						}
						integrationName = projectIntegration.Name
					} else {
						integrationName = sdk.DefaultStorageIntegrationName
					}

					storageDriver, err := api.getStorageDriver(key, integrationName)
					if err != nil {
						log.Error("Cannot load storage driver: %v", err)
						return
					}

					s, temporaryURLSupported := storageDriver.(objectstore.DriverWithRedirect)
					if temporaryURLSupported { // with temp URL
						fURL, _, err := s.FetchURL(art)
						if err != nil {
							log.Error("Cannot fetch cache object: %v", err)
						} else if fURL != "" {
							art.TempURL = fURL
						}
					}
				}(&runs[0].Artifacts[i])
			}
			wg.Wait()
			arts = append(arts, runs[0].Artifacts...)
		}

		return service.WriteJSON(w, arts, http.StatusOK)
	}
}

func (api *API) getWorkflowNodeRunJobSpawnInfosHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		runJobID, errJ := requestVarInt(r, "runJobId")
		if errJ != nil {
			return sdk.WrapError(errJ, "getWorkflowNodeRunJobSpawnInfosHandler> runJobId: invalid number")
		}
		db := api.mustDB()

		spawnInfos, err := workflow.LoadNodeRunJobInfo(db, runJobID)
		if err != nil {
			return sdk.WrapError(err, "cannot load spawn infos for node run job id %d", runJobID)
		}

		l := r.Header.Get("Accept-Language")
		for ki, info := range spawnInfos {
			m := sdk.NewMessage(sdk.Messages[info.Message.ID], info.Message.Args...)
			spawnInfos[ki].UserMessage = m.String(l)
		}
		return service.WriteJSON(w, spawnInfos, http.StatusOK)
	}
}

func (api *API) getWorkflowNodeRunJobServiceLogsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		runJobID, errJ := requestVarInt(r, "runJobId")
		if errJ != nil {
			return sdk.WrapError(errJ, "runJobId: invalid number")
		}
		db := api.mustDB()

		logsServices, err := workflow.LoadServicesLogsByJob(db, runJobID)
		if err != nil {
			return sdk.WrapError(err, "cannot load service logs for node run job id %d", runJobID)
		}

		return service.WriteJSON(w, logsServices, http.StatusOK)
	}
}

func (api *API) getWorkflowNodeRunJobStepHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		workflowName := vars["permWorkflowName"]
		number, errN := requestVarInt(r, "number")
		if errN != nil {
			return sdk.WrapError(errN, "number: invalid number")
		}
		nodeRunID, errNI := requestVarInt(r, "nodeRunID")
		if errNI != nil {
			return sdk.WrapError(errNI, "id: invalid number")
		}
		runJobID, errJ := requestVarInt(r, "runJobId")
		if errJ != nil {
			return sdk.WrapError(errJ, "runJobId: invalid number")
		}
		stepOrder, errS := requestVarInt(r, "stepOrder")
		if errS != nil {
			return sdk.WrapError(errS, "stepOrder: invalid number")
		}

		// Check nodeRunID is link to workflow
		nodeRun, errNR := workflow.LoadNodeRun(api.mustDB(), projectKey, workflowName, number, nodeRunID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if errNR != nil {
			return sdk.WrapError(errNR, "cannot find nodeRun %d/%d for workflow %s in project %s", nodeRunID, number, workflowName, projectKey)
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
			return sdk.WrapError(sdk.ErrStepNotFound, "cannot find step %d on job %d in nodeRun %d/%d for workflow %s in project %s",
				stepOrder, runJobID, nodeRunID, number, workflowName, projectKey)
		}

		logs, errL := workflow.LoadStepLogs(api.mustDB(), runJobID, stepOrder)
		if errL != nil {
			return sdk.WrapError(errL, "cannot load log for runJob %d on step %d", runJobID, stepOrder)
		}

		ls := &sdk.Log{}
		if logs != nil {
			ls = logs
		}
		result := &sdk.BuildState{
			Status:   sdk.StatusFromString(stepStatus),
			StepLogs: *ls,
		}

		return service.WriteJSON(w, result, http.StatusOK)
	}
}

func (api *API) getWorkflowRunTagsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		workflowName := vars["permWorkflowName"]

		res, err := workflow.GetTagsAndValue(api.mustDB(), projectKey, workflowName)
		if err != nil {
			return sdk.WrapError(err, "Error")
		}

		return service.WriteJSON(w, res, http.StatusOK)
	}
}

func (api *API) postResyncVCSWorkflowRunHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		db := api.mustDB()
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		number, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}

		proj, errP := project.Load(db, api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.WithVariables)
		if errP != nil {
			return sdk.WrapError(errP, "postResyncVCSWorkflowRunHandler> Cannot load project")
		}

		wfr, errW := workflow.LoadRun(ctx, db, key, name, number, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if errW != nil {
			return sdk.WrapError(errW, "postResyncVCSWorkflowRunHandler> Cannot load workflow run")
		}

		if err := workflow.ResyncCommitStatus(ctx, db, api.Cache, proj, wfr); err != nil {
			return sdk.WrapError(err, "Cannot resync workflow run commit status")
		}

		return nil
	}
}
