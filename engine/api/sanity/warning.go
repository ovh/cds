package sanity

import (
	"fmt"
	"sync"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// CheckApplication checks all application variables
func CheckApplication(db *gorp.DbMap, project *sdk.Project, app *sdk.Application) error {
	return nil
}

func checkApplicationVariable(project *sdk.Project, app *sdk.Application, variable *sdk.Variable) []sdk.Warning {
	return nil
}

// CheckProjectPipelines checks all pipelines in project
func CheckProjectPipelines(db *gorp.DbMap, project *sdk.Project) error {

	// Load all pipelines
	pips, err := pipeline.LoadPipelines(db, project.ID, true, &sdk.User{Admin: true})
	if err != nil {
		return err
	}

	wg := &sync.WaitGroup{}
	for i := range pips {
		wg.Add(1)
		go func(p *sdk.Pipeline) {
			defer wg.Done()
			CheckPipeline(db, project, p)
		}(&pips[i])
	}

	wg.Wait()
	return nil
}

// CheckPipeline loads all PipelineAction and checks them all
func CheckPipeline(db *gorp.DbMap, project *sdk.Project, pip *sdk.Pipeline) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, s := range pip.Stages {
		for _, j := range s.Jobs {
			warnings, err := CheckAction(tx, project, pip, j.Action.ID)
			if err != nil {
				return err
			}
			err = InsertActionWarnings(tx, project.ID, pip.ID, j.Action.ID, warnings)
			if err != nil {
				return err
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return err
}

// CheckAction checks for configuration errors like:
// - incompatible requirements
// - inexisting variable usage
func CheckAction(tx gorp.SqlExecutor, project *sdk.Project, pip *sdk.Pipeline, actionID int64) ([]sdk.Warning, error) {
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

	w, err = checkApplicationVariables(tx, avars, project, pip, a)
	if err != nil {
		return nil, fmt.Errorf("CheckAction> checkApplicationVariables> %s", err)
	}
	warnings = append(warnings, w...)

	w = checkGitVariables(tx, gitvars, project, pip, a)
	warnings = append(warnings, w...)

	return warnings, nil
}
