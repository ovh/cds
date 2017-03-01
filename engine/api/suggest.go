package main

import (
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getVariablesHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]
	appName := r.FormValue("appName")

	var allVariables []string

	// Load variable project
	projectVar, err := project.GetAllVariableNameInProjectByKey(db, projectKey)
	if err != nil {
		log.Warning("getVariablesHandler> Cannot Load project variables: %s\n", err)
		return err

	}
	for i := range projectVar {
		projectVar[i] = fmt.Sprintf("{{.cds.proj.%s}}", projectVar[i])
	}
	allVariables = append(allVariables, projectVar...)

	// Load env variable
	envVarNameArray, err := environment.GetAllVariableNameByProject(db, projectKey)
	if err != nil {
		log.Warning("getVariablesHandler> Cannot Load env variables: %s\n", err)
		return err

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
			log.Warning("getPipelineTypeHandler> Cannot Load application: %s\n", err)
			return err

		}

		if !permission.AccessToApplication(app.ID, c.User, permission.PermissionRead) {
			log.Warning("getVariablesHandler> Not allow to access to this application: %s\n", appName)
			return sdk.ErrForbidden

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
			log.Warning("getVariablesHandler> Cannot Load all applications variables: %s\n", err)
			return err

		}
		defer rows.Close()
		for rows.Next() {
			var name string
			err := rows.Scan(&name)
			if err != nil {
				log.Warning("getVariablesHandler> Cannot scan results: %s\n", err)
				return err

			}
			appVar = append(appVar, fmt.Sprintf("{{.cds.app.%s}}", name))

		}
	}
	allVariables = append(allVariables, appVar...)
	// add cds variable
	cdsVar := []string{
		"{{.cds.application}}",
		"{{.cds.buildNumber}}",
		"{{.cds.environment}}",
		"{{.cds.parent.application}}",
		"{{.cds.parent.buildNumber}}",
		"{{.cds.parent.pipeline}}",
		"{{.cds.pipeline}}",
		"{{.cds.project}}",
		"{{.cds.triggered_by.email}}",
		"{{.cds.triggered_by.fullname}}",
		"{{.cds.triggered_by.username}}",
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
