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
	"github.com/sirupsen/logrus"

	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/luascript"
)

const (
	defaultLimit = 10
)

func (api *API) searchWorkflowRun(w http.ResponseWriter, r *http.Request, route, key, name string) error {
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
		return api.searchWorkflowRun(w, r, route, key, name)
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
		return api.searchWorkflowRun(w, r, route, key, name)
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
			return sdk.WrapError(err, "cannot load current run num")
		}

		if m.Num < num {
			return sdk.WrapError(sdk.ErrWrongRequest, "cannot num must be > %d, got %d", num, m.Num)
		}

		proj, err := project.Load(api.mustDB(), key, project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load projet")
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
		run, err := workflow.LoadLastRun(api.mustDB(), key, name, workflow.LoadRunOptions{WithArtifacts: true})
		if err != nil {
			return sdk.WrapError(err, "Unable to load last workflow run")
		}
		run.Translate(r.Header.Get("Accept-Language"))
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

		withDetailledNodeRun := QueryString(r, "withDetails")

		isService := isService(ctx)

		// loadRun, DisableDetailledNodeRun = false for calls from CDS Service
		// as hook service. It's needed to have the buildParameters.
		run, err := workflow.LoadRun(ctx, api.mustDB(), key, name, number,
			workflow.LoadRunOptions{
				WithDeleted:             false,
				WithArtifacts:           true,
				WithLightTests:          true,
				DisableDetailledNodeRun: !isService && withDetailledNodeRun != "true",
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

		proj, errP := project.Load(api.mustDB(), key)
		if errP != nil {
			return sdk.WrapError(errP, "stopWorkflowRunHandler> Unable to load project")
		}

		report, err := stopWorkflowRun(ctx, api.mustDB, api.Cache, proj, run, getAPIConsumer(ctx), 0)
		if err != nil {
			return sdk.WrapError(err, "Unable to stop workflow")
		}
		workflowRuns := report.WorkflowRuns()

		go WorkflowSendEvent(context.Background(), api.mustDB(), api.Cache, *proj, report)

		go func(ID int64) {
			wRun, errLw := workflow.LoadRunByID(api.mustDB(), ID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
			if errLw != nil {
				log.Error(ctx, "workflow.stopWorkflowNodeRun> Cannot load run for resync commit status %v", errLw)
				return
			}
			//The function could be called with nil project so we need to test if project is not nil
			if sdk.StatusIsTerminated(wRun.Status) && proj != nil {
				wRun.LastExecution = time.Now()
				if err := workflow.ResyncCommitStatus(context.Background(), api.mustDB(), api.Cache, *proj, wRun); err != nil {
					log.Error(ctx, "workflow.UpdateNodeJobRunStatus> %v", err)
				}
			}
		}(run.ID)

		if len(workflowRuns) > 0 {
			observability.Current(ctx,
				observability.Tag(observability.TagProjectKey, proj.Key),
				observability.Tag(observability.TagWorkflow, workflowRuns[0].Workflow.Name),
			)

			if workflowRuns[0].Status == sdk.StatusFail {
				observability.Record(api.Router.Background, api.Metrics.WorkflowRunFailed, 1)
			}
		}

		return service.WriteJSON(w, run, http.StatusOK)
	}
}

func stopWorkflowRun(ctx context.Context, dbFunc func() *gorp.DbMap, store cache.Store, p *sdk.Project,
	run *sdk.WorkflowRun, ident sdk.Identifiable, parentWorkflowRunID int64) (*workflow.ProcessorReport, error) {
	report := new(workflow.ProcessorReport)

	tx, errTx := dbFunc().Begin()
	if errTx != nil {
		return nil, sdk.WrapError(errTx, "unable to create transaction")
	}
	defer tx.Rollback() //nolint

	spwnMsg := sdk.SpawnMsg{ID: sdk.MsgWorkflowNodeStop.ID, Args: []interface{}{ident.GetUsername()}, Type: sdk.MsgWorkflowNodeStop.Type}

	stopInfos := sdk.SpawnInfo{
		APITime:    time.Now(),
		RemoteTime: time.Now(),
		Message:    spwnMsg,
	}

	workflow.AddWorkflowRunInfo(run, spwnMsg)

	for _, wn := range run.WorkflowNodeRuns {
		for _, wnr := range wn {
			if wnr.SubNumber != run.LastSubNumber || (wnr.Status == sdk.StatusSuccess ||
				wnr.Status == sdk.StatusFail || wnr.Status == sdk.StatusSkipped) {
				log.Debug("stopWorkflowRun> cannot stop this workflow node run with current status %s", wnr.Status)
				continue
			}

			r1, err := workflow.StopWorkflowNodeRun(ctx, dbFunc, store, *p, wnr, stopInfos)
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
					m, errM := workflow.LoadOutgoingHookModelByID(dbFunc(), wnr.OutgoingHook.HookModelID)
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

					targetRun, errL := workflow.LoadRun(ctx, dbFunc(), targetProject, targetWorkflow, *wnr.Callback.WorkflowRunNumber, workflow.LoadRunOptions{})
					if errL != nil {
						log.Error(ctx, "stopWorkflowRun> Unable to load last workflow run: %v", errL)
						continue
					}

					targetProj, errP := project.Load(dbFunc(), targetProject)
					if errP != nil {
						log.Error(ctx, "stopWorkflowRun> Unable to load project %v", errP)
						continue
					}

					r2, err := stopWorkflowRun(ctx, dbFunc, store, targetProj, targetRun, ident, run.ID)
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
		report, err := updateParentWorkflowRun(ctx, dbFunc, store, run)
		if err != nil {
			return nil, sdk.WithStack(err)
		}
		go WorkflowSendEvent(context.Background(), dbFunc(), store, *p, report)
	}

	return report, nil
}

func updateParentWorkflowRun(ctx context.Context, dbFunc func() *gorp.DbMap, store cache.Store, run *sdk.WorkflowRun) (*workflow.ProcessorReport, error) {
	if !run.HasParentWorkflow() {
		return nil, nil
	}

	parentProj, err := project.Load(
		dbFunc(), run.RootRun().HookEvent.ParentWorkflow.Key,
		project.LoadOptions.WithVariables,
		project.LoadOptions.WithFeatures(store),
		project.LoadOptions.WithIntegrations,
		project.LoadOptions.WithApplicationVariables,
		project.LoadOptions.WithApplicationWithDeploymentStrategies,
	)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot load project")
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
		return nil, sdk.WrapError(err, "unable to load parent run: %v", run.RootRun().HookEvent)
	}

	report, err := workflow.UpdateParentWorkflowRun(ctx, dbFunc, store, run, *parentProj, parentWR)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	go WorkflowSendEvent(context.Background(), dbFunc(), store, *parentProj, report)

	return report, nil
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

		proj, errP := project.Load(api.mustDB(), key, project.LoadOptions.WithIntegrations)
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

		log.Debug("getWorkflowCommitsHandler> VCSHash: %s VCSBranch: %s", wfNodeRun.VCSHash, wfNodeRun.VCSBranch)
		commits, _, err := workflow.GetNodeRunBuildCommits(ctx, api.mustDB(), api.Cache, *proj, wf, nodeName, number, wfNodeRun, &app, &env)
		if err != nil {
			return sdk.WrapError(err, "unable to load commits")
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

		p, errP := project.Load(api.mustDB(), key, project.LoadOptions.WithVariables)
		if errP != nil {
			return sdk.WrapError(errP, "stopWorkflowNodeRunHandler> Cannot load project")
		}

		// Load node run
		nodeRun, err := workflow.LoadNodeRun(api.mustDB(), key, name, number, id, workflow.LoadRunOptions{})
		if err != nil {
			return sdk.WrapError(err, "Unable to load last workflow run")
		}

		report, err := api.stopWorkflowNodeRun(ctx, api.mustDB, api.Cache, p, nodeRun, name, getAPIConsumer(ctx))
		if err != nil {
			return sdk.WrapError(err, "Unable to stop workflow run")
		}

		go WorkflowSendEvent(context.Background(), api.mustDB(), api.Cache, *p, report)

		return service.WriteJSON(w, nodeRun, http.StatusOK)
	}
}

func (api *API) stopWorkflowNodeRun(ctx context.Context, dbFunc func() *gorp.DbMap, store cache.Store,
	p *sdk.Project, nodeRun *sdk.WorkflowNodeRun, workflowName string, ident sdk.Identifiable) (*workflow.ProcessorReport, error) {
	tx, errTx := dbFunc().Begin()
	if errTx != nil {
		return nil, sdk.WrapError(errTx, "unable to create transaction")
	}
	defer tx.Rollback() // nolint

	stopInfos := sdk.SpawnInfo{
		APITime:    time.Now(),
		RemoteTime: time.Now(),
		Message:    sdk.SpawnMsg{ID: sdk.MsgWorkflowNodeStop.ID, Args: []interface{}{ident.GetUsername()}},
	}
	report, err := workflow.StopWorkflowNodeRun(ctx, dbFunc, store, *p, *nodeRun, stopInfos)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to stop workflow node run")
	}

	wr, errLw := workflow.LoadRun(ctx, tx, p.Key, workflowName, nodeRun.Number, workflow.LoadRunOptions{})
	if errLw != nil {
		return nil, sdk.WrapError(errLw, "unable to load workflow run %s", workflowName)
	}

	r1, errR := workflow.ResyncWorkflowRunStatus(ctx, tx, wr)
	if errR != nil {
		return nil, sdk.WrapError(errR, "unable to resync workflow run status")
	}

	report.Merge(ctx, r1)

	observability.Current(ctx,
		observability.Tag(observability.TagProjectKey, p.Key),
		observability.Tag(observability.TagWorkflow, wr.Workflow.Name),
	)
	if wr.Status == sdk.StatusFail {
		observability.Record(api.Router.Background, api.Metrics.WorkflowRunFailed, 1)
	}

	if errC := tx.Commit(); errC != nil {
		return nil, sdk.WrapError(errC, "unable to commit")
	}

	go func(ID int64) {
		wRun, errLw := workflow.LoadRunByID(api.mustDB(), ID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if errLw != nil {
			log.Error(ctx, "workflow.stopWorkflowNodeRun> Cannot load run for resync commit status %v", errLw)
			return
		}
		//The function could be called with nil project so we need to test if project is not nil
		if sdk.StatusIsTerminated(wRun.Status) && p != nil {
			wRun.LastExecution = time.Now()
			if err := workflow.ResyncCommitStatus(context.Background(), api.mustDB(), api.Cache, *p, wRun); err != nil {
				log.Error(ctx, "workflow.stopWorkflowNodeRun> %v", err)
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

		observability.Current(ctx,
			observability.Tag(observability.TagProjectKey, key),
			observability.Tag(observability.TagWorkflow, name),
		)
		observability.Record(api.Router.Background, api.Metrics.WorkflowRunStarted, 1)

		// LOAD PROJECT
		_, next := observability.Span(ctx, "project.Load")
		p, errP := project.Load(api.mustDB(), key,
			project.LoadOptions.WithVariables,
			project.LoadOptions.WithFeatures(api.Cache),
			project.LoadOptions.WithIntegrations,
			project.LoadOptions.WithApplicationVariables,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
		)
		next()
		if errP != nil {
			return sdk.WrapError(errP, "cannot load project")
		}

		// GET BODY
		opts := &sdk.WorkflowRunPostHandlerOption{}
		if err := service.UnmarshalBody(r, opts); err != nil {
			return err
		}

		// Request check
		if opts.Manual != nil && opts.Manual.OnlyFailedJobs && opts.Manual.Resync {
			return sdk.WrapError(sdk.ErrWrongRequest, "You cannot resync workflow and run only failed jobs")
		}

		// CHECK IF IT S AN EXISTING RUN
		var lastRun *sdk.WorkflowRun
		if opts.Number != nil {
			var errlr error
			lastRun, errlr = workflow.LoadRun(ctx, api.mustDB(), key, name, *opts.Number, workflow.LoadRunOptions{})
			if errlr != nil {
				return sdk.WrapError(errlr, "unable to load workflow run")
			}
		}

		c := getAPIConsumer(ctx)
		// To handle conditions on hooks
		if opts.Hook != nil {
			hook, errH := workflow.LoadHookByUUID(api.mustDB(), opts.Hook.WorkflowNodeHookUUID)
			if errH != nil {
				return sdk.WrapError(errH, "cannot load hook for uuid %s", opts.Hook.WorkflowNodeHookUUID)
			}
			conditions := hook.Conditions
			params := sdk.ParametersFromMap(opts.Hook.Payload)

			var errc error
			var conditionsOK bool
			if conditions.LuaScript == "" {
				conditionsOK, errc = sdk.WorkflowCheckConditions(conditions.PlainConditions, params)
			} else {
				luacheck, err := luascript.NewCheck()
				if err != nil {
					return sdk.WrapError(err, "cannot check lua script")
				}
				luacheck.SetVariables(sdk.ParametersToMap(params))
				errc = luacheck.Perform(conditions.LuaScript)
				conditionsOK = luacheck.Result
			}
			if errc != nil {
				return sdk.WrapError(errc, "cannot check conditions")
			}

			if !conditionsOK {
				return sdk.WithStack(sdk.ErrConditionsNotOk)
			}
		}

		var wf *sdk.Workflow
		// IF CONTINUE EXISTING RUN
		if lastRun != nil {
			if opts != nil && opts.Manual != nil && opts.Manual.Resync {
				log.Debug("Resync workflow %d for run %d", lastRun.Workflow.ID, lastRun.ID)
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

				if !permission.AccessToWorkflowNode(ctx, api.mustDB(), &lastRun.Workflow, fromNode, getAPIConsumer(ctx), sdk.PermissionReadExecute) {
					return sdk.WrapError(sdk.ErrNoPermExecution, "not enough right on node %s", fromNode.Name)
				}
			}

			lastRun.Status = sdk.StatusWaiting
		} else {
			var errWf error
			wf, errWf = workflow.Load(ctx, api.mustDB(), api.Cache, *p, name, workflow.LoadOptions{
				DeepPipeline:          true,
				Base64Keys:            true,
				WithAsCodeUpdateEvent: true,
				WithIcon:              true,
				WithIntegrations:      true,
				WithTemplate:          true,
			})
			if errWf != nil {
				return sdk.WrapError(errWf, "unable to load workflow %s", name)
			}

			// Check node permission
			if isService := isService(ctx); !isService && !permission.AccessToWorkflowNode(ctx, api.mustDB(), wf, &wf.WorkflowData.Node, getAPIConsumer(ctx), sdk.PermissionReadExecute) {
				return sdk.WrapError(sdk.ErrNoPermExecution, "not enough right on node %s", wf.WorkflowData.Node.Name)
			}

			// CREATE WORKFLOW RUN
			var errCreateRun error
			lastRun, errCreateRun = workflow.CreateRun(api.mustDB(), wf, opts, c)
			if errCreateRun != nil {
				return errCreateRun
			}
		}

		// Workflow Run initialization
		sdk.GoRoutine(context.Background(), fmt.Sprintf("api.initWorkflowRun-%d", lastRun.ID), func(ctx context.Context) {
			api.initWorkflowRun(ctx, p.Key, wf, lastRun, opts, c)
		}, api.PanicDump())

		return service.WriteJSON(w, lastRun, http.StatusAccepted)
	}
}

func (api *API) initWorkflowRun(ctx context.Context, projKey string, wf *sdk.Workflow, wfRun *sdk.WorkflowRun, opts *sdk.WorkflowRunPostHandlerOption, u *sdk.AuthConsumer) {
	var asCodeInfosMsg []sdk.Message
	report := new(workflow.ProcessorReport)

	p, err := project.Load(api.mustDB(), projKey,
		project.LoadOptions.WithVariables,
		project.LoadOptions.WithFeatures(api.Cache),
		project.LoadOptions.WithIntegrations,
		project.LoadOptions.WithApplicationVariables,
		project.LoadOptions.WithApplicationWithDeploymentStrategies,
		project.LoadOptions.WithEnvironments,
		project.LoadOptions.WithPipelines,
	)
	if err != nil {
		r := failInitWorkflowRun(ctx, api.mustDB(), wfRun, sdk.WrapError(err, "cannot load project for as code workflow creation"))
		report.Merge(ctx, r)
		return
	}

	defer func() {
		go WorkflowSendEvent(context.Background(), api.mustDB(), api.Cache, *p, report)
	}()

	if wfRun.Status == sdk.StatusPending {
		// Sync as code event to remove events in case where a PR was merged
		if len(wf.AsCodeEvent) > 0 {
			if wf.WorkflowData.Node.Context.ApplicationID == 0 {
				r1 := failInitWorkflowRun(ctx, api.mustDB(), wfRun, sdk.WrapError(sdk.ErrNotFound, "unable to find application on root node"))
				report.Merge(ctx, r1)
				return
			}
			app := wf.Applications[wf.WorkflowData.Node.Context.ApplicationID]

			res, err := ascode.SyncEvents(ctx, api.mustDB(), api.Cache, *p, app, u.AuthentifiedUser)
			if err != nil {
				r := failInitWorkflowRun(ctx, api.mustDB(), wfRun, sdk.WrapError(err, "unable to sync as code event"))
				report.Merge(ctx, r)
				return
			}
			for _, id := range res.MergedWorkflow {
				if err := workflow.UpdateFromRepository(api.mustDB(), id, res.FromRepository); err != nil {
					r := failInitWorkflowRun(ctx, api.mustDB(), wfRun, sdk.WrapError(err, "unable to sync as code event"))
					report.Merge(ctx, r)
					return
				}
				if id == wf.ID {
					wf.FromRepository = res.FromRepository
					event.PublishWorkflowUpdate(ctx, p.Key, *wf, *wf, u)
				}
			}
		}

		// If the workflow is as code we need to reimport it.
		// NOTICE: Only repository webhooks and manual run will perform the repository analysis.
		workflowStartedByRepoWebHook := opts.Hook != nil && wf.WorkflowData.Node.GetHook(opts.Hook.WorkflowNodeHookUUID) != nil &&
			wf.WorkflowData.Node.GetHook(opts.Hook.WorkflowNodeHookUUID).HookModelName == sdk.RepositoryWebHookModelName

		if wf.FromRepository != "" && (workflowStartedByRepoWebHook || opts.Manual != nil) {
			log.Debug("initWorkflowRun> rebuild workflow %s/%s from as code configuration", p.Key, wf.Name)
			p1, err := project.Load(api.mustDB(), projKey,
				project.LoadOptions.WithVariables,
				project.LoadOptions.WithGroups,
				project.LoadOptions.WithApplicationVariables,
				project.LoadOptions.WithApplicationWithDeploymentStrategies,
				project.LoadOptions.WithEnvironments,
				project.LoadOptions.WithPipelines,
				project.LoadOptions.WithClearKeys,
				project.LoadOptions.WithClearIntegrations,
				project.LoadOptions.WithFeatures(api.Cache),
			)
			if err != nil {
				r := failInitWorkflowRun(ctx, api.mustDB(), wfRun, sdk.WrapError(err, "cannot load project for as code workflow creation"))
				report.Merge(ctx, r)
				return
			}

			// Get workflow from repository
			log.Debug("workflow.CreateFromRepository> %s", wf.Name)
			oldWf := *wf
			asCodeInfosMsg, err := workflow.CreateFromRepository(ctx, api.mustDB(), api.Cache, p1, wf, *opts, *u, project.DecryptWithBuiltinKey)
			if err != nil {
				infos := make([]sdk.SpawnMsg, len(asCodeInfosMsg))
				for i, msg := range asCodeInfosMsg {
					infos[i] = sdk.SpawnMsg{
						ID:   msg.ID,
						Args: msg.Args,
						Type: msg.Type,
					}

				}
				workflow.AddWorkflowRunInfo(wfRun, infos...)
				r1 := failInitWorkflowRun(ctx, api.mustDB(), wfRun, sdk.WrapError(err, "unable to get workflow from repository"))
				report.Merge(ctx, r1)
				return
			}

			event.PublishWorkflowUpdate(ctx, p.Key, *wf, oldWf, u)
		}

		wfRun.Workflow = *wf
	}

	r, err := workflow.StartWorkflowRun(ctx, api.mustDB(), api.Cache, *p, wfRun, opts, u, asCodeInfosMsg)
	report.Merge(ctx, r)
	if err != nil {
		r := failInitWorkflowRun(ctx, api.mustDB(), wfRun, sdk.WrapError(err, "unable to start workflow %s/%s", p.Key, wf.Name))
		report.Merge(ctx, r)
		return
	}

	workflow.ResyncNodeRunsWithCommits(ctx, api.mustDB(), api.Cache, *p, report)

	// Purge workflow run
	sdk.GoRoutine(ctx, "workflow.PurgeWorkflowRun", func(ctx context.Context) {
		if err := workflow.PurgeWorkflowRun(ctx, api.mustDB(), *wf, api.Metrics.WorkflowRunsMarkToDelete); err != nil {
			log.Error(ctx, "workflow.PurgeWorkflowRun> error %v", err)
		}
	}, api.PanicDump())
}

func failInitWorkflowRun(ctx context.Context, db *gorp.DbMap, wfRun *sdk.WorkflowRun, err error) *workflow.ProcessorReport {
	report := new(workflow.ProcessorReport)

	var info sdk.SpawnMsg
	if sdk.ErrorIs(err, sdk.ErrConditionsNotOk) {
		info = sdk.SpawnMsg{
			ID:   sdk.MsgWorkflowConditionError.ID,
			Type: sdk.MsgWorkflowConditionError.Type,
		}
		if len(wfRun.WorkflowNodeRuns) == 0 {
			wfRun.Status = sdk.StatusNeverBuilt
		}
	} else {
		httpErr := sdk.ExtractHTTPError(err, "")
		isErrWithStack := sdk.IsErrorWithStack(err)
		fields := logrus.Fields{}
		if isErrWithStack {
			fields["stack_trace"] = fmt.Sprintf("%+v", err)
		}
		log.ErrorWithFields(ctx, fields, "%s", err)
		wfRun.Status = sdk.StatusFail
		info = sdk.SpawnMsg{
			ID:   sdk.MsgWorkflowError.ID,
			Args: []interface{}{httpErr.Error()},
			Type: sdk.MsgWorkflowError.Type,
		}
	}

	workflow.AddWorkflowRunInfo(wfRun, info)
	if errU := workflow.UpdateWorkflowRun(ctx, db, wfRun); errU != nil {
		log.Error(ctx, "unable to fail workflow run %v", errU)
	}
	report.Add(ctx, wfRun)
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

		f, err := api.SharedStorage.Fetch(ctx, art)
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
			return sdk.NewErrorFrom(sdk.ErrInvalidID, "invalid node job run ID")
		}

		proj, err := project.Load(api.mustDB(), key, project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load projet")
		}

		options := workflow.LoadOptions{}
		work, err := workflow.Load(ctx, api.mustDB(), api.Cache, *proj, name, options)
		if err != nil {
			return sdk.WrapError(err, "cannot load workflow")
		}

		art, err := workflow.LoadArtifactByIDs(api.mustDB(), work.ID, id)
		if err != nil {
			return sdk.WrapError(err, "cannot load artifacts")
		}

		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", art.Name))

		var integrationName string
		if art.ProjectIntegrationID != nil && *art.ProjectIntegrationID > 0 {
			projectIntegration, err := integration.LoadProjectIntegrationByID(api.mustDB(), *art.ProjectIntegrationID)
			if err != nil {
				return sdk.WrapError(err, "cannot load project integration %s/%d", proj.Key, *art.ProjectIntegrationID)
			}
			integrationName = projectIntegration.Name
		} else {
			integrationName = sdk.DefaultStorageIntegrationName
		}

		storageDriver, err := objectstore.GetDriver(ctx, api.mustDB(), api.SharedStorage, proj.Key, integrationName)
		if err != nil {
			return err
		}

		f, err := storageDriver.Fetch(ctx, art)
		if err != nil {
			_ = f.Close()
			return sdk.WrapError(err, "cannot fetch artifact")
		}

		if _, err := io.Copy(w, f); err != nil {
			_ = f.Close()
			return sdk.WrapError(err, "cannot stream artifact")
		}

		if err := f.Close(); err != nil {
			return sdk.WrapError(err, "cannot close artifact")
		}
		return nil
	}
}

func (api *API) getWorkflowRunArtifactsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		number, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}

		wr, err := workflow.LoadRun(ctx, api.mustDB(), key, name, number, workflow.LoadRunOptions{WithArtifacts: true})
		if err != nil {
			return err
		}

		arts := []sdk.WorkflowNodeRunArtifact{}
		for _, runs := range wr.WorkflowNodeRuns {
			if len(runs) == 0 {
				continue
			}

			sort.Slice(runs, func(i, j int) bool {
				return runs[i].SubNumber > runs[j].SubNumber
			})

			artifacts := workflow.MergeArtifactWithPreviousSubRun(runs)

			wg := &sync.WaitGroup{}
			for i := range artifacts {
				wg.Add(1)
				go func(art *sdk.WorkflowNodeRunArtifact) {
					defer wg.Done()

					var integrationName string
					if art.ProjectIntegrationID != nil && *art.ProjectIntegrationID > 0 {
						projectIntegration, err := integration.LoadProjectIntegrationByID(api.mustDB(), *art.ProjectIntegrationID)
						if err != nil {
							log.Error(ctx, "Cannot load LoadProjectIntegrationByID %s/%d: err: %v", key, *art.ProjectIntegrationID, err)
							return
						}
						integrationName = projectIntegration.Name
					} else {
						integrationName = sdk.DefaultStorageIntegrationName
					}

					storageDriver, err := objectstore.GetDriver(ctx, api.mustDB(), api.SharedStorage, key, integrationName)
					if err != nil {
						log.Error(ctx, "Cannot load storage driver: %v", err)
						return
					}

					s, temporaryURLSupported := storageDriver.(objectstore.DriverWithRedirect)
					if temporaryURLSupported { // with temp URL
						fURL, _, err := s.FetchURL(art)
						if err != nil {
							log.Error(ctx, "Cannot fetch cache object: %v", err)
						} else if fURL != "" {
							art.TempURL = fURL
						}
					}
				}(&artifacts[i])
			}
			wg.Wait()
			arts = append(arts, artifacts...)
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

		spawnInfos, err := workflow.LoadNodeRunJobInfo(ctx, db, runJobID)
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
			Status:   stepStatus,
			StepLogs: *ls,
		}

		log.Debug("logs: %+v", result)

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

		proj, err := project.Load(db, key, project.LoadOptions.WithVariables)
		if err != nil {
			return sdk.WrapError(err, "cannot load project")
		}

		wfr, err := workflow.LoadRun(ctx, db, key, name, number, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if err != nil {
			return sdk.WrapError(err, "cannot load workflow run")
		}

		if err := workflow.ResyncCommitStatus(ctx, db, api.Cache, *proj, wfr); err != nil {
			return sdk.WrapError(err, "cannot resync workflow run commit status")
		}

		return nil
	}
}
