package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/context"
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

	offset, err := requestVarInt(r, "offset")
	limit, err := requestVarInt(r, "limit")

	return nil
}

func getLatestWorkflowRunHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

func getWorkflowRunHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}
