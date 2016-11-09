package action

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// Exists check if an action with same name already exists in database
func Exists(db database.Querier, name string) (bool, error) {
	query := `SELECT * FROM action WHERE action.name = $1`
	rows, err := db.Query(query, name)
	if err != nil {
		log.Warning("Exists> Cannot check if action %s exist: %s\n", name, err)
		return false, err
	}
	defer rows.Close()
	if rows.Next() {
		log.Debug("Exists> Action %s already exists\n", name)
		return true, nil
	}
	return false, nil
}

// InsertAction insert given action into given database
func InsertAction(tx database.QueryExecuter, a *sdk.Action, public bool) error {
	ok, err := isTreeLoopFree(tx, a, nil)
	if err != nil {
		return err
	}
	if !ok {
		return sdk.ErrActionLoop
	}

	query := `INSERT INTO action (name, description, type, enabled, public) VALUES($1, $2, $3, $4, $5) RETURNING id`
	err = tx.QueryRow(query, a.Name, a.Description, a.Type, a.Enabled, public).Scan(&a.ID)
	if err != nil {
		return err
	}

	for i := range a.Actions {
		// Check that action does not use itself recursively
		if a.Actions[i].ID == a.ID {
			return fmt.Errorf("cds: cannot use action recursively")
		}

		// if child id is not given, try to load by name
		if a.Actions[i].ID == 0 {
			ch, errl := LoadPublicAction(tx, a.Actions[i].Name)
			if errl != nil {
				return errl
			}
			a.Actions[i].ID = ch.ID
		}

		if err = insertActionChild(tx, a.ID, a.Actions[i], i+1); err != nil {
			return err
		}
	}

	// Requirements of children are requirement of parent
	for _, c := range a.Actions {
		// Now for each requirement of child, check if it exists in parent
		for _, cr := range c.Requirements {
			found := false
			for _, pr := range a.Requirements {
				if pr.Type == cr.Type && pr.Value == cr.Value {
					found = true
					break
				}
			}
			if !found {
				a.Requirements = append(a.Requirements, cr)
			}
		}
	}
	for i := range a.Requirements {
		if err = InsertActionRequirement(tx, a.ID, a.Requirements[i]); err != nil {
			return err
		}
	}

	for i := range a.Parameters {
		if err = InsertActionParameter(tx, a.ID, a.Parameters[i]); err != nil {
			log.Warning("InsertAction> Cannot InsertActionParameter %s: %s\n", a.Parameters[i].Name, err)
			return err
		}
	}

	return nil
}

// LoadPipelineActionByID retrieves and action by its id but check project and pipeline
func LoadPipelineActionByID(db database.Querier, project, pip string, actionID int64) (*sdk.Action, error) {
	query := `
	SELECT action.id, action.name, action.description, action.type, action.last_modified, action.enabled
	FROM action
	JOIN pipeline_action ON pipeline_action.action_id = $1
	JOIN pipeline_stage ON pipeline_stage.id = pipeline_action.pipeline_stage_id
	JOIN pipeline ON pipeline.id = pipeline_stage.pipeline_id
	JOIN project ON project.id = pipeline.project_id
	WHERE action.id = $1 AND pipeline.name = $2 AND project.projectkey = $3`
	return loadAction(db, db.QueryRow(query, actionID, pip, project))
}

// LoadPublicAction load an action from database
func LoadPublicAction(db database.Querier, name string) (*sdk.Action, error) {
	query := `SELECT id, name, description, type, last_modified, enabled FROM action WHERE action.name = $1 AND public = true`
	return loadAction(db, db.QueryRow(query, name))
}

// LoadActionByID retrieves in database the action with given id
func LoadActionByID(db database.Querier, actionID int64) (*sdk.Action, error) {
	query := `SELECT id, name, description, type, last_modified, enabled FROM action WHERE action.id = $1`
	//return loadAction(db, db.QueryRow(query, actionID), args...)
	return loadAction(db, db.QueryRow(query, actionID))
}

// LoadActionByPipelineActionID load an action from database
func LoadActionByPipelineActionID(db database.Querier, pipelineActionID int64) (*sdk.Action, error) {
	query := `SELECT action.id, action.name, action.description, action.type, action.last_modified, action.enabled
	          FROM action
	          JOIN pipeline_action ON pipeline_action.action_id = action.id
	          WHERE pipeline_action.id = $1`

	//return loadAction(db, db.QueryRow(query, pipelineActionID), args...)
	return loadAction(db, db.QueryRow(query, pipelineActionID))
}

// LoadActions load all actions from database
func LoadActions(db *sql.DB) ([]sdk.Action, error) {
	var acts []sdk.Action

	query := `SELECT id, name, description, type, last_modified, enabled FROM action WHERE public = true ORDER BY name`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		a, err := loadAction(db, rows)
		if err != nil {
			return nil, err
		}
		acts = append(acts, *a)
	}
	return acts, nil
}

func loadAction(db database.Querier, s database.Scanner) (*sdk.Action, error) {
	a := &sdk.Action{}

	var lastModified time.Time
	err := s.Scan(&a.ID, &a.Name, &a.Description, &a.Type, &lastModified, &a.Enabled)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNoAction
		}
		return nil, fmt.Errorf("cannot Scan> %s", err)
	}
	a.LastModified = lastModified.Unix()
	// Load requirements
	a.Requirements, err = LoadActionRequirements(db, a.ID)
	if err != nil {
		return nil, fmt.Errorf("cannot LoadActionRequirements> %s", err)
	}

	// Load parameters
	a.Parameters, err = LoadActionParameters(db, a.ID)
	if err != nil {
		return nil, fmt.Errorf("cannot LoadActionParameters> %s", err)
	}

	// Don't try to load children is action is builtin
	if a.Type == sdk.BuiltinAction {
		return a, nil
	}

	// Load children
	a.Actions, err = loadActionChildren(db, a.ID)
	if err != nil {
		return nil, fmt.Errorf("cannot loadActionChildren> %s", err)
	}

	// Requirements of children are requirement of parent
	for _, c := range a.Actions {
		// Now for each requirement of child, check if it exists in parent
		for _, cr := range c.Requirements {
			found := false
			for _, pr := range a.Requirements {
				if pr.Type == cr.Type && pr.Value == cr.Value {
					found = true
					break
				}
			}
			if !found {
				a.Requirements = append(a.Requirements, cr)
			}
		}
	}

	return a, nil
}

// UpdateActionDB  Update an action
func UpdateActionDB(tx *sql.Tx, a *sdk.Action, userID int64) error {

	ok, err := isTreeLoopFree(tx, a, nil)
	if err != nil {
		return err
	}
	if !ok {
		return sdk.ErrActionLoop
	}

	err = insertAudit(tx, a.ID, userID, "Action update")
	if err != nil {
		return err
	}

	err = deleteActionChildren(tx, a.ID)
	if err != nil {
		return err
	}
	for i := range a.Actions {
		// if child id is not given, try to load by name
		if a.Actions[i].ID == 0 {
			ch, errl := LoadPublicAction(tx, a.Actions[i].Name)
			if errl != nil {
				return errl
			}
			a.Actions[i].ID = ch.ID
		}

		if err = insertActionChild(tx, a.ID, a.Actions[i], i+1); err != nil {
			return err
		}
	}

	err = DeleteActionParameters(tx, a.ID)
	if err != nil {
		return err
	}
	for i := range a.Parameters {
		err = InsertActionParameter(tx, a.ID, a.Parameters[i])
		if err != nil {
			log.Warning("UpdateAction> InsertActionParameter for %s failed: %s\n", a.Parameters[i].Name, err)
			return err
		}
	}

	err = DeleteActionRequirements(tx, a.ID)
	if err != nil {
		return err
	}
	// Requirements of children are requirement of parent
	for _, c := range a.Actions {
		// Now for each requirement of child, check if it exists in parent
		for _, cr := range c.Requirements {
			found := false
			for _, pr := range a.Requirements {
				if pr.Type == cr.Type && pr.Value == cr.Value {
					found = true
					break
				}
			}
			if !found {
				a.Requirements = append(a.Requirements, cr)
			}
		}
	}
	for i := range a.Requirements {
		err = InsertActionRequirement(tx, a.ID, a.Requirements[i])
		if err != nil {
			return err
		}
	}

	query := `UPDATE action SET name=$1,description=$2, type=$3, enabled=$4 WHERE id=$5`
	_, err = tx.Exec(query, a.Name, a.Description, string(a.Type), a.Enabled, a.ID)
	if err != nil {
		return err
	}

	return nil
}

// DeleteAction remove action from database
func DeleteAction(db database.QueryExecuter, actionID, userID int64) error {

	err := insertAudit(db, actionID, userID, "Action delete")
	if err != nil {
		return err
	}

	err = deleteActionChildren(db, actionID)
	if err != nil {
		return err
	}

	query := `DELETE FROM build_log WHERE action_build_id IN
	(SELECT id FROM action_build WHERE pipeline_action_id IN
		(SELECT id FROM pipeline_action WHERE action_id = $1)
	)`
	_, err = db.Exec(query, actionID)
	if err != nil {
		return err
	}

	query = `DELETE FROM action_build WHERE pipeline_action_id IN
		(SELECT id FROM pipeline_action WHERE action_id = $1)`
	_, err = db.Exec(query, actionID)
	if err != nil {
		return err
	}

	query = `DELETE FROM pipeline_action WHERE action_id = $1`
	_, err = db.Exec(query, actionID)
	if err != nil {
		return err
	}

	query = `DELETE FROM action_parameter WHERE action_id = $1`
	_, err = db.Exec(query, actionID)
	if err != nil {
		return err
	}

	query = `DELETE FROM action_requirement WHERE action_id = $1`
	_, err = db.Exec(query, actionID)
	if err != nil {
		return err
	}

	query = `DELETE FROM action WHERE action.id = $1`
	_, err = db.Exec(query, actionID)
	if err != nil {
		return err
	}

	return nil
}

// Used checks if action is used in another action or in a pipeline
func Used(db *sql.DB, actionID int64) (bool, error) {
	var count int

	query := `SELECT COUNT(id) FROM pipeline_action WHERE pipeline_action.action_id = $1`
	err := db.QueryRow(query, actionID).Scan(&count)
	if err != nil {
		return false, err
	}

	if count > 0 {
		return true, nil
	}

	query = `SELECT COUNT(id) FROM action_edge WHERE child_id = $1`
	err = db.QueryRow(query, actionID).Scan(&count)
	if err != nil {
		return false, err
	}

	if count > 0 {
		return true, nil
	}

	return false, nil
}

func isTreeLoopFree(db database.Querier, a *sdk.Action, parents []int64) (bool, error) {
	var err error

	// First, check yourself
	for _, p := range parents {
		if a.ID == p {
			log.Warning("Action %s is already used higher in the tree\n", a.Name)
			return false, nil
		}
	}

	// if builtin, then it's ok
	if a.Type == sdk.BuiltinAction {
		return true, nil
	}

	// Then check your children
	for _, child := range a.Actions {
		cobaye := &child

		// If child id is not provided, load it properly
		if cobaye.ID == 0 {
			cobaye, err = LoadPublicAction(db, cobaye.Name)
			if err != nil {
				log.Warning("isTreeLoopFree> error on action %s: %s", child.Name, err)
				return false, err
			}
		}

		ret, err := isTreeLoopFree(db, cobaye, append(parents, a.ID))
		if !ret {
			return false, err
		}
	}

	return true, nil
}

func insertAudit(db database.QueryExecuter, actionID, userID int64, change string) error {
	a, err := LoadActionByID(db, actionID)
	if err != nil {
		return err
	}

	query := `INSERT INTO action_audit (action_id, user_id, change, versionned, action_json)
			VALUES ($1, $2, $3, NOW(), $4)`
	_, err = db.Exec(query, actionID, userID, change, a.JSON())
	if err != nil {
		return err
	}

	return nil
}
