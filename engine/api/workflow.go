package main

import (
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/context"
)

// getWorkflowsHandler returns ID and name of workflows for a given project/user
func getWorkflowsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

// getWorkflowHandler returns a full workflow
func getWorkflowHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

// postWorkflowHandler create a new workflow
func postWorkflowHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

// putWorkflowHandler updates a workflow
func putWorkflowHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

// putWorkflowHandler deletes a workflow
func deleteWorkflowHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

// postWorkflowNodeHandler creates a node in a workflow
func postWorkflowNodeHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

// putWorkflowNodeHandler updates a node in a workflow
func putWorkflowNodeHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

// deleteWorkflowNodeHandler deletes a node in a workflow and all children nodes
func deleteWorkflowNodeHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

// postWorkflowNodeHookHandler creates a node in a workflow
func postWorkflowNodeHookHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

// putWorkflowNodeHookHandler updates a node in a workflow
func putWorkflowNodeHookHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}

// deleteWorkflowNodeHookHandler deletes a node in a workflow and all children nodes
func deleteWorkflowNodeHookHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return nil
}
