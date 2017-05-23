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

	if limit == 0 {
		limit = 10
	}

	key := vars["permProjectKey"]
	name := vars["workflowName"]

	runs, offset, limit, count, err := workflow.LoadRuns(db, key, name, offset, limit)

	w.Header().Add("Content-Range", fmt.Sprintf("%d-%d/%d", offset, limit, count))

	return nil
}

func getLatestWorkflowRunHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

func getWorkflowRunHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}
