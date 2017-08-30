package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

func getVariablesHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	appName := r.FormValue("appName")
	pipID := r.FormValue("pipId")

	var allVariables []string

	// Load variable project
	projectVar, err := project.GetAllVariableNameInProjectByKey(db, projectKey)
	if err != nil {
		return sdk.WrapError(err, "getVariablesHandler> Cannot Load project variables: %s", err)
	}
	for i := range projectVar {
		projectVar[i] = fmt.Sprintf("{{.cds.proj.%s}}", projectVar[i])
	}
	allVariables = append(allVariables, projectVar...)

	// Load env variable
	envVarNameArray, err := environment.GetAllVariableNameByProject(db, projectKey)
	if err != nil {
		return sdk.WrapError(err, "getVariablesHandler> Cannot Load env variables: %s", err)
	}
	for i := range envVarNameArray {
		envVarNameArray[i] = fmt.Sprintf("{{.cds.env.%s}}", envVarNameArray[i])
	}
	allVariables = append(allVariables, envVarNameArray...)

	// Load app
	appVar := []string{}
	if appName != "" {
		// Check permission on application
		app, err := application.LoadByName(db, projectKey, appName, c.User, application.LoadOptions.WithVariables)
		if err != nil {
			return sdk.WrapError(err, "getPipelineTypeHandler> Cannot Load application: %s", err)
		}

		if !permission.AccessToApplication(app.ID, c.User, permission.PermissionRead) {
			return sdk.WrapError(sdk.ErrForbidden, "getVariablesHandler> Not allow to access to this application: %s", appName)
		}

		for _, v := range app.Variable {
			appVar = append(appVar, fmt.Sprintf("{{.cds.app.%s}}", v.Name))
		}

	} else {
		// Load all app variables
		query := `
			SELECT distinct var_name
			FROM application_variable
			LEFT JOIN application ON application.id = application_variable.application_id
			LEFT JOIN project ON project.id = application.project_id
			WHERE project.projectkey = $1
			ORDER BY var_name;
		`
		rows, err := db.Query(query, projectKey)
		if err != nil {
			return sdk.WrapError(err, "getVariablesHandler> Cannot Load all applications variables: %s", err)
		}
		defer rows.Close()
		for rows.Next() {
			var name string
			err := rows.Scan(&name)
			if err != nil {
				return sdk.WrapError(err, "getVariablesHandler> Cannot scan results: %s", err)
			}
			appVar = append(appVar, fmt.Sprintf("{{.cds.app.%s}}", name))

		}
	}
	allVariables = append(allVariables, appVar...)

	if pipID != "" {
		pipIDN, err := strconv.ParseInt(pipID, 10, 64)
		if err != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "getVariablesHandler> Cannot convert pipId to int : %s", err)
		}
		pipParams, err := pipeline.GetAllParametersInPipeline(db, pipIDN)

		if err != nil {
			return sdk.WrapError(err, "getVariablesHandler> Cannot get all parameters in pipeline: %s", err)
		}

		for _, param := range pipParams {
			allVariables = append(allVariables, fmt.Sprintf("{{.cds.pip.%s}}", param.Name))
		}
	}

	// add cds variable
	cdsVar := []string{
		"{{.cds.application}}",
		"{{.cds.buildNumber}}",
		"{{.cds.environment}}",
		"{{.cds.job}}",
		"{{.cds.parent.application}}",
		"{{.cds.parent.buildNumber}}",
		"{{.cds.parent.pipeline}}",
		"{{.cds.pipeline}}",
		"{{.cds.project}}",
		"{{.cds.stage}}",
		"{{.cds.triggered_by.email}}",
		"{{.cds.triggered_by.fullname}}",
		"{{.cds.triggered_by.username}}",
		"{{.cds.worker}}",
	}
	allVariables = append(allVariables, cdsVar...)

	// add git variable
	gitVar := []string{
		"{{.git.hash}}",
		"{{.git.branch}}",
		"{{.git.author}}",
		"{{.git.project}}",
		"{{.git.repository}}",
		"{{.git.url}}",
		"{{.git.http_url}}",
	}
	allVariables = append(allVariables, gitVar...)

	// Check permission on application
	return WriteJSON(w, r, allVariables, http.StatusOK)
}
