package sanity

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"text/template"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// Warning unique identifiers
const (
	_ = iota
	MultipleWorkerModelWarning
	NoWorkerModelMatchRequirement
	InvalidVariableFormat
	ProjectVariableDoesNotExist
	ApplicationVariableDoesNotExist
	EnvironmentVariableDoesNotExist
	CannotUseEnvironmentVariable
	MultipleHostnameRequirement
	IncompatibleBinaryAndModelRequirements
	IncompatibleServiceAndModelRequirements
	IncompatibleMemoryAndModelRequirements
	GitURLWithoutLinkedRepository
	GitURLWithoutKey
)

var messageAmericanEnglish = map[int64]string{
	MultipleWorkerModelWarning:              `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}} has multiple Worker Model as requirement. It will never start building.`,
	NoWorkerModelMatchRequirement:           `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}}: No worker model matches all required binaries`,
	InvalidVariableFormat:                   `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}}: Invalid variable format '{{index . "VarName"}}'`,
	ProjectVariableDoesNotExist:             `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}}: Project variable '{{index . "VarName"}}' used but doesn't exist`,
	ApplicationVariableDoesNotExist:         `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}}: Application variable '{{index . "VarName"}}' used but doesn't exist in application '{{index . "AppName"}}'`,
	EnvironmentVariableDoesNotExist:         `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}}: Environment variable {{index . "VarName"}} used but doesn't exist in all environments`,
	CannotUseEnvironmentVariable:            `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}}: Cannot use environment variable '{{index . "VarName"}} in a pipeline of type 'Build'`,
	MultipleHostnameRequirement:             `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}} has multiple Hostname requirements. It will never start building.`,
	IncompatibleBinaryAndModelRequirements:  `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}}: Model {{index . "ModelName"}} does not have the binary '{{index . "BinaryRequirement"}}' capability`,
	IncompatibleServiceAndModelRequirements: `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}}: Model {{index . "ModelName"}} cannot be linked to service '{{index . "ServiceRequirement"}}'`,
	IncompatibleMemoryAndModelRequirements:  `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}}: Model {{index . "ModelName"}} cannot handle memory requirement`,
	GitURLWithoutLinkedRepository:           `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}} is used but one one more applications are linked to any repository. Git clone will failed`,
	GitURLWithoutKey:                        `Action {{index . "ActionName"}}{{if index . "PipelineName"}} in pipeline {{index . "ProjectKey"}}/{{index . "PipelineName"}}{{end}} is used but no ssh key were found. Git clone will failed`,
}

func processWarning(w *sdk.Warning, acceptedlanguage string) error {
	var buffer bytes.Buffer
	// Find warning body matching accepted language
	tmplBody := messageAmericanEnglish[w.ID]

	// Execute template
	t := template.Must(template.New("warning").Parse(tmplBody))
	err := t.Execute(&buffer, w.MessageParam)
	if err != nil {
		return err
	}

	// Set message value
	w.Message = buffer.String()
	return nil
}

// LoadAllWarnings loads all warnings existing in CDS
func LoadAllWarnings(db gorp.SqlExecutor, al string) ([]sdk.Warning, error) {
	query := `
	SELECT distinct(warning.id), warning_id, warning.message_param, warning.project_id, warning.pip_id, warning.app_id, warning.env_id, warning.action_id,
	       project.name as projName, application.name as appName, pip.name as pipName, env.name as envName, action.name as actionName,
	       project.projectkey as projKey,
				 pipeline_action.pipeline_stage_id
	FROM warning
	LEFT JOIN action ON action.id = warning.action_id
	JOIN project ON project.id = warning.project_id
	LEFT JOIN pipeline_action ON pipeline_action.action_id = warning.action_id
	LEFT JOIN application ON application.id = warning.app_id
	LEFT JOIN pipeline as pip ON pip.id = warning.pip_id
	LEFT JOIN environment as env ON env.id = warning.env_id
	GROUP BY warning.id, warning_id, warning.project_id, warning.pip_id, warning.app_id, warning.env_id, warning.action_id,
	      projKey, pipeline_action.pipeline_stage_id, projName, appName, pipName, envName, actionName
	LIMIT 10000
	`

	var warnings []sdk.Warning
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var id int64
	for rows.Next() {
		var w sdk.Warning
		var appID, pipID, envID, actionID, stageID sql.NullInt64
		var appName, pipName, envName, actionName sql.NullString
		var messageParam string

		err := rows.Scan(&id, &w.ID, &messageParam,
			&w.Project.ID, &pipID, &appID, &envID, &actionID,
			&w.Project.Name, &appName, &pipName, &envName, &actionName,
			&w.Project.Key, &stageID,
		)
		if err != nil {
			return nil, err
		}

		if stageID.Valid {
			w.StageID = stageID.Int64
		}

		if appID.Valid && appName.Valid {
			w.Application.ID = appID.Int64
			w.Application.Name = appName.String
		}

		if pipID.Valid && pipName.Valid {
			w.Pipeline.ID = pipID.Int64
			w.Pipeline.Name = pipName.String
		}

		if envID.Valid && envName.Valid {
			w.Environment.ID = envID.Int64
			w.Environment.Name = envName.String
		}

		if actionID.Valid && actionName.Valid {
			w.Action.ID = actionID.Int64
			w.Action.Name = actionName.String
		}

		err = json.Unmarshal([]byte(messageParam), &w.MessageParam)
		if err != nil {
			return nil, err
		}

		err = processWarning(&w, al)
		if err != nil {
			return nil, err
		}

		warnings = append(warnings, w)
	}

	return warnings, nil
}

// LoadUserWarnings loads all warnings related to Jobs user has access to
func LoadUserWarnings(db gorp.SqlExecutor, al string, userID int64) ([]sdk.Warning, error) {
	query := `
	SELECT distinct(warning.id), warning_id, warning.message_param, warning.project_id, warning.pip_id, warning.app_id, warning.env_id, warning.action_id,
	       project.name as projName, application.name as appName, pip.name as pipName, env.name as envName, action.name as actionName,
	       project.projectkey as projKey,
				 pipeline_action.pipeline_stage_id,
	       max(project_group.role) as projectPerm,
	       max(application_group.role) as appPerm,
	       max(pipeline_group.role) as pipPerm,
	       max(environment_group.role) as envPerm
	FROM warning
	LEFT JOIN action ON action.id = warning.action_id
	JOIN project ON project.id = warning.project_id
	LEFT JOIN pipeline_action ON pipeline_action.action_id = warning.action_id
	LEFT JOIN application ON application.id = warning.app_id
	LEFT JOIN pipeline as pip ON pip.id = warning.pip_id
	LEFT JOIN environment as env ON env.id = warning.env_id
	JOIN project_group on  project_group.project_id = warning.project_id
	JOIN group_user guproj ON guproj.group_id = project_group.group_id AND guproj.user_id = $1
	LEFT JOIN application_group on application_group.application_id = warning.app_id
	LEFT JOIN group_user guapp ON guapp.group_id = application_group.group_id AND guapp.user_id = $1
	LEFT JOIN pipeline_group on pipeline_group.pipeline_id = warning.pip_id
	LEFT JOIN group_user gupip ON gupip.group_id = pipeline_group.group_id AND gupip.user_id = $1
	LEFT JOIN environment_group ON environment_group.environment_id = warning.env_id
	LEFT JOIN group_user guenv ON guenv.group_id = environment_group.group_id AND guenv.user_id = $1
	GROUP BY warning.id, warning_id, warning.project_id, warning.pip_id, warning.app_id, warning.env_id, warning.action_id,
	      projKey, pipeline_action.pipeline_stage_id, projName, appName, pipName, envName, actionName
	`

	var warnings []sdk.Warning
	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var id int64
	for rows.Next() {
		var w sdk.Warning
		var appID, pipID, envID, actionID, stageID sql.NullInt64
		var appName, pipName, envName, actionName sql.NullString
		var projPerm, appPerm, pipPerm, envPerm sql.NullInt64
		var messageParam string

		err := rows.Scan(&id, &w.ID, &messageParam,
			&w.Project.ID, &pipID, &appID, &envID, &actionID,
			&w.Project.Name, &appName, &pipName, &envName, &actionName,
			&w.Project.Key, &stageID,
			&projPerm, &appPerm, &pipPerm, &envPerm)
		if err != nil {
			return nil, err
		}

		if !projPerm.Valid {
			// User cannot see this warning
			continue
		}

		if stageID.Valid {
			w.StageID = stageID.Int64
		}

		if appID.Valid && appName.Valid {
			if !appPerm.Valid {
				continue
			}
			w.Application.ID = appID.Int64
			w.Application.Name = appName.String
		}

		if pipID.Valid && pipName.Valid {
			if !pipPerm.Valid {
				continue
			}
			w.Pipeline.ID = pipID.Int64
			w.Pipeline.Name = pipName.String
		}

		if envID.Valid && envName.Valid {
			if !envPerm.Valid {
				continue
			}
			w.Environment.ID = envID.Int64
			w.Environment.Name = envName.String
		}

		if actionID.Valid && actionName.Valid {
			w.Action.ID = actionID.Int64
			w.Action.Name = actionName.String
		}

		err = json.Unmarshal([]byte(messageParam), &w.MessageParam)
		if err != nil {
			return nil, err
		}

		err = processWarning(&w, al)
		if err != nil {
			return nil, err
		}

		warnings = append(warnings, w)
	}

	return warnings, nil
}

// InsertActionWarnings in database
func InsertActionWarnings(tx gorp.SqlExecutor, projectID, pipelineID int64, actionID int64, warnings []sdk.Warning) error {

	query := `DELETE FROM warning WHERE action_id = $1`
	_, err := tx.Exec(query, actionID)
	if err != nil {
		return err
	}

	query = `INSERT INTO warning (project_id, app_id, pip_id, action_id, warning_id, message_param) VALUES ($1, $2, $3, $4, $5, $6)`
	for _, w := range warnings {
		if w.Pipeline.ID == 0 {
			w.Pipeline.ID = pipelineID
		}
		if w.Action.ID == 0 {
			w.Action.ID = actionID
		}

		mParam, err := json.Marshal(w.MessageParam)
		if err != nil {
			return err
		}

		var appID sql.NullInt64
		if w.Application.ID > 0 {
			appID.Valid = true
			appID.Int64 = w.Application.ID
		}
		_, err = tx.Exec(query, projectID, appID, w.Pipeline.ID, w.Action.ID, w.ID, string(mParam))
		if err != nil {
			return err
		}
	}

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

	warnings = checkGitVariables(tx, gitvars, project, pip, a)
	warnings = append(warnings, w...)

	return warnings, nil
}
