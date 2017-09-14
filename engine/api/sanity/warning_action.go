package sanity

import (
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// CheckAction checks for configuration errors
func CheckAction(tx gorp.SqlExecutor, store cache.Store, project *sdk.Project, pip *sdk.Pipeline, actionID int64) ([]sdk.Warning, error) {
	var warnings []sdk.Warning

	a, err := action.LoadActionByID(tx, actionID)
	if err != nil {
		return nil, err
	}

	for _, app := range project.Applications {
		app.Variable, err = application.GetAllVariable(tx, project.Key, app.Name)
		if err != nil {
			log.Warning("CheckAction> Unable to load application variable : %s", err)
			return nil, err
		}
	}

	// Load registered worker model
	wms, err := worker.LoadWorkerModels(tx)
	if err != nil {
		log.Warning("CheckAction> Cannot LoadWorkerModels")
		return nil, err
	}

	w, err := checkActionRequirements(a, project.Key, pip.Name, wms)
	if err != nil {
		return nil, err
	}
	warnings = append(warnings, w...)

	pvars, avars, evars, gitvars, badvars := loadUsedVariables(a)

	// Add warning for all badly formatted variables
	for _, v := range badvars {
		log.Warning("CheckAction> Badly formatted variable: '%s'\n", v)
		w := sdk.Warning{
			ID: InvalidVariableFormat,
			MessageParam: map[string]string{
				"VarName":      v,
				"ProjectKey":   project.Key,
				"ActionName":   a.Name,
				"PipelineName": pip.Name,
			},
		}
		w.Action.ID = a.ID
		warnings = append(warnings, w)
	}

	w, err = checkProjectVariables(tx, pvars, project, pip, a)
	if err != nil {
		return nil, fmt.Errorf("CheckAction> checkProjectVariables> %s", err)
	}
	warnings = append(warnings, w...)

	w, err = checkEnvironmentVariables(tx, evars, project, pip, a)
	if err != nil {
		return nil, fmt.Errorf("CheckAction> checkEnvironmentVariables> %s", err)
	}
	warnings = append(warnings, w...)

	w, err = checkApplicationVariables(tx, store, avars, project, pip, a)
	if err != nil {
		return nil, fmt.Errorf("CheckAction> checkApplicationVariables> %s", err)
	}
	warnings = append(warnings, w...)

	w = checkGitVariables(tx, store, gitvars, project, pip, a)
	warnings = append(warnings, w...)

	return warnings, nil
}
