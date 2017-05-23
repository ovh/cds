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

func getWorkflowRunsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	//La réponse de votre API sur une collection devra obligatoirement fournir dans les en-têtes HTTP :
	// Content-Range offset – limit / count
	//  offset : l’index du premier élément retourné par la requête.
	//  limit : l’index du dernier élément retourné par la requête.
	//  count : le nombre total d’élément que contient la collection.
	// Accept-Range resource max
	//  resource : le type de la pagination, on parlera ici systématiquement de la ressource en cours d’utilisation, ex : client, order, restaurant, …
	//  max : le nombre maximum pouvant être requêté en une seule fois.

	vars := mux.Vars(r)
	var limit, offset int

	offsetS, ok := vars["offset"]
	var errAtoi error
	if ok {
		offset, errAtoi = strconv.Atoi(offsetS)
		if errAtoi != nil {
			return sdk.ErrWrongRequest
		}
	}
	limitS, ok := vars["limit"]
	if ok {
		limit, errAtoi = strconv.Atoi(limitS)
		if errAtoi != nil {
			return sdk.ErrWrongRequest
		}
	}

	if offset < 0 {
		offset = 0
	}
	if limit == 0 {
		limit = 10
	}

	//Maximim range is set to 50
	w.Header().Add("Accept-Range", "run 50")
	if limit-offset > 50 {
		return sdk.WrapError(sdk.ErrWrongRequest, "getWorkflowRunsHandler> Requested range %d not allowed", (limit - offset))
	}

	key := vars["permProjectKey"]
	name := vars["workflowName"]
	runs, offset, limit, count, err := workflow.LoadRuns(db, key, name, offset, limit)
	if err != nil {
		return sdk.WrapError(err, "getWorkflowRunsHandler> Unable to load workflow runs")
	}

	w.Header().Add("Content-Range", fmt.Sprintf("%d-%d/%d", offset, limit, count))
	code := http.StatusOK

	if len(runs) < count {
		code = http.StatusPartialContent
	}

	//TODO implement  RFC5988: Link : <https://api.fakecompany.com/v1/orders?range=0-7>; rel="first", <https://api.fakecompany.com/v1/orders?range=40-47>; rel="prev", <https://api.fakecompany.com/v1/orders?range=56-64>; rel="next", <https://api.fakecompany.com/v1/orders?range=968-975>; rel="last"

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
