package api

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/artifact"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const (
	rangeMax     = 50
	defaultLimit = 10
)

func (api *API) getWorkflowRunsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// About pagination: [FR] http://blog.octo.com/designer-une-api-rest/#pagination
		vars := mux.Vars(r)
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

		//Maximim range is set to 50
		w.Header().Add("Accept-Range", "run 50")
		if limit-offset > rangeMax {
			return sdk.WrapError(sdk.ErrWrongRequest, "getWorkflowRunsHandler> Requested range %d not allowed", (limit - offset))
		}

		key := vars["key"]
		name := vars["permWorkflowName"]
		runs, offset, limit, count, err := workflow.LoadRuns(api.mustDB(), key, name, offset, limit)
		if err != nil {
			return sdk.WrapError(err, "getWorkflowRunsHandler> Unable to load workflow runs")
		}

		if offset > count {
			return sdk.WrapError(sdk.ErrWrongRequest, "getWorkflowRunsHandler> Requested range %d not allowed", (limit - offset))
		}

		code := http.StatusOK

		//RFC5988: Link : <https://api.fakecompany.com/v1/orders?range=0-7>; rel="first", <https://api.fakecompany.com/v1/orders?range=40-47>; rel="prev", <https://api.fakecompany.com/v1/orders?range=56-64>; rel="next", <https://api.fakecompany.com/v1/orders?range=968-975>; rel="last"
		if len(runs) < count {
			baseLinkURL := api.Router.URL +
				api.Router.GetRoute("GET", api.getWorkflowRunsHandler, map[string]string{
					"permProjectKey": key,
					"workflowName":   name,
				})
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
		return WriteJSON(w, r, runs, code)
	}
}

func (api *API) getLatestWorkflowRunHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		run, err := workflow.LoadLastRun(api.mustDB(), key, name)
		if err != nil {
			return sdk.WrapError(err, "getLatestWorkflowRunHandler> Unable to load last workflow run")
		}
		run.Translate(r.Header.Get("Accept-Language"))
		return WriteJSON(w, r, run, http.StatusOK)
	}
}

func (api *API) resyncWorkflowRunPipelinesHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		number, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}
		run, err := workflow.LoadRun(api.mustDB(), key, name, number)
		if err != nil {
			return sdk.WrapError(err, "resyncWorkflowRunPipelinesHandler> Unable to load last workflow run [%s/%d]", name, number)
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "resyncWorkflowRunPipelinesHandler> Cannot start transaction")
		}

		if err := workflow.ResyncPipeline(tx, run); err != nil {
			return sdk.WrapError(err, "resyncWorkflowRunPipelinesHandler> Cannot resync pipelines")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "resyncWorkflowRunPipelinesHandler> Cannot commit transaction")
		}
		return WriteJSON(w, r, run, http.StatusOK)
	}
}

func (api *API) getWorkflowRunHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		number, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}
		run, err := workflow.LoadRun(api.mustDB(), key, name, number)
		if err != nil {
			return sdk.WrapError(err, "getWorkflowRunHandler> Unable to load last workflow run")
		}
		run.Translate(r.Header.Get("Accept-Language"))
		return WriteJSON(w, r, run, http.StatusOK)
	}
}

func (api *API) stopWorkflowRunHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		number, err := requestVarInt(r, "number")
		if err != nil {
			return err
		}

		run, errL := workflow.LoadRun(api.mustDB(), key, name, number)
		if errL != nil {
			return sdk.WrapError(errL, "stopWorkflowRunHandler> Unable to load last workflow run")
		}

		proj, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "stopWorkflowRunHandler> Unable to load project")
		}

		chanEvent := make(chan interface{}, 1)
		chanError := make(chan error, 1)

		go stopWorkflowRun(chanEvent, chanError, api.mustDB(), api.Cache, proj, run, getUser(ctx))

		workflowRuns, workflowNodeRuns, workflowNodeJobRuns, err := workflow.GetWorkflowRunEventData(chanError, chanEvent)
		if err != nil {
			return err
		}
		go workflow.SendEvent(api.mustDB(), workflowRuns, workflowNodeRuns, workflowNodeJobRuns, proj.Key)

		return WriteJSON(w, r, run, http.StatusOK)
	}
}

func stopWorkflowRun(chEvent chan<- interface{}, chError chan<- error, db *gorp.DbMap, store cache.Store, p *sdk.Project, run *sdk.WorkflowRun, u *sdk.User) {
	defer close(chEvent)
	defer close(chError)

	tx, errTx := db.Begin()
	if errTx != nil {
		chError <- sdk.WrapError(errTx, "stopWorkflowRunHandler> Unable to create transaction")
	}
	defer tx.Rollback()

	stopInfos := sdk.SpawnInfo{
		APITime:    time.Now(),
		RemoteTime: time.Now(),
		Message:    sdk.SpawnMsg{ID: sdk.MsgWorkflowNodeStop.ID, Args: []interface{}{u.Username}},
	}

	for _, wn := range run.WorkflowNodeRuns {
		for _, wnr := range wn {
			if wnr.SubNumber != run.LastSubNumber || (wnr.Status == sdk.StatusSuccess.String() ||
				wnr.Status == sdk.StatusFail.String() || wnr.Status == sdk.StatusSkipped.String()) {
				log.Debug("stopWorkflowRunHandler> cannot stop this workflow node run with current status %s", wnr.Status)
				continue
			}

			if errS := workflow.StopWorkflowNodeRun(tx, store, p, wnr, stopInfos, chEvent); errS != nil {
				chError <- sdk.WrapError(errS, "stopWorkflowRunHandler> Unable to stop workflow node run %d", wnr.ID)
			}
			wnr.Status = sdk.StatusStopped.String()
		}
	}

	if errU := workflow.UpdateWorkflowRunStatus(tx, run.ID, sdk.StatusStopped.String()); errU != nil {
		chError <- sdk.WrapError(errU, "stopWorkflowRunHandler> Unable to update workflow run status %d", run.ID)
	}

	if err := tx.Commit(); err != nil {
		chError <- sdk.WrapError(err, "stopWorkflowRunHandler> Cannot commit transaction")
	}
}

func (api *API) getWorkflowNodeRunHistoryHandler() Handler {
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

		run, errR := workflow.LoadRun(api.mustDB(), key, name, number)
		if errR != nil {
			return sdk.WrapError(errR, "getWorkflowNodeRunHistoryHandler")
		}

		nodeRuns, ok := run.WorkflowNodeRuns[nodeID]
		if !ok {
			return sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "getWorkflowNodeRunHistoryHandler")
		}
		return WriteJSON(w, r, nodeRuns, http.StatusOK)
	}
}

func (api *API) stopWorkflowNodeRunHandler() Handler {
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

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithVariables)
		if errP != nil {
			return sdk.WrapError(errP, "stopWorkflowNodeRunHandler> Cannot load project")
		}

		// Load node run
		nodeRun, err := workflow.LoadNodeRun(api.mustDB(), key, name, number, id)
		if err != nil {
			return sdk.WrapError(err, "stopWorkflowNodeRunHandler> Unable to load last workflow run")
		}

		chanEvent := make(chan interface{}, 1)
		chanError := make(chan error, 1)

		go stopWorkflowNodeRun(chanEvent, chanError, api.mustDB(), api.Cache, p, nodeRun, name, getUser(ctx))

		workflowRuns, workflowNodeRuns, workflowNodeJobRuns, err := workflow.GetWorkflowRunEventData(chanError, chanEvent)
		if err != nil {
			return err
		}
		go workflow.SendEvent(api.mustDB(), workflowRuns, workflowNodeRuns, workflowNodeJobRuns, p.Key)

		return nil
	}
}

func stopWorkflowNodeRun(chEvent chan<- interface{}, chError chan<- error, db *gorp.DbMap, store cache.Store, p *sdk.Project, nodeRun *sdk.WorkflowNodeRun, workflowName string, u *sdk.User) {
	tx, errTx := db.Begin()
	if errTx != nil {
		chError <- sdk.WrapError(errTx, "stopWorkflowNodeRunHandler> Unable to create transaction")
	}
	defer tx.Rollback()

	stopInfos := sdk.SpawnInfo{
		APITime:    time.Now(),
		RemoteTime: time.Now(),
		Message:    sdk.SpawnMsg{ID: sdk.MsgWorkflowNodeStop.ID, Args: []interface{}{u.Username}},
	}
	if errS := workflow.StopWorkflowNodeRun(tx, store, p, *nodeRun, stopInfos, chEvent); errS != nil {
		chError <- sdk.WrapError(errS, "stopWorkflowNodeRunHandler> Unable to stop workflow node run")
	}

	wr, errLw := workflow.LoadRun(tx, p.Key, workflowName, nodeRun.Number)
	if errLw != nil {
		chError <- sdk.WrapError(errLw, "stopWorkflowNodeRunHandler> Unable to load workflow run %s", workflowName)
	}

	if errR := workflow.ResyncWorkflowRunStatus(tx, wr, chEvent); errR != nil {
		chError <- sdk.WrapError(errR, "stopWorkflowNodeRunHandler> Unable to resync workflow run status")
	}

	if errC := tx.Commit(); errC != nil {
		chError <- sdk.WrapError(errC, "stopWorkflowNodeRunHandler> Unable to commit")
	}
}

func (api *API) getWorkflowNodeRunHandler() Handler {
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
		run, err := workflow.LoadNodeRun(api.mustDB(), key, name, number, id)
		if err != nil {
			return sdk.WrapError(err, "getWorkflowRunHandler> Unable to load last workflow run")
		}
		run.Translate(r.Header.Get("Accept-Language"))
		return WriteJSON(w, r, run, http.StatusOK)
	}
}

func (api *API) postWorkflowRunHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithVariables)
		if errP != nil {
			return sdk.WrapError(errP, "postWorkflowRunHandler> Cannot load project")
		}

		opts := &sdk.WorkflowRunPostHandlerOption{}
		if err := UnmarshalBody(r, opts); err != nil {
			return err
		}

		wf, errl := workflow.Load(api.mustDB(), api.Cache, key, name, getUser(ctx))
		if errl != nil {
			return sdk.WrapError(errl, "postWorkflowRunHandler> Unable to load workflow")
		}

		var lastRun *sdk.WorkflowRun
		if opts.Number != nil {
			var errlr error
			lastRun, errlr = workflow.LoadRun(api.mustDB(), key, name, *opts.Number)
			if errlr != nil {
				return sdk.WrapError(errlr, "postWorkflowRunHandler> Unable to load workflow run")
			}
		}

		chanEvent := make(chan interface{}, 1)
		chanError := make(chan error, 1)

		go startWorkflowRun(chanEvent, chanError, api.mustDB(), api.Cache, p, wf, lastRun, opts, getUser(ctx))

		workflowRuns, workflowNodeRuns, workflowNodeJobRuns, err := workflow.GetWorkflowRunEventData(chanError, chanEvent)
		if err != nil {
			return err
		}
		go workflow.SendEvent(api.mustDB(), workflowRuns, workflowNodeRuns, workflowNodeJobRuns, p.Key)

		// Purge workflow run
		go workflow.PurgeWorkflowRun(api.mustDB(), *wf)

		var wr *sdk.WorkflowRun
		if len(workflowRuns) > 0 {
			wr = &workflowRuns[0]
			wr.Translate(r.Header.Get("Accept-Language"))
		}
		return WriteJSON(w, r, wr, http.StatusAccepted)
	}
}

func startWorkflowRun(chEvent chan<- interface{}, chError chan<- error, db *gorp.DbMap, store cache.Store, p *sdk.Project, wf *sdk.Workflow, lastRun *sdk.WorkflowRun, opts *sdk.WorkflowRunPostHandlerOption, u *sdk.User) {
	defer close(chEvent)
	defer close(chError)

	tx, errb := db.Begin()
	if errb != nil {
		chError <- sdk.WrapError(errb, "startWorkflowRun> Cannot start transaction")
	}
	defer tx.Rollback()

	//Run from hook
	if opts.Hook != nil {
		var errfh error
		_, errfh = workflow.RunFromHook(tx, store, p, wf, opts.Hook, chEvent)
		if errfh != nil {
			chError <- sdk.WrapError(errfh, "postWorkflowRunHandler> Unable to run workflow from hook")
		}
	} else {
		//Default manual run
		if opts.Manual == nil {
			opts.Manual = &sdk.WorkflowNodeRunManual{}
		}
		opts.Manual.User = *u
		opts.Manual.User.Groups = nil

		fromNodes := []*sdk.WorkflowNode{}
		if len(opts.FromNodeIDs) > 0 {
			for _, fromNodeID := range opts.FromNodeIDs {
				fromNode := lastRun.Workflow.GetNode(fromNodeID)
				if fromNode == nil {
					chError <- sdk.WrapError(sdk.ErrWorkflowNodeNotFound, "postWorkflowRunHandler> Payload: Unable to get node %d", fromNodeID)
				}
				fromNodes = append(fromNodes, fromNode)
			}
		} else {
			fromNodes = append(fromNodes, wf.Root)
		}

		for _, fromNode := range fromNodes {
			// Check Env Permission
			if fromNode.Context.Environment != nil {
				if !permission.AccessToEnvironment(fromNode.Context.Environment.ID, u, permission.PermissionReadExecute) {
					chError <- sdk.WrapError(sdk.ErrNoEnvExecution, "postWorkflowRunHandler> Not enough right to run on environment %s", fromNode.Context.Environment.Name)
				}
			}

			//If payload is not set, keep the default payload
			if opts.Manual.Payload == interface{}(nil) {
				opts.Manual.Payload = fromNode.Context.DefaultPayload
			}

			//If PipelineParameters are not set, keep the default PipelineParameters
			if len(opts.Manual.PipelineParameters) == 0 {
				opts.Manual.PipelineParameters = fromNode.Context.DefaultPipelineParameters
			}
			log.Debug("Manual run: %#v", opts.Manual)

			//Manual run
			if lastRun != nil {
				var errmr error
				_, errmr = workflow.ManualRunFromNode(tx, store, p, wf, lastRun.Number, opts.Manual, fromNode.ID, chEvent)
				if errmr != nil {
					chError <- sdk.WrapError(errmr, "postWorkflowRunHandler> Unable to run workflow from node")
				}
			}
		}

		if lastRun == nil {
			var errmr error
			_, errmr = workflow.ManualRun(tx, store, p, wf, opts.Manual, chEvent)
			if errmr != nil {
				chError <- sdk.WrapError(errmr, "postWorkflowRunHandler> Unable to run workflow")
			}
		}
	}

	//Commit and return success
	if err := tx.Commit(); err != nil {
		chError <- sdk.WrapError(err, "postWorkflowRunHandler> Unable to commit transaction")
	}
}

func (api *API) downloadworkflowArtifactDirectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		hash := vars["hash"]

		art, err := workflow.LoadWorkfowArtifactByHash(api.mustDB(), hash)
		if err != nil {
			return sdk.WrapError(err, "downloadworkflowArtifactDirectHandler> Could not load artifact with hash %s", hash)
		}

		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", art.Name))

		log.Debug("downloadworkflowArtifactDirectHandler: Serving %+v", art)
		if err := artifact.StreamFile(w, art); err != nil {
			return sdk.WrapError(err, "downloadworkflowArtifactDirectHandler: Cannot stream artifact")
		}
		return nil
	}
}

func (api *API) getWorkflowNodeRunArtifactsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		number, errNu := requestVarInt(r, "number")
		if errNu != nil {
			return sdk.WrapError(errNu, "getWorkflowJobArtifactsHandler> Invalid node job run ID")
		}

		id, errI := requestVarInt(r, "nodeRunID")
		if errI != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "getWorkflowJobArtifactsHandler> Invalid node job run ID")
		}
		nodeRun, errR := workflow.LoadNodeRun(api.mustDB(), key, name, number, id)
		if errR != nil {
			return sdk.WrapError(errR, "getWorkflowJobArtifactsHandler> Cannot load node run")
		}

		return WriteJSON(w, r, nodeRun.Artifacts, http.StatusOK)
	}
}

func (api *API) getDownloadArtifactHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		id, errI := requestVarInt(r, "artifactId")
		if errI != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "getDownloadArtifactHandler> Invalid node job run ID")
		}

		work, errW := workflow.Load(api.mustDB(), api.Cache, key, name, getUser(ctx))
		if errW != nil {
			return sdk.WrapError(errW, "getDownloadArtifactHandler> Cannot load workflow")
		}

		art, errA := workflow.LoadArtifactByIDs(api.mustDB(), work.ID, id)
		if errA != nil {
			return sdk.WrapError(errA, "getDownloadArtifactHandler> Cannot load artifacts")
		}

		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", art.Name))

		if err := artifact.StreamFile(w, art); err != nil {
			return sdk.WrapError(err, "Cannot stream artifact %s", art.Name)
		}
		return nil
	}
}

func (api *API) getWorkflowRunArtifactsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		number, errNu := requestVarInt(r, "number")
		if errNu != nil {
			return sdk.WrapError(errNu, "getWorkflowJobArtifactsHandler> Invalid node job run ID")
		}

		wr, errW := workflow.LoadRun(api.mustDB(), key, name, number)
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

			arts = append(arts, runs[0].Artifacts...)
		}

		return WriteJSON(w, r, arts, http.StatusOK)
	}
}

func (api *API) getWorkflowNodeRunJobStepHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		workflowName := vars["permWorkflowName"]
		number, errN := requestVarInt(r, "number")
		if errN != nil {
			return sdk.WrapError(errN, "getWorkflowNodeRunJobBuildLogsHandler> Number: invalid number")
		}
		nodeRunID, errNI := requestVarInt(r, "nodeRunID")
		if errNI != nil {
			return sdk.WrapError(errNI, "getWorkflowNodeRunJobBuildLogsHandler> id: invalid number")
		}
		runJobID, errJ := requestVarInt(r, "runJobId")
		if errJ != nil {
			return sdk.WrapError(errJ, "getWorkflowNodeRunJobBuildLogsHandler> runJobId: invalid number")
		}
		stepOrder, errS := requestVarInt(r, "stepOrder")
		if errS != nil {
			return sdk.WrapError(errS, "getWorkflowNodeRunJobBuildLogsHandler> stepOrder: invalid number")
		}

		// Check workflow is in project
		if _, errW := workflow.Load(api.mustDB(), api.Cache, projectKey, workflowName, getUser(ctx)); errW != nil {
			return sdk.WrapError(errW, "getWorkflowNodeRunJobBuildLogsHandler> Cannot find workflow %s in project %s", workflowName, projectKey)
		}

		// Check nodeRunID is link to workflow
		nodeRun, errNR := workflow.LoadNodeRun(api.mustDB(), projectKey, workflowName, number, nodeRunID)
		if errNR != nil {
			return sdk.WrapError(errNR, "getWorkflowNodeRunJobBuildLogsHandler> Cannot find nodeRun %d/%d for workflow %s in project %s", nodeRunID, number, workflowName, projectKey)
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
			return sdk.WrapError(fmt.Errorf("getWorkflowNodeRunJobStepHandler> Cannot find step %d on job %d in nodeRun %d/%d for workflow %s in project %s",
				stepOrder, runJobID, nodeRunID, number, workflowName, projectKey), "")
		}

		logs, errL := workflow.LoadStepLogs(api.mustDB(), runJobID, stepOrder)
		if errL != nil {
			return sdk.WrapError(errL, "getWorkflowNodeRunJobStepHandler> Cannot load log for runJob %d on step %d", runJobID, stepOrder)
		}

		ls := &sdk.Log{}
		if logs != nil {
			ls = logs
		}
		result := &sdk.BuildState{
			Status:   sdk.StatusFromString(stepStatus),
			StepLogs: *ls,
		}

		return WriteJSON(w, r, result, http.StatusOK)
	}
}

func (api *API) getWorkflowRunTagsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		workflowName := vars["permWorkflowName"]

		res, err := workflow.GetTagsAndValue(api.mustDB(), projectKey, workflowName)
		if err != nil {
			return sdk.WrapError(err, "getWorkflowRunTagsHandler> Error")
		}

		return WriteJSON(w, r, res, http.StatusOK)
	}
}
