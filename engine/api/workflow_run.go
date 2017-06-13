package main

import (
	"net/http"
	"strconv"

	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
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

	var run *sdk.WorkflowNodeRun
	var errNodeRun error

	subNumberS := r.Form.Get("subnumber")
	if subNumberS == "" {
		run, errNodeRun = workflow.LoadNodeRun(db, key, name, number, id)
	} else {
		sub, errS := strconv.ParseInt("subNumberS", 10, 64)
		if errS != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "getWorkflowRunHandler> subnumber is not an ID")
		}
		run, errNodeRun = workflow.LoadNodeRunBySub(db, key, name, number, id, sub)
	}

	if errNodeRun != nil {
		return sdk.WrapError(errNodeRun, "getWorkflowRunHandler> Unable to load last workflow run")
	}
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

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	opts := &postWorkflowRunHandlerOption{}
	if err := UnmarshalBody(r, opts); err != nil {
		return err
	}

	wf, err := workflow.Load(tx, key, name, c.User)
	if err != nil {
		return sdk.WrapError(err, "postWorkflowRunHandler> Unable to load workflow")
	}

	var lastRun *sdk.WorkflowRun
	if opts.Number != nil {
		lastRun, err = workflow.LoadRun(tx, key, name, *opts.Number)
		if err != nil {
			return sdk.WrapError(err, "postWorkflowRunHandler> Unable to load workflow run")
		}
	}

	var wr *sdk.WorkflowRun

	//Run from hook
	if opts.Hook != nil {
		wr, err = workflow.RunFromHook(tx, wf, opts.Hook)
		if err != nil {
			return sdk.WrapError(err, "postWorkflowRunHandler> Unable to run workflow")
		}
	} else {
		//Default manual run
		if opts.Manual == nil {
			opts.Manual = &sdk.WorkflowNodeRunManual{
				User: *c.User,
			}
		}

		//Manual run
		if lastRun != nil {
			if opts.FromNodeID == nil {
				opts.FromNodeID = &lastRun.Workflow.RootID
			}
			wr, err = workflow.ManualRunFromNode(tx, wf, lastRun.Number, opts.Manual, *opts.FromNodeID)
			if err != nil {
				return sdk.WrapError(err, "postWorkflowRunHandler> Unable to run workflow")
			}
		} else {
			wr, err = workflow.ManualRun(tx, wf, opts.Manual)
			if err != nil {
				return sdk.WrapError(err, "postWorkflowRunHandler> Unable to run workflow")
			}
		}
	}

	//Commit and return success
	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "postWorkflowRunHandler> Unable to run workflow")
	}
	return WriteJSON(w, r, wr, http.StatusOK)
}
