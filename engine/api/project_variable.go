package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) postEncryptVariableHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		p, errp := project.Load(api.mustDB(), api.Cache, key)
		if errp != nil {
			return sdk.WrapError(errp, "unable to load project")
		}

		variable := new(sdk.Variable)
		if err := service.UnmarshalBody(r, variable); err != nil {
			return sdk.WrapError(err, "unable to read body")
		}

		encryptedValue, erre := project.EncryptWithBuiltinKey(api.mustDB(), p.ID, variable.Name, variable.Value)
		if erre != nil {
			return sdk.WrapError(erre, "unable to encrypte content %s", variable.Name)
		}

		variable.Value = encryptedValue
		return service.WriteJSON(w, variable, http.StatusOK)
	}
}

func (api *API) getVariablesAuditInProjectnHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		audits, err := project.GetVariableAudit(api.mustDB(), key)
		if err != nil {
			return sdk.WrapError(err, "cannot get variable audit for project %s", key)

		}

		return service.WriteJSON(w, audits, http.StatusOK)
	}
}

func (api *API) getVariablesInProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		p, err := project.Load(api.mustDB(), api.Cache, key, project.LoadOptions.WithVariables)
		if err != nil {
			return sdk.WrapError(err, "cannot load %s", key)
		}

		return service.WriteJSON(w, p.Variables, http.StatusOK)
	}
}

func (api *API) deleteVariableFromProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		varName := vars["name"]

		p, err := project.Load(api.mustDB(), api.Cache, key, project.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(err, "deleteVariableFromProject: Cannot load %s", key)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "deleteVariableFromProject: Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		varToDelete, errV := project.LoadVariable(api.mustDB(), p.ID, varName)
		if errV != nil {
			return sdk.WrapError(errV, "deleteVariableFromProject> Cannot load variable %s", varName)
		}

		if err := project.DeleteVariable(tx, p.ID, varToDelete, getAPIConsumer(ctx)); err != nil {
			return sdk.WrapError(err, "deleteVariableFromProject: Cannot delete %s", varName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteVariableFromProject: Cannot commit transaction")
		}

		event.PublishDeleteProjectVariable(ctx, p, *varToDelete, getAPIConsumer(ctx))

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) updateVariableInProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		varName := vars["name"]

		var newVar sdk.Variable
		if err := service.UnmarshalBody(r, &newVar); err != nil {
			return err
		}

		p, err := project.Load(api.mustDB(), api.Cache, key, project.LoadOptions.Default)
		if err != nil {
			return sdk.WrapError(err, "updateVariableInProject: Cannot load %s", key)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "updateVariableInProject: cannot start transaction")

		}
		defer tx.Rollback() // nolint

		previousVar, err := project.LoadVariable(tx, p.ID, varName)
		if err != nil {
			return err
		}
		if err := project.UpdateVariable(tx, p.ID, &newVar, previousVar, getAPIConsumer(ctx)); err != nil {
			return sdk.WrapError(err, "updateVariableInProject: Cannot update variable %s in project %s", varName, p.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateVariableInProject: cannot commit transaction")
		}

		event.PublishUpdateProjectVariable(ctx, p, newVar, *previousVar, getAPIConsumer(ctx))

		return service.WriteJSON(w, newVar, http.StatusOK)
	}
}

func (api *API) addVariableInProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		varName := vars["name"]

		var newVar sdk.Variable
		if err := service.UnmarshalBody(r, &newVar); err != nil {
			return err
		}
		if newVar.Name != varName {
			return sdk.WithStack(sdk.ErrWrongRequest)

		}

		p, err := project.Load(api.mustDB(), api.Cache, key, project.LoadOptions.Default)
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "addVariableInProjectHandler: cannot begin tx")
		}
		defer tx.Rollback() // nolint

		if !sdk.IsInArray(newVar.Type, sdk.AvailableVariableType) {
			return sdk.WithStack(sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid variable type %s", newVar.Type))
		}

		if err := project.InsertVariable(tx, p.ID, &newVar, getAPIConsumer(ctx)); err != nil {
			return err

		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addVariableInProjectHandler: cannot commit tx")
		}

		// Send Add variable event
		event.PublishAddProjectVariable(ctx, p, newVar, getAPIConsumer(ctx))

		return service.WriteJSON(w, newVar, http.StatusOK)
	}
}

func (api *API) getVariableInProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		varName := vars["name"]

		p, err := project.Load(api.mustDB(), api.Cache, key, project.LoadOptions.WithVariables)
		if err != nil {
			return sdk.WrapError(err, "getVariableInProjectHandler: Cannot load %s", key)
		}

		variable, err := project.LoadVariable(api.mustDB(), p.ID, varName)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, variable, http.StatusOK)
	}
}

func (api *API) getVariableAuditInProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		varName := vars["name"]

		p, err := project.Load(api.mustDB(), api.Cache, key)
		if err != nil {
			return err
		}

		variable, err := project.LoadVariable(api.mustDB(), p.ID, varName)
		if err != nil {
			return err
		}

		audits, err := project.LoadVariableAudits(api.mustDB(), p.ID, variable.ID)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, audits, http.StatusOK)
	}
}
