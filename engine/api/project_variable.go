package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

func (api *API) postEncryptVariableHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		p, errp := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errp != nil {
			return sdk.WrapError(errp, "postEncryptVariableHandler> unable to load project")
		}

		variable := new(sdk.Variable)
		if err := UnmarshalBody(r, variable); err != nil {
			return sdk.WrapError(err, "postEncryptVariableHandler> unable to read body")
		}

		encryptedValue, erre := project.EncryptWithBuiltinKey(api.mustDB(), p.ID, variable.Name, variable.Value)
		if erre != nil {
			return sdk.WrapError(erre, "postEncryptVariableHandler> unable to encrypte content %s", variable.Name)
		}

		variable.Value = encryptedValue
		return WriteJSON(w, variable, http.StatusOK)
	}
}

func (api *API) getVariablesAuditInProjectnHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]

		audits, err := project.GetVariableAudit(api.mustDB(), key)
		if err != nil {
			return sdk.WrapError(err, "getVariablesAuditInProjectnHandler: Cannot get variable audit for project %s", key)

		}
		return WriteJSON(w, audits, http.StatusOK)
	}
}

func (api *API) getVariablesInProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		p, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithVariables)
		if err != nil {
			return sdk.WrapError(err, "deleteVariableFromProject: Cannot load %s", key)
		}

		return WriteJSON(w, p.Variable, http.StatusOK)
	}
}

func (api *API) deleteVariableFromProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]
		varName := vars["name"]

		p, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(err, "deleteVariableFromProject: Cannot load %s", key)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "deleteVariableFromProject: Cannot start transaction")
		}
		defer tx.Rollback()

		varToDelete, errV := project.GetVariableInProject(api.mustDB(), p.ID, varName)
		if errV != nil {
			return sdk.WrapError(errV, "deleteVariableFromProject> Cannot load variable %s", varName)
		}

		if err := project.DeleteVariable(tx, p, varToDelete, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "deleteVariableFromProject: Cannot delete %s", varName)
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectVariableLastModificationType); err != nil {
			return sdk.WrapError(err, "deleteVariableFromProject: Cannot update last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteVariableFromProject: Cannot commit transaction")
		}

		event.PublishDeleteProjectVariable(p, *varToDelete, getUser(ctx))

		return WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) updateVariableInProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]
		varName := vars["name"]

		var newVar sdk.Variable
		if err := UnmarshalBody(r, &newVar); err != nil {
			return err
		}
		if newVar.Name != varName {
			return sdk.ErrWrongRequest

		}

		p, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(err, "updateVariableInProject: Cannot load %s", key)

		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "updateVariableInProject: cannot start transaction")

		}
		defer tx.Rollback()

		previousVar, err := project.GetVariableByID(tx, p.ID, newVar.ID, project.WithClearPassword())
		if err := project.UpdateVariable(tx, p, &newVar, previousVar, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "updateVariableInProject: Cannot update variable %s in project %s", varName, p.Name)
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectVariableLastModificationType); err != nil {
			return sdk.WrapError(err, "updateVariableInProject: Cannot update last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateVariableInProject: cannot commit transaction")

		}

		event.PublishUpdateProjectVariable(p, newVar, *previousVar, getUser(ctx))

		return WriteJSON(w, newVar, http.StatusOK)
	}
}

func (api *API) addVariableInProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]
		varName := vars["name"]

		var newVar sdk.Variable
		if err := UnmarshalBody(r, &newVar); err != nil {
			return err
		}
		if newVar.Name != varName {
			return sdk.ErrWrongRequest

		}

		p, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(err, "AddVariableInProject: Cannot load %s", key)
		}

		varInProject, err := project.CheckVariableInProject(api.mustDB(), p.ID, varName)
		if err != nil {
			return sdk.WrapError(err, "AddVariableInProject: Cannot check if variable %s is already in the project %s", varName, p.Name)
		}

		if varInProject {
			return sdk.ErrVariableExists
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "addVariableInProjectHandler: cannot begin tx")
		}
		defer tx.Rollback()

		switch newVar.Type {
		case sdk.KeyVariable:
			err = project.AddKeyPair(tx, p, newVar.Name, getUser(ctx))
			break
		default:
			err = project.InsertVariable(tx, p, &newVar, getUser(ctx))
			break
		}
		if err != nil {
			return sdk.WrapError(err, "AddVariableInProject: Cannot add variable %s in project %s", varName, p.Name)

		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectVariableLastModificationType); err != nil {
			return sdk.WrapError(err, "updateVariablesInProjectHandler: Cannot update last modified")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addVariableInProjectHandler: cannot commit tx")
		}

		// Send Add variable event
		event.PublishAddProjectVariable(p, newVar, getUser(ctx))

		p.Variable, err = project.GetAllVariableInProject(api.mustDB(), p.ID)
		if err != nil {
			return sdk.WrapError(err, "AddVariableInProject: Cannot get variables")

		}

		return WriteJSON(w, p, http.StatusOK)
	}
}

func (api *API) getVariableAuditInProjectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]
		varName := vars["name"]

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "getVariableAuditInProjectHandler> Cannot load project %s", key)
		}

		variable, errV := project.GetVariableInProject(api.mustDB(), p.ID, varName)
		if errV != nil {
			return sdk.WrapError(errV, "getVariableAuditInProjectHandler> Cannot load variable %s", varName)
		}

		audits, errA := project.LoadVariableAudits(api.mustDB(), p.ID, variable.ID)
		if errA != nil {
			return sdk.WrapError(errA, "getVariableAuditInProjectHandler> Cannot load audit for variable %s", varName)
		}
		return WriteJSON(w, audits, http.StatusOK)
	}
}
