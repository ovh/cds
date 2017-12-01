package sanity

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"html/template"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/sdk"
)

func processWarning(w *sdk.Warning, acceptedlanguage string) error {
	var buffer bytes.Buffer
	// Find warning body matching accepted language
	tmplBody := messageAmericanEnglish[w.ID]

	// Execute template
	t := template.Must(template.New("warning").Parse(tmplBody))
	if err := t.Execute(&buffer, w.MessageParam); err != nil {
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

		if err := rows.Scan(&id, &w.ID, &messageParam,
			&w.Project.ID, &pipID, &appID, &envID, &actionID,
			&w.Project.Name, &appName, &pipName, &envName, &actionName,
			&w.Project.Key, &stageID,
		); err != nil {
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

		if err := json.Unmarshal([]byte(messageParam), &w.MessageParam); err != nil {
			return nil, err
		}

		if err := processWarning(&w, al); err != nil {
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
	JOIN project ON project.id = warning.project_id
    JOIN project_group on  project_group.project_id = warning.project_id
	JOIN group_user guproj ON guproj.group_id = project_group.group_id AND guproj.user_id = $1
    LEFT OUTER JOIN action ON action.id = warning.action_id
	LEFT OUTER JOIN pipeline_action ON pipeline_action.action_id = warning.action_id
	LEFT OUTER JOIN application ON application.id = warning.app_id
	LEFT OUTER JOIN pipeline as pip ON pip.id = warning.pip_id
	LEFT OUTER JOIN environment as env ON env.id = warning.env_id
	LEFT OUTER JOIN application_group on application_group.application_id = warning.app_id
	LEFT OUTER JOIN group_user guapp ON guapp.group_id = application_group.group_id AND guapp.user_id = $1
	LEFT OUTER JOIN pipeline_group on pipeline_group.pipeline_id = warning.pip_id
	LEFT OUTER JOIN group_user gupip ON gupip.group_id = pipeline_group.group_id AND gupip.user_id = $1
	LEFT OUTER JOIN environment_group ON environment_group.environment_id = warning.env_id
	LEFT OUTER JOIN group_user guenv ON guenv.group_id = environment_group.group_id AND guenv.user_id = $1
	WHERE project_group.role >= $2 AND application_group.role >= $2 AND pipeline_group.role >= $2
	GROUP BY warning.id, warning_id, warning.project_id, warning.pip_id, warning.app_id, warning.env_id, warning.action_id,
	      projKey, pipeline_action.pipeline_stage_id, projName, appName, pipName, envName, actionName;
	`

	var warnings []sdk.Warning
	rows, errq := db.Query(query, userID, permission.PermissionReadWriteExecute)
	if errq != nil {
		return nil, sdk.WrapError(errq, "LoadUserWarnings>")
	}
	defer rows.Close()

	var id int64
	for rows.Next() {
		var w sdk.Warning
		var appID, pipID, envID, actionID, stageID sql.NullInt64
		var appName, pipName, envName, actionName sql.NullString
		var projPerm, appPerm, pipPerm, envPerm sql.NullInt64
		var messageParam string

		if err := rows.Scan(&id, &w.ID, &messageParam,
			&w.Project.ID, &pipID, &appID, &envID, &actionID,
			&w.Project.Name, &appName, &pipName, &envName, &actionName,
			&w.Project.Key, &stageID,
			&projPerm, &appPerm, &pipPerm, &envPerm); err != nil {
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
			// no check for default env, no warning if permission on env is not RW
			if w.Environment.Name != sdk.DefaultEnv.Name && w.Environment.Permission < permission.PermissionReadWriteExecute {
				continue
			}
			w.Environment.ID = envID.Int64
			w.Environment.Name = envName.String
		}

		if actionID.Valid && actionName.Valid {
			w.Action.ID = actionID.Int64
			w.Action.Name = actionName.String
		}

		if err := json.Unmarshal([]byte(messageParam), &w.MessageParam); err != nil {
			return nil, err
		}

		if err := processWarning(&w, al); err != nil {
			return nil, err
		}

		warnings = append(warnings, w)
	}

	return warnings, nil
}

// InsertActionWarnings in database
func InsertActionWarnings(tx gorp.SqlExecutor, projectID, pipelineID int64, actionID int64, warnings []sdk.Warning) error {
	if _, err := tx.Exec(`DELETE FROM warning WHERE action_id = $1`, actionID); err != nil {
		return err
	}

	query := `INSERT INTO warning (project_id, app_id, pip_id, action_id, warning_id, message_param) VALUES ($1, $2, $3, $4, $5, $6)`
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

		if _, err = tx.Exec(query, projectID, appID, w.Pipeline.ID, w.Action.ID, w.ID, string(mParam)); err != nil {
			return sdk.WrapError(err, "InsertActionWarnings> Error with query: %s %d %d %d %d %d %s", query, projectID, appID, w.Pipeline.ID, w.Action.ID, w.ID, string(mParam))
		}
	}

	return nil
}

// DeleteAllApplicationWarnings deletes all warnings for application only (ie. not related to an action)
func DeleteAllApplicationWarnings(tx gorp.SqlExecutor, projectID, appID int64) error {
	if _, err := tx.Exec(`DELETE FROM warning WHERE app_id = $1 and action_id is null`, appID); err != nil {
		return err
	}
	return nil
}

// InsertApplicationWarning in database
func InsertApplicationWarning(tx gorp.SqlExecutor, projectID, appID int64, w *sdk.Warning) error {
	query := `INSERT INTO warning (project_id, app_id, warning_id, message_param) VALUES ($1, $2, $3, $4)`
	if w.Project.ID == 0 {
		w.Project.ID = projectID
	}
	if w.Application.ID == 0 {
		w.Application.ID = appID
	}

	mParam, errJSON := json.Marshal(w.MessageParam)
	if errJSON != nil {
		return errJSON
	}

	if _, err := tx.Exec(query, projectID, appID, w.ID, string(mParam)); err != nil {
		return err
	}

	return nil
}
