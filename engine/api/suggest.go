package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getVariablesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		pipID := r.FormValue("pipId")

		var allVariables []string

		proj, err := project.Load(ctx, api.mustDB(), projectKey, project.LoadOptions.WithVariables)
		if err != nil {
			return err
		}
		var projectVar = make([]string, len(proj.Variables))
		for i := range projectVar {
			projectVar[i] = fmt.Sprintf("{{.cds.proj.%s}}", proj.Variables[i].Name)
		}
		allVariables = append(allVariables, projectVar...)

		appVar := []string{}

		// Load all app variables
		query := `
			SELECT distinct var_name
			FROM application_variable
			LEFT JOIN application ON application.id = application_variable.application_id
			LEFT JOIN project ON project.id = application.project_id
			WHERE project.projectkey = $1
			ORDER BY var_name;
		`
		rows, err := api.mustDB().Query(query, projectKey)
		if err != nil {
			return sdk.WrapError(err, "Cannot Load all applications variables")
		}
		defer rows.Close()
		for rows.Next() {
			var name string
			err := rows.Scan(&name)
			if err != nil {
				return sdk.WrapError(err, "Cannot scan results")
			}
			appVar = append(appVar, fmt.Sprintf("{{.cds.app.%s}}", name))
		}
		allVariables = append(allVariables, appVar...)

		if pipID != "" {
			pipIDN, err := strconv.ParseInt(pipID, 10, 64)
			if err != nil {
				return sdk.WrapError(sdk.ErrWrongRequest, "getVariablesHandler> Cannot convert pipId to int : %s", err)
			}
			pipParams, err := pipeline.GetAllParametersInPipeline(ctx, api.mustDB(), pipIDN)

			if err != nil {
				return sdk.WrapError(err, "Cannot get all parameters in pipeline")
			}

			for _, param := range pipParams {
				allVariables = append(allVariables, fmt.Sprintf("{{.cds.pip.%s}}", param.Name))
			}
		}

		// add cds variable
		for i := range sdk.BasicVariableNames {
			allVariables = append(allVariables, "{{."+sdk.BasicVariableNames[i]+"}}")
		}

		// add git variable
		for i := range sdk.BasicGitVariableNames {
			allVariables = append(allVariables, "{{."+sdk.BasicGitVariableNames[i]+"}}")
		}
		allVariables = append(allVariables, "{{.git.tag}}")

		// Check permission on application
		return service.WriteJSON(w, allVariables, http.StatusOK)
	}
}
