package main

import (
	"net/http"
	"strconv"

	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

const (
	rangeMax     = 50
	defaultLimit = 10
)

func getWorkflowRunsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
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

func getLatestWorkflowRunHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	name := vars["workflowName"]
	run, err := workflow.LoadLastRun(db, key, name)
	if err != nil {
		return sdk.WrapError(err, "getLatestWorkflowRunHandler> Unable to load last workflow run")
	}
	return WriteJSON(w, r, run, http.StatusOK)
}

func getWorkflowRunHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
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

func getWorkflowNodeRunHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
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
	return WriteJSON(w, r, run, http.StatusOK)
}
