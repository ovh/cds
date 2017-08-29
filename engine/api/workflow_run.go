package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/artifact"
	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const (
	rangeMax     = 50
	defaultLimit = 10
)

func getWorkflowRunsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

	key := vars["permProjectKey"]
	name := vars["workflowName"]
	runs, offset, limit, count, err := workflow.LoadRuns(db, key, name, offset, limit)
	if err != nil {
		return sdk.WrapError(err, "getWorkflowRunsHandler> Unable to load workflow runs")
	}

	if limit-offset > count {
		return sdk.WrapError(sdk.ErrWrongRequest, "getWorkflowRunsHandler> Requested range %d not allowed", (limit - offset))
	}

	code := http.StatusOK

	//RFC5988: Link : <https://api.fakecompany.com/v1/orders?range=0-7>; rel="first", <https://api.fakecompany.com/v1/orders?range=40-47>; rel="prev", <https://api.fakecompany.com/v1/orders?range=56-64>; rel="next", <https://api.fakecompany.com/v1/orders?range=968-975>; rel="last"
	if len(runs) < count {
		baseLinkURL := router.url +
			router.getRoute("GET", getWorkflowRunsHandler, map[string]string{
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

	return WriteJSON(w, r, runs, code)
}

func getLatestWorkflowRunHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	name := vars["workflowName"]
	run, err := workflow.LoadLastRun(db, key, name)
	if err != nil {
		return sdk.WrapError(err, "getLatestWorkflowRunHandler> Unable to load last workflow run")
	}
	run.Translate(r.Header.Get("Accept-Language"))
	return WriteJSON(w, r, run, http.StatusOK)
}

func getWorkflowRunHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	name := vars["workflowName"]
	number, err := requestVarInt(r, "number")
	if err != nil {
		return err
	}
	run, err := workflow.LoadRun(db, key, name, number)
	if err != nil {
		return sdk.WrapError(err, "getWorkflowRunHandler> Unable to load last workflow run")
	}
	run.Translate(r.Header.Get("Accept-Language"))
	return WriteJSON(w, r, run, http.StatusOK)
}

func getWorkflowNodeRunHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	name := vars["workflowName"]
	number, err := requestVarInt(r, "number")
	if err != nil {
		return err
	}
	id, err := requestVarInt(r, "id")
	if err != nil {
		return err
	}
	run, err := workflow.LoadNodeRun(db, key, name, number, id)
	if err != nil {
		return sdk.WrapError(err, "getWorkflowRunHandler> Unable to load last workflow run")
	}
	run.Translate(r.Header.Get("Accept-Language"))
	return WriteJSON(w, r, run, http.StatusOK)
}

type postWorkflowRunHandlerOption struct {
	Hook       *sdk.WorkflowNodeRunHookEvent `json:"hook,omitempty"`
	Manual     *sdk.WorkflowNodeRunManual    `json:"manual,omitempty"`
	Number     *int64                        `json:"number,omitempty"`
	FromNodeID *int64                        `json:"from_node,omitempty"`
}

func postWorkflowRunHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	name := vars["workflowName"]

	tx, errb := db.Begin()
	if errb != nil {
		return errb
	}
	defer tx.Rollback()

	opts := &postWorkflowRunHandlerOption{}
	if err := UnmarshalBody(r, opts); err != nil {
		return err
	}

	wf, errl := workflow.Load(tx, key, name, c.User)
	if errl != nil {
		return sdk.WrapError(errl, "postWorkflowRunHandler> Unable to load workflow")
	}

	var lastRun *sdk.WorkflowRun
	if opts.Number != nil {
		var errlr error
		lastRun, errlr = workflow.LoadRun(tx, key, name, *opts.Number)
		if errlr != nil {
			return sdk.WrapError(errlr, "postWorkflowRunHandler> Unable to load workflow run")
		}
	}

	var wr *sdk.WorkflowRun

	//Run from hook
	if opts.Hook != nil {
		var errfh error
		wr, errfh = workflow.RunFromHook(tx, wf, opts.Hook)
		if errfh != nil {
			return sdk.WrapError(errfh, "postWorkflowRunHandler> Unable to run workflow")
		}
	} else {
		//Default manual run
		if opts.Manual == nil {
			opts.Manual = &sdk.WorkflowNodeRunManual{
				User: *c.User,
			}
		}

		//If payload is not set, keep the default payload
		if opts.Manual.Payload == interface{}(nil) {
			n := wf.Root
			if opts.FromNodeID != nil {
				n = wf.GetNode(*opts.FromNodeID)
				if n == nil {
					return sdk.WrapError(sdk.ErrWorkflowNotFound, "postWorkflowRunHandler> Unable to run workflow")
				}
			}
			opts.Manual.Payload = n.Context.DefaultPayload
		}

		//If PipelineParameters are not set, keep the default PipelineParameters
		if len(opts.Manual.PipelineParameters) == 0 {
			n := wf.Root
			if opts.FromNodeID != nil {
				n = wf.GetNode(*opts.FromNodeID)
				if n == nil {
					return sdk.WrapError(sdk.ErrWorkflowNotFound, "postWorkflowRunHandler> Unable to run workflow")
				}
			}
			opts.Manual.PipelineParameters = n.Context.DefaultPipelineParameters
		}

		log.Debug("Manual run: %#v", opts.Manual)

		//Manual run
		if lastRun != nil {
			if opts.FromNodeID == nil {
				opts.FromNodeID = &lastRun.Workflow.RootID
			}
			var errmr error
			wr, errmr = workflow.ManualRunFromNode(tx, wf, lastRun.Number, opts.Manual, *opts.FromNodeID)
			if errmr != nil {
				return sdk.WrapError(errmr, "postWorkflowRunHandler> Unable to run workflow")
			}
		} else {
			var errmr error
			wr, errmr = workflow.ManualRun(tx, wf, opts.Manual)
			if errmr != nil {
				return sdk.WrapError(errmr, "postWorkflowRunHandler> Unable to run workflow")
			}
		}
	}

	//Commit and return success
	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "postWorkflowRunHandler> Unable to run workflow")
	}

	wr.Translate(r.Header.Get("Accept-Language"))
	return WriteJSON(w, r, wr, http.StatusOK)
}

func getWorkflowNodeRunArtifactsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	name := vars["workflowName"]

	number, errNu := requestVarInt(r, "number")
	if errNu != nil {
		return sdk.WrapError(errNu, "getWorkflowJobArtifactsHandler> Invalid node job run ID")
	}

	id, errI := requestVarInt(r, "id")
	if errI != nil {
		return sdk.WrapError(sdk.ErrInvalidID, "getWorkflowJobArtifactsHandler> Invalid node job run ID")
	}
	nodeRun, errR := workflow.LoadNodeRun(db, key, name, number, id)
	if errR != nil {
		return sdk.WrapError(errR, "getWorkflowJobArtifactsHandler> Cannot load node run")
	}

	return WriteJSON(w, r, nodeRun.Artifacts, http.StatusOK)
}

func getDownloadArtifactHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	name := vars["workflowName"]

	id, errI := requestVarInt(r, "artifactId")
	if errI != nil {
		return sdk.WrapError(sdk.ErrInvalidID, "getDownloadArtifactHandler> Invalid node job run ID")
	}

	work, errW := workflow.Load(db, key, name, c.User)
	if errW != nil {
		return sdk.WrapError(errW, "getDownloadArtifactHandler> Cannot load workflow")
	}

	art, errA := workflow.LoadArtifactByIDs(db, work.ID, id)
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

func getWorkflowRunArtifactsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	name := vars["workflowName"]

	number, errNu := requestVarInt(r, "number")
	if errNu != nil {
		return sdk.WrapError(errNu, "getWorkflowJobArtifactsHandler> Invalid node job run ID")
	}

	wr, errW := workflow.LoadRun(db, key, name, number)
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

func getWorkflowNodeRunJobStepHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	workflowName := vars["workflowName"]
	number, errN := requestVarInt(r, "number")
	if errN != nil {
		return sdk.WrapError(errN, "getWorkflowNodeRunJobBuildLogsHandler> Number: invalid number")
	}
	nodeRunID, errNI := requestVarInt(r, "id")
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
	if _, errW := workflow.Load(db, projectKey, workflowName, c.User); errW != nil {
		return sdk.WrapError(errW, "getWorkflowNodeRunJobBuildLogsHandler> Cannot find workflow %s in project %s", workflowName, projectKey)
	}

	// Check nodeRunID is link to workflow
	nodeRun, errNR := workflow.LoadNodeRun(db, projectKey, workflowName, number, nodeRunID)
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
		return sdk.WrapError(fmt.Errorf("getWorkflowNodeRunJobBuildLogsHandler> Cannot find step %d on job %d in nodeRun %d/%d for workflow %s in project %s",
			stepOrder, runJobID, nodeRunID, number, workflowName, projectKey), "")
	}

	logs, errL := workflow.LoadStepLogs(db, runJobID, stepOrder)
	if errL != nil {
		return sdk.WrapError(errL, "getWorkflowNodeRunJobBuildLogsHandler> Cannot load log for runJob %d on step %d", runJobID, stepOrder)
	}

	result := &sdk.BuildState{
		Status:   sdk.StatusFromString(stepStatus),
		StepLogs: *logs,
	}

	return WriteJSON(w, r, result, http.StatusOK)
}
