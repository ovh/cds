package trigger

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// InsertTriggerParameter insert given parameter in database
func InsertTriggerParameter(db gorp.SqlExecutor, triggerID int64, p sdk.Parameter) error {
	if string(p.Type) == string(sdk.SecretVariable) {
		return sdk.WrapError(sdk.ErrNoDirectSecretUse, "InsertTriggerParameter>")
	}

	query := `INSERT INTO pipeline_trigger_parameter (pipeline_trigger_id, name, type, value, description) VALUES ($1, $2, $3, $4, $5)`
	_, err := db.Exec(query, triggerID, p.Name, string(p.Type), p.Value, p.Description)
	return err
}

// InsertTrigger adds a new trigger in database
func InsertTrigger(tx gorp.SqlExecutor, t *sdk.PipelineTrigger) error {
	query := `INSERT INTO pipeline_trigger (src_application_id, src_pipeline_id, src_environment_id,
	dest_application_id, dest_pipeline_id, dest_environment_id, manual) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`

	var srcEnvID sql.NullInt64
	if t.SrcEnvironment.ID != 0 {
		srcEnvID.Valid = true
		srcEnvID.Int64 = t.SrcEnvironment.ID
	}

	var dstEnvID sql.NullInt64
	if t.DestEnvironment.ID != 0 {
		dstEnvID.Valid = true
		dstEnvID.Int64 = t.DestEnvironment.ID
	}

	// Check we are not creating an infinite loop first
	err := isTriggerLoopFree(tx, t, []parent{parent{AppID: t.SrcApplication.ID, PipID: t.SrcPipeline.ID, EnvID: t.SrcEnvironment.ID}})
	if err != nil {
		log.Warning("InsertTrigger: Infinite trigger loop found for trigger %s(%d)/%s(%d)/%s(%d)[%s(%d)] %s(%d)/%s(%d)/%s(%d)[%s(%d)]\n",
			t.SrcProject.Name, t.SrcProject.ID,
			t.SrcApplication.Name, t.SrcApplication.ID,
			t.SrcPipeline.Name, t.SrcPipeline.ID,
			t.SrcEnvironment.Name, t.SrcEnvironment.ID,
			t.DestProject.Name, t.DestProject.ID,
			t.DestApplication.Name, t.DestApplication.ID,
			t.DestPipeline.Name, t.DestPipeline.ID,
			t.DestEnvironment.Name, t.DestEnvironment.ID,
		)
		return err
	}

	// Insert trigger
	err = tx.QueryRow(query, t.SrcApplication.ID, t.SrcPipeline.ID, srcEnvID,
		t.DestApplication.ID, t.DestPipeline.ID, dstEnvID, t.Manual).Scan(&t.ID)
	if err != nil {
		return err
	}

	// Insert parameters
	for _, p := range t.Parameters {
		err = InsertTriggerParameter(tx, t.ID, p)
		if err != nil {
			return err
		}
	}

	// Insert prerequisites
	for _, p := range t.Prerequisites {
		err := InsertTriggerPrerequisite(tx, t.ID, p.Parameter, p.ExpectedValue)
		if err != nil {
			return err
		}

	}

	return nil
}

type parent struct {
	AppID int64
	PipID int64
	EnvID int64
}

func isTriggerLoopFree(tx gorp.SqlExecutor, t *sdk.PipelineTrigger, parents []parent) error {
	// First, check yourself
	for _, p := range parents {
		if t.DestApplication.ID == p.AppID &&
			t.DestPipeline.ID == p.PipID &&
			t.DestEnvironment.ID == p.EnvID {
			log.Warning("isTriggerLoopFree: Infinite trigger loop found !\n")
			log.Warning("isTriggerLoopFree: Infinite trigger loop found for trigger %s(%d)/%s(%d)[%s(%d)] (%d)/(%d)[(%d)]\n",
				t.DestApplication.Name, t.DestApplication.ID,
				t.DestPipeline.Name, t.DestPipeline.ID,
				t.DestEnvironment.Name, t.DestEnvironment.ID,
				p.AppID,
				p.PipID,
				p.EnvID)
			return sdk.ErrInfiniteTriggerLoop
		}
	}

	// Load all dest trigger
	tr, err := LoadTriggersAsSource(tx, t.DestApplication.ID, t.DestPipeline.ID, t.DestEnvironment.ID)
	if err != nil {
		return sdk.WrapError(err, "isTriggerLoopFree: cannot load trigger as source")
	}

	// Add yourself to parent
	parents = append(parents, parent{
		AppID: t.DestApplication.ID,
		PipID: t.DestPipeline.ID,
		EnvID: t.DestEnvironment.ID,
	})

	// Check all children
	for _, c := range tr {

		cerr := isTriggerLoopFree(tx, &c, parents)
		if cerr != nil {
			return cerr
		}
	}

	return nil
}

// InsertTriggerPrerequisite  Insert the given prerequisite
func InsertTriggerPrerequisite(db gorp.SqlExecutor, triggerID int64, paramName, value string) error {
	query := `INSERT INTO pipeline_trigger_prerequisite (pipeline_trigger_id, parameter, expected_value) VALUES ($1, $2, $3)`
	_, err := db.Exec(query, triggerID, paramName, value)
	return err
}

// UpdateTrigger update trigger data
func UpdateTrigger(db gorp.SqlExecutor, t *sdk.PipelineTrigger) error {
	var srcEnvID sql.NullInt64
	if t.SrcEnvironment.ID != 0 {
		srcEnvID.Valid = true
		srcEnvID.Int64 = t.SrcEnvironment.ID
	}

	var destEnvID sql.NullInt64
	if t.DestEnvironment.ID != 0 {
		destEnvID.Valid = true
		destEnvID.Int64 = t.DestEnvironment.ID
	}

	// Check we are not creating an infinite loop first
	if err := isTriggerLoopFree(db, t, []parent{parent{AppID: t.SrcApplication.ID, PipID: t.SrcPipeline.ID, EnvID: t.SrcEnvironment.ID}}); err != nil {
		log.Warning("UpdateTrigger: Infinite trigger loop found for trigger %s/%s/%s[%s] %s/%s/%s[%s]\n",
			t.SrcProject.Name, t.SrcApplication.Name, t.SrcPipeline.Name, t.SrcEnvironment.Name, t.DestProject.Name, t.DestApplication.Name, t.DestPipeline.Name, t.DestEnvironment.Name)
		return err
	}

	// Update trigger
	query := `UPDATE pipeline_trigger SET
	src_application_id = $1, src_pipeline_id = $2, src_environment_id = $3,
	dest_application_id = $4, dest_pipeline_id = $5, dest_environment_id = $6,
	manual = $7
	WHERE id = $8`
	if _, err := db.Exec(query, t.SrcApplication.ID, t.SrcPipeline.ID, srcEnvID, t.DestApplication.ID, t.DestPipeline.ID, destEnvID, t.Manual, t.ID); err != nil {
		return err
	}

	// Update parameters
	query = `DELETE FROM pipeline_trigger_parameter WHERE pipeline_trigger_id = $1`
	if _, err := db.Exec(query, t.ID); err != nil {
		return err
	}
	for _, p := range t.Parameters {
		if err := InsertTriggerParameter(db, t.ID, p); err != nil {
			return err
		}
	}

	// Update prerequisite
	query = `DELETE FROM pipeline_trigger_prerequisite WHERE pipeline_trigger_id = $1`
	if _, err := db.Exec(query, t.ID); err != nil {
		return err
	}
	query = `INSERT INTO pipeline_trigger_prerequisite (pipeline_trigger_id, parameter, expected_value) VALUES ($1, $2, $3)`
	for _, p := range t.Prerequisites {
		if _, err := db.Exec(query, t.ID, p.Parameter, p.ExpectedValue); err != nil {
			return err
		}
	}

	return nil
}

// LoadTriggersAsSource will only retrieves from database triggers where given pipeline is the source
func LoadTriggersAsSource(db gorp.SqlExecutor, appID, pipelineID, envID int64) ([]sdk.PipelineTrigger, error) {
	query := `
	SELECT pipeline_trigger.id,
	src_application_id, src_app.name,
	src_pipeline_id, src_pip.name, src_pip.type,
	src_environment_id, src_env.name,
	src_project.id, src_project.projectkey, src_project.name,
	dest_application_id, dest_app.name,
	dest_pipeline_id, dest_pip.name, dest_pip.type,
	dest_environment_id, dest_env.name,
	dest_project.id, dest_project.projectkey, dest_project.name,
	manual
	FROM pipeline_trigger
	JOIN pipeline as src_pip ON src_pip.id = src_pipeline_id
	JOIN application AS src_app ON src_app.id = src_application_id
	JOIN project AS src_project ON src_project.id = src_app.project_id
	JOIN pipeline as dest_pip ON dest_pip.id = dest_pipeline_id
	JOIN application AS dest_app ON dest_app.id = dest_application_id
	JOIN project AS dest_project ON dest_project.id = dest_app.project_id
	LEFT JOIN environment AS src_env ON src_env.id = src_environment_id
	LEFT JOIN environment AS dest_env ON dest_env.id = dest_environment_id
	WHERE %s
	ORDER by pipeline_trigger.id
	`
	var rows *sql.Rows
	var err error
	if envID > 1 {
		queryClause := "src_application_id = $1 AND src_pipeline_id = $2 AND src_environment_id = $3"
		query = fmt.Sprintf(query, queryClause)
		rows, err = db.Query(query, appID, pipelineID, envID)
	} else {
		queryClause := "src_application_id = $1 AND src_pipeline_id = $2"
		query = fmt.Sprintf(query, queryClause)
		rows, err = db.Query(query, appID, pipelineID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	triggers := []sdk.PipelineTrigger{}
	for rows.Next() {
		t, err := loadTrigger(db, rows, false)
		if err != nil {
			rows.Close()
			return nil, err
		}
		triggers = append(triggers, t)
	}
	rows.Close()

	for i := range triggers {
		triggers[i].Parameters, err = loadTriggerParameters(db, triggers[i].ID)
		if err != nil {
			return nil, err
		}

		triggers[i].Prerequisites, err = loadTriggerPrerequisites(db, triggers[i].ID)
		if err != nil {
			return nil, err
		}
	}
	return triggers, nil
}

// LoadAutomaticTriggersAsSource will only retrieves from database triggers where given pipeline is the source
//func LoadAutomaticTriggersAsSource(db gorp.SqlExecutor, appID, pipelineID, envID int64, mods ...mod) ([]sdk.PipelineTrigger, error) {
func LoadAutomaticTriggersAsSource(db gorp.SqlExecutor, appID, pipelineID, envID int64) ([]sdk.PipelineTrigger, error) {
	query := `
	SELECT pipeline_trigger.id,
	src_application_id, src_app.name,
	src_pipeline_id, src_pip.name, src_pip.type,
	src_environment_id, src_env.name,
	src_project.id, src_project.projectkey, src_project.name,
	dest_application_id, dest_app.name,
	dest_pipeline_id, dest_pip.name, dest_pip.type,
	dest_environment_id, dest_env.name,
	dest_project.id, dest_project.projectkey, dest_project.name,
	manual
	FROM pipeline_trigger
	JOIN pipeline as src_pip ON src_pip.id = src_pipeline_id
	JOIN application AS src_app ON src_app.id = src_application_id
	JOIN project AS src_project ON src_project.id = src_app.project_id
	JOIN pipeline as dest_pip ON dest_pip.id = dest_pipeline_id
	JOIN application AS dest_app ON dest_app.id = dest_application_id
	JOIN project AS dest_project ON dest_project.id = dest_app.project_id
	LEFT JOIN environment AS src_env ON src_env.id = src_environment_id
	LEFT JOIN environment AS dest_env ON dest_env.id = dest_environment_id
	WHERE pipeline_trigger.manual = false AND %s
	FOR UPDATE OF pipeline_trigger NOWAIT
	`
	var rows *sql.Rows
	var err error
	if envID > 1 {
		queryClause := "src_application_id = $1 AND src_pipeline_id = $2 AND src_environment_id = $3"
		query = fmt.Sprintf(query, queryClause)
		rows, err = db.Query(query, appID, pipelineID, envID)
	} else {
		queryClause := "src_application_id = $1 AND src_pipeline_id = $2 AND (src_environment_id = 1 OR src_environment_id IS NULL)"
		query = fmt.Sprintf(query, queryClause)
		rows, err = db.Query(query, appID, pipelineID)
	}
	if err != nil {
		return nil, err
	}
	triggers := []sdk.PipelineTrigger{}
	for rows.Next() {
		t, err := loadTrigger(db, rows, false)
		if err != nil {
			rows.Close()
			return nil, err
		}
		triggers = append(triggers, t)
	}
	rows.Close()

	for i := range triggers {
		triggers[i].Parameters, err = loadTriggerParameters(db, triggers[i].ID)
		if err != nil {
			return nil, err
		}

		triggers[i].Prerequisites, err = loadTriggerPrerequisites(db, triggers[i].ID)
		if err != nil {
			return nil, err
		}
	}

	return triggers, nil
}

// LoadTriggersByAppAndPipeline Load triggers for the given app and pipeline
func LoadTriggersByAppAndPipeline(db gorp.SqlExecutor, appID int64, pipID int64) ([]sdk.PipelineTrigger, error) {
	query := `
	SELECT pipeline_trigger.id,
	src_application_id, src_app.name,
	src_pipeline_id, src_pip.name, src_pip.type,
	src_environment_id, src_env.name,
	src_project.id, src_project.projectkey, src_project.name,
	dest_application_id, dest_app.name,
	dest_pipeline_id, dest_pip.name, dest_pip.type,
	dest_environment_id, dest_env.name,
	dest_project.id, dest_project.projectkey, dest_project.name,
	manual
	FROM pipeline_trigger
	JOIN pipeline as src_pip ON src_pip.id = src_pipeline_id
	JOIN application AS src_app ON src_app.id = src_application_id
	JOIN project AS src_project ON src_project.id = src_app.project_id
	JOIN pipeline as dest_pip ON dest_pip.id = dest_pipeline_id
	JOIN application AS dest_app ON dest_app.id = dest_application_id
	JOIN project AS dest_project ON dest_project.id = dest_app.project_id
	LEFT JOIN environment AS src_env ON src_env.id = src_environment_id
	LEFT JOIN environment AS dest_env ON dest_env.id = dest_environment_id
	WHERE (src_application_id = $1 AND src_pipeline_id = $2) OR (dest_application_id = $1 AND dest_pipeline_id = $2)
	`
	rows, err := db.Query(query, appID, pipID)
	if err != nil {
		return nil, err
	}
	triggers := []sdk.PipelineTrigger{}
	for rows.Next() {
		t, err := loadTrigger(db, rows, false)
		if err != nil {
			rows.Close()
			return nil, err
		}
		triggers = append(triggers, t)
	}
	rows.Close()

	for i := range triggers {
		triggers[i].Parameters, err = loadTriggerParameters(db, triggers[i].ID)
		if err != nil {
			return nil, err
		}

		triggers[i].Prerequisites, err = loadTriggerPrerequisites(db, triggers[i].ID)
		if err != nil {
			return nil, err
		}
	}

	return triggers, nil
}

// LoadTriggerByApp Load trigger where given app is source
func LoadTriggerByApp(db gorp.SqlExecutor, appID int64) ([]sdk.PipelineTrigger, error) {
	query := `
	SELECT pipeline_trigger.id,
	src_application_id, src_app.name,
	src_pipeline_id, src_pip.name, src_pip.type,
	src_environment_id, src_env.name,
	src_project.id, src_project.projectkey, src_project.name,
	dest_application_id, dest_app.name,
	dest_pipeline_id, dest_pip.name, dest_pip.type,
	dest_environment_id, dest_env.name,
	dest_project.id, dest_project.projectkey, dest_project.name,
	manual
	FROM pipeline_trigger
	JOIN pipeline as src_pip ON src_pip.id = src_pipeline_id
	JOIN application AS src_app ON src_app.id = src_application_id
	JOIN project AS src_project ON src_project.id = src_app.project_id
	JOIN pipeline as dest_pip ON dest_pip.id = dest_pipeline_id
	JOIN application AS dest_app ON dest_app.id = dest_application_id
	JOIN project AS dest_project ON dest_project.id = dest_app.project_id
	LEFT JOIN environment AS src_env ON src_env.id = src_environment_id
	LEFT JOIN environment AS dest_env ON dest_env.id = dest_environment_id
	WHERE src_application_id = $1
	`
	rows, err := db.Query(query, appID)
	if err != nil {
		return nil, err
	}
	triggers := []sdk.PipelineTrigger{}
	for rows.Next() {
		t, err := loadTrigger(db, rows, false)
		if err != nil {
			rows.Close()
			return nil, err
		}
		triggers = append(triggers, t)
	}
	rows.Close()

	for i := range triggers {
		triggers[i].Parameters, err = loadTriggerParameters(db, triggers[i].ID)
		if err != nil {
			return nil, err
		}

		triggers[i].Prerequisites, err = loadTriggerPrerequisites(db, triggers[i].ID)
		if err != nil {
			return nil, err
		}
	}

	return triggers, nil
}

// LoadTrigger load the given trigger
func LoadTrigger(db gorp.SqlExecutor, triggerID int64) (*sdk.PipelineTrigger, error) {
	query := `
	SELECT pipeline_trigger.id,
	src_application_id, src_app.name,
	src_pipeline_id, src_pip.name, src_pip.type,
	src_environment_id, src_env.name,
	src_project.id, src_project.projectkey, src_project.name,
	dest_application_id, dest_app.name,
	dest_pipeline_id, dest_pip.name, dest_pip.type,
	dest_environment_id, dest_env.name,
	dest_project.id, dest_project.projectkey, dest_project.name,
	manual
	FROM pipeline_trigger
	JOIN pipeline as src_pip ON src_pip.id = src_pipeline_id
	JOIN application AS src_app ON src_app.id = src_application_id
	JOIN project AS src_project ON src_project.id = src_app.project_id
	JOIN pipeline as dest_pip ON dest_pip.id = dest_pipeline_id
	JOIN application AS dest_app ON dest_app.id = dest_application_id
	JOIN project AS dest_project ON dest_project.id = dest_app.project_id
	LEFT JOIN environment AS src_env ON src_env.id = src_environment_id
	LEFT JOIN environment AS dest_env ON dest_env.id = dest_environment_id
	WHERE pipeline_trigger.id = $1
	`

	rows, err := db.Query(query, triggerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	triggers := []sdk.PipelineTrigger{}
	for rows.Next() {
		t, err := loadTrigger(db, rows, true)
		if err != nil {
			return nil, err
		}
		triggers = append(triggers, t)
	}

	t := triggers[0]
	t.Parameters, err = loadTriggerParameters(db, triggerID)
	if err != nil {
		return nil, err
	}

	return &t, nil

}

// LoadTriggers loads all triggers from database where given pipeline-env tuple is either triggering or triggered
//func LoadTriggers(db gorp.SqlExecutor, appID, pipelineID, envID int64, mods ...mod) ([]sdk.PipelineTrigger, error) {
func LoadTriggers(db gorp.SqlExecutor, appID, pipelineID, envID int64) ([]sdk.PipelineTrigger, error) {
	query := `
	SELECT pipeline_trigger.id,
	src_application_id, src_app.name,
	src_pipeline_id, src_pip.name, src_pip.type,
	src_environment_id, src_env.name,
	src_project.id, src_project.projectkey, src_project.name,
	dest_application_id, dest_app.name,
	dest_pipeline_id, dest_pip.name, dest_pip.type,
	dest_environment_id, dest_env.name,
	dest_project.id, dest_project.projectkey, dest_project.name,
	manual
	FROM pipeline_trigger
	JOIN pipeline as src_pip ON src_pip.id = src_pipeline_id
	JOIN application AS src_app ON src_app.id = src_application_id
	JOIN project AS src_project ON src_project.id = src_app.project_id
	JOIN pipeline as dest_pip ON dest_pip.id = dest_pipeline_id
	JOIN application AS dest_app ON dest_app.id = dest_application_id
	JOIN project AS dest_project ON dest_project.id = dest_app.project_id
	LEFT JOIN environment AS src_env ON src_env.id = src_environment_id
	LEFT JOIN environment AS dest_env ON dest_env.id = dest_environment_id
	WHERE %s
	`

	var rows *sql.Rows
	var err error
	if envID > 0 {
		queryClause := "(src_application_id = $1 AND src_pipeline_id = $2 AND src_environment_id = $3) OR (dest_application_id = $1 AND dest_pipeline_id = $2 AND dest_environment_id = $3)"
		query = fmt.Sprintf(query, queryClause)
		rows, err = db.Query(query, appID, pipelineID, envID)
	} else {
		queryClause := "(src_application_id = $1 AND src_pipeline_id = $2) OR (dest_application_id = $1 AND dest_pipeline_id = $2)"
		query = fmt.Sprintf(query, queryClause)
		rows, err = db.Query(query, appID, pipelineID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	triggers := []sdk.PipelineTrigger{}
	for rows.Next() {
		t, err := loadTrigger(db, rows, true)
		if err != nil {
			return nil, err
		}
		triggers = append(triggers, t)
	}

	return triggers, nil
}

func loadTrigger(db gorp.SqlExecutor, s *sql.Rows, subqueries bool) (sdk.PipelineTrigger, error) {
	var t sdk.PipelineTrigger
	var srcEnvName, destEnvName sql.NullString
	var srcEnvID, destEnvID sql.NullInt64
	err := s.Scan(&t.ID,
		&t.SrcApplication.ID, &t.SrcApplication.Name,
		&t.SrcPipeline.ID, &t.SrcPipeline.Name, &t.SrcPipeline.Type,
		&srcEnvID, &srcEnvName,
		&t.SrcProject.ID, &t.SrcProject.Key, &t.SrcProject.Name,
		&t.DestApplication.ID, &t.DestApplication.Name,
		&t.DestPipeline.ID, &t.DestPipeline.Name, &t.DestPipeline.Type,
		&destEnvID, &destEnvName,
		&t.DestProject.ID, &t.DestProject.Key, &t.DestProject.Name,
		&t.Manual,
	)
	if err != nil {
		return t, err
	}

	// Handle nullable envirnoments
	if destEnvName.Valid {
		t.DestEnvironment.Name = destEnvName.String
	}
	if destEnvID.Valid {
		t.DestEnvironment.ID = destEnvID.Int64
	} else {
		t.DestEnvironment = sdk.DefaultEnv
	}
	if srcEnvName.Valid {
		t.SrcEnvironment.Name = srcEnvName.String
	}
	if srcEnvID.Valid {
		t.SrcEnvironment.ID = srcEnvID.Int64
	} else {
		t.SrcEnvironment = sdk.DefaultEnv
	}

	if !subqueries {
		return t, nil
	}

	t.Parameters, err = loadTriggerParameters(db, t.ID)
	if err != nil {
		return t, err
	}

	t.Prerequisites, err = loadTriggerPrerequisites(db, t.ID)
	if err != nil {
		return t, err
	}

	return t, nil
}

func loadTriggerPrerequisites(db gorp.SqlExecutor, triggerID int64) ([]sdk.Prerequisite, error) {
	query := `SELECT parameter, expected_value FROM pipeline_trigger_prerequisite WHERE pipeline_trigger_id = $1`

	rows, err := db.Query(query, triggerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prereq []sdk.Prerequisite
	for rows.Next() {
		var p sdk.Prerequisite
		err = rows.Scan(&p.Parameter, &p.ExpectedValue)
		if err != nil {
			return nil, err
		}
		prereq = append(prereq, p)
	}

	return prereq, nil
}

func loadTriggerParameters(db gorp.SqlExecutor, triggerID int64) ([]sdk.Parameter, error) {
	query := `SELECT name, type, value, description FROM pipeline_trigger_parameter WHERE pipeline_trigger_id = $1`

	rows, err := db.Query(query, triggerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var params []sdk.Parameter
	for rows.Next() {
		var p sdk.Parameter
		var pType, val string
		err = rows.Scan(&p.Name, &pType, &val, &p.Description)
		if err != nil {
			return nil, err
		}
		p.Type = pType
		p.Value = val

		params = append(params, p)
	}

	return params, nil
}

// DeleteApplicationPipelineTriggers removes from database all triggers linked to a pipeline in a specific app
func DeleteApplicationPipelineTriggers(db gorp.SqlExecutor, proj, app, pip string) error {
	// Delete parameters
	query := `DELETE FROM pipeline_trigger_parameter WHERE pipeline_trigger_id IN (
			SELECT pipeline_trigger.id FROM pipeline_trigger
			JOIN pipeline AS src_pip ON src_pip.id = src_pipeline_id
			JOIN application AS src_app ON src_app.id = src_application_id
			JOIN project AS src_proj ON src_proj.id = src_app.project_id
			JOIN pipeline AS dest_pip ON dest_pip.id = dest_pipeline_id
			JOIN application AS dest_app ON dest_app.id = dest_application_id
			JOIN project AS dest_proj ON dest_proj.id = dest_app.project_id
			WHERE (src_pip.name = $1 AND src_app.name = $2 AND src_proj.projectkey = $3)
			OR (dest_pip.name = $1 AND dest_app.name = $2 AND dest_proj.projectkey = $3)
	)`
	_, err := db.Exec(query, pip, app, proj)
	if err != nil {
		return err
	}

	// Delete prerequisites
	query = `DELETE FROM pipeline_trigger_prerequisite WHERE pipeline_trigger_id IN (
			SELECT pipeline_trigger.id FROM pipeline_trigger
			JOIN pipeline AS src_pip ON src_pip.id = src_pipeline_id
			JOIN application AS src_app ON src_app.id = src_application_id
			JOIN project AS src_proj ON src_proj.id = src_app.project_id
			JOIN pipeline AS dest_pip ON dest_pip.id = dest_pipeline_id
			JOIN application AS dest_app ON dest_app.id = dest_application_id
			JOIN project AS dest_proj ON dest_proj.id = dest_app.project_id
			WHERE (src_pip.name = $1 AND src_app.name = $2 AND src_proj.projectkey = $3)
			OR (dest_pip.name = $1 AND dest_app.name = $2 AND dest_proj.projectkey = $3)
	)`
	_, err = db.Exec(query, pip, app, proj)
	if err != nil {
		return err
	}

	// Delete trigger
	query = `DELETE FROM pipeline_trigger WHERE pipeline_trigger.id IN (
			SELECT pipeline_trigger.id FROM pipeline_trigger
			JOIN pipeline AS src_pip ON src_pip.id = src_pipeline_id
			JOIN application AS src_app ON src_app.id = src_application_id
			JOIN project AS src_proj ON src_proj.id = src_app.project_id
			JOIN pipeline AS dest_pip ON dest_pip.id = dest_pipeline_id
			JOIN application AS dest_app ON dest_app.id = dest_application_id
			JOIN project AS dest_proj ON dest_proj.id = dest_app.project_id
			WHERE (src_pip.name = $1 AND src_app.name = $2 AND src_proj.projectkey = $3)
			OR (dest_pip.name = $1 AND dest_app.name = $2 AND dest_proj.projectkey = $3)
	)`
	_, err = db.Exec(query, pip, app, proj)
	if err != nil {
		return err
	}

	return nil
}

// DeletePipelineTriggers removes from database all triggers where given pipeline is present
func DeletePipelineTriggers(db gorp.SqlExecutor, pipelineID int64) error {

	// Delete parameters
	query := `DELETE FROM pipeline_trigger_parameter WHERE pipeline_trigger_id IN (
				SELECT id FROM pipeline_trigger WHERE src_pipeline_id = $1 OR dest_pipeline_id = $1
			)`
	_, err := db.Exec(query, pipelineID)
	if err != nil {
		return err
	}

	// Delete prerequisites
	query = `DELETE FROM pipeline_trigger_prerequisite WHERE pipeline_trigger_id IN (
					SELECT id FROM pipeline_trigger WHERE src_pipeline_id = $1 OR dest_pipeline_id = $1
				)`
	_, err = db.Exec(query, pipelineID)
	if err != nil {
		return err
	}

	// Delete triggers
	query = `DELETE FROM pipeline_trigger WHERE src_pipeline_id = $1 OR dest_pipeline_id = $1`
	_, err = db.Exec(query, pipelineID)
	if err != nil {
		return err
	}

	return nil
}

// DeleteApplicationTriggers removes from database all triggers where given application is present
func DeleteApplicationTriggers(db gorp.SqlExecutor, appID int64) error {

	// Delete parameters
	query := `DELETE FROM pipeline_trigger_parameter WHERE pipeline_trigger_id IN (
					SELECT id FROM pipeline_trigger WHERE src_application_id = $1 OR dest_application_id = $1
				)`
	_, err := db.Exec(query, appID)
	if err != nil {
		return err
	}

	// Delete prerequisites
	query = `DELETE FROM pipeline_trigger_prerequisite WHERE pipeline_trigger_id IN (
					SELECT id FROM pipeline_trigger WHERE src_application_id = $1 OR dest_application_id = $1
				)`
	_, err = db.Exec(query, appID)
	if err != nil {
		return err
	}

	// Delete triggers
	query = `DELETE FROM pipeline_trigger WHERE src_application_id = $1 OR dest_application_id = $1`
	_, err = db.Exec(query, appID)
	if err != nil {
		return err
	}

	return nil
}

// DeleteTrigger removes from database given trigger
func DeleteTrigger(db gorp.SqlExecutor, triggerID int64) error {

	// Delete parameters
	query := `DELETE FROM pipeline_trigger_parameter WHERE pipeline_trigger_id = $1`
	_, err := db.Exec(query, triggerID)
	if err != nil {
		return err
	}

	// Delete prerequisites
	query = `DELETE FROM pipeline_trigger_prerequisite WHERE pipeline_trigger_id = $1`
	_, err = db.Exec(query, triggerID)
	if err != nil {
		return err
	}

	// Delete trigger
	query = `DELETE FROM pipeline_trigger WHERE id = $1`
	_, err = db.Exec(query, triggerID)
	if err != nil {
		return err
	}

	return nil
}

// ProcessTriggerParameters replaces all placeholders in trigger before execution
func ProcessTriggerParameters(t sdk.PipelineTrigger, pbParams []sdk.Parameter) []sdk.Parameter {
	loopEscape := 0
	for loopEscape < 10 {
		replaced := false
		// Now for each trigger parameter
		for _, pbp := range pbParams {
			// Replace placeholders with their value
			for i := range t.Parameters {
				old := t.Parameters[i].Value
				t.Parameters[i].Value = strings.Replace(t.Parameters[i].Value, "{{."+pbp.Name+"}}", pbp.Value, -1)
				if t.Parameters[i].Value != old {
					replaced = true
				}
			}
		}
		// If nothing has been replace, exit
		if !replaced {
			break
		}
		loopEscape++
	}

	return t.Parameters
}

//ProcessTriggerExpectedValue processes prerequisites expected values
func ProcessTriggerExpectedValue(payload string, pb *sdk.PipelineBuild) string {
	loopEscape := 0
	for loopEscape < 10 {
		replaced := false
		for _, pbp := range pb.Parameters {
			old := payload
			payload = strings.Replace(payload, "{{."+pbp.Name+"}}", pbp.Value, -1)
			if payload != old {
				replaced = true
			}
		}
		if !replaced {
			break
		}
		loopEscape++
	}

	return payload
}

// CheckPrerequisites verifies that all prerequisite are matched before scheduling
func CheckPrerequisites(t sdk.PipelineTrigger, pb *sdk.PipelineBuild) (bool, error) {

	// Process parameters
	parameters := ProcessTriggerParameters(t, pb.Parameters)

	// Check conditions
	prerequisitesOK := true
	for _, p := range t.Prerequisites {
		// Process prerequisite too !
		expectedValue := ProcessTriggerExpectedValue(p.ExpectedValue, pb)
		var not bool
		if strings.HasPrefix(expectedValue, "not ") {
			expectedValue = strings.Replace(expectedValue, "not ", "", 1)
			not = true
		}
		// Look for parameter in PipelineBuild
		found := false
		for i := range parameters {
			if p.Parameter == parameters[i].Name {
				found = true
				ok, err := regexp.Match("^"+expectedValue+"$", []byte(parameters[i].Value))
				if err != nil {
					log.Warning("CheckPrerequisites> Cannot eval regexp '%s': %s", expectedValue, err)
					return false, fmt.Errorf("CheckPrerequisites> %s", err)
				}
				if (!not && !ok) || (not && ok) {
					log.Debug("CheckPrerequisites> Expected %s='%s', got '%s'\n", parameters[i].Name, expectedValue, parameters[i].Value)
					prerequisitesOK = false
					break
				}
			}
		}

		// Look for git.branch in PipelineBuild parameters now
		for _, pbp := range pb.Parameters {
			if p.Parameter == pbp.Name {
				found = true
				ok, err := regexp.Match("^"+expectedValue+"$", []byte(pbp.Value))
				if err != nil {
					log.Warning("CheckPrerequisites> Cannot eval regexp '%s': %s", expectedValue, err)
					return false, fmt.Errorf("CheckPrerequisites> %s", err)
				}
				if (!not && !ok) || (not && ok) {
					log.Debug("CheckPrerequisites> Expected %s='%s', got '%s'\n", p.Parameter, expectedValue, pbp.Value)
					prerequisitesOK = false
					break
				}

			}
		}

		if !found { // Not even found...
			prerequisitesOK = false
			log.Info("CheckPrerequisites> Prereq on '%s', not found\n", p.Parameter)
			break
		}
	}

	return prerequisitesOK, nil
}

//Exists checks if trigger exists
func Exists(db gorp.SqlExecutor, applicationSource, pipelineSource, EnvSource, applicationDest, pipelineDest, EnvDest int64) (bool, error) {
	query := `SELECT COUNT(1) FROM pipeline_trigger
			  WHERE src_application_id = $1
			  AND src_pipeline_id = $2
			  AND src_environment_id = $3
			  AND dest_application_id = $4
			  AND dest_pipeline_id = $5
			  AND dest_environment_id = $6`
	var n int
	if err := db.QueryRow(query, applicationSource, pipelineSource, EnvSource, applicationDest, pipelineDest, EnvDest).Scan(&n); err != nil {
		return false, err
	}
	return n == 1, nil
}
