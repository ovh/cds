package action

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Exists check if an action with same name already exists in database
func Exists(db gorp.SqlExecutor, name string) (bool, error) {
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
func InsertAction(tx gorp.SqlExecutor, a *sdk.Action, public bool) error {
	ok, errLoop := isTreeLoopFree(tx, a, nil)
	if errLoop != nil {
		return errLoop
	}
	if !ok {
		return sdk.ErrActionLoop
	}

	query := `INSERT INTO action (name, description, type, enabled, deprecated, public) VALUES($1, $2, $3, $4, $5, $6) RETURNING id`
	if err := tx.QueryRow(query, a.Name, a.Description, a.Type, a.Enabled, a.Deprecated, public).Scan(&a.ID); err != nil {
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
			a.Actions[i].AlwaysExecuted = ch.AlwaysExecuted || a.Actions[i].AlwaysExecuted
			a.Actions[i].Optional = ch.Optional || a.Actions[i].Optional
			log.Debug("InsertAction> Get existing child Action %s with enabled:%t", a.Actions[i].Name, a.Actions[i].Enabled)
		} else {
			log.Debug("InsertAction> Child Action %s is knowned with enabled:%t", a.Actions[i].Name, a.Actions[i].Enabled)
		}

		log.Debug("InsertAction> Insert Child Action %s", a.Actions[i].Name)
		if err := insertActionChild(tx, a.ID, a.Actions[i], i+1); err != nil {
			return err
		}
	}

	// Requirements of children are requirement of parent
	for _, c := range a.Actions {
		if len(c.Requirements) == 0 {
			log.Debug("Try load children action requirement for id:%d", c.ID)
			var errLoad error
			c.Requirements, errLoad = LoadActionRequirements(tx, c.ID)
			if errLoad != nil {
				return fmt.Errorf("cannot LoadActionRequirements in InsertAction> %s", errLoad)
			}
		}
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

	if err := isRequirementsValid(a.Requirements); err != nil {
		return err
	}

	for i := range a.Requirements {
		if err := InsertActionRequirement(tx, a.ID, a.Requirements[i]); err != nil {
			return err
		}
	}

	for i := range a.Parameters {
		if err := InsertActionParameter(tx, a.ID, a.Parameters[i]); err != nil {
			return sdk.WrapError(err, "Cannot InsertActionParameter %s", a.Parameters[i].Name)
		}
	}

	return nil
}

// LoadPublicAction load an action from database
func LoadPublicAction(db gorp.SqlExecutor, name string) (*sdk.Action, error) {
	query := `SELECT id, name, description, type, last_modified, enabled, deprecated FROM action WHERE lower(action.name) = lower($1) AND public = true`
	a, err := loadActions(db, query, name)
	if err != nil {
		return nil, err
	}
	return &a[0], nil
}

// LoadActionByID retrieves in database the action with given id
func LoadActionByID(db gorp.SqlExecutor, actionID int64) (*sdk.Action, error) {
	query := `SELECT id, name, description, type, last_modified, enabled, deprecated FROM action WHERE action.id = $1`
	a, err := loadActions(db, query, actionID)
	if err != nil {
		return nil, err
	}
	return &a[0], nil
}

// LoadActions load all actions from database
func LoadActions(db gorp.SqlExecutor) ([]sdk.Action, error) {
	query := `SELECT id, name, description, type, last_modified, enabled, deprecated FROM action WHERE public = true ORDER BY name`
	return loadActions(db, query)
}

func loadActions(db gorp.SqlExecutor, query string, args ...interface{}) ([]sdk.Action, error) {
	var acts []sdk.Action
	rows, err := db.Query(query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.ErrNoAction
		}
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		a := sdk.Action{}
		var lastModified time.Time
		if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.Type, &lastModified, &a.Enabled, &a.Deprecated); err != nil {
			if err == sql.ErrNoRows {
				return nil, sdk.ErrNoAction
			}
			return nil, fmt.Errorf("cannot Scan> %s", err)
		}
		a.LastModified = lastModified.Unix()
		acts = append(acts, a)
	}

	if len(acts) == 0 {
		return nil, sdk.ErrNoAction
	}

	for i := range acts {
		if err := loadActionDependencies(db, &acts[i]); err != nil {
			return nil, err
		}
	}
	return acts, nil
}

func loadActionDependencies(db gorp.SqlExecutor, a *sdk.Action) error {
	var err error
	// Load requirements
	a.Requirements, err = LoadActionRequirements(db, a.ID)
	if err != nil {
		return fmt.Errorf("cannot LoadActionRequirements> %s", err)
	}

	// Load parameters
	a.Parameters, err = LoadActionParameters(db, a.ID)
	if err != nil {
		return fmt.Errorf("cannot LoadActionParameters> %s", err)
	}

	// Don't try to load children is action is builtin
	if a.Type == sdk.BuiltinAction {
		return nil
	}

	// Load children
	a.Actions, err = loadActionChildren(db, a.ID)
	if err != nil {
		return fmt.Errorf("cannot loadActionChildren> %s", err)
	}

	computeRequirements(a)

	return nil
}

// UpdateActionDB  Update an action
func UpdateActionDB(db gorp.SqlExecutor, a *sdk.Action, userID int64) error {
	ok, errLoop := isTreeLoopFree(db, a, nil)
	if errLoop != nil {
		return errLoop
	}
	if !ok {
		return sdk.ErrActionLoop
	}

	if err := insertAudit(db, a.ID, userID, "Action update"); err != nil {
		return err
	}

	if err := deleteActionChildren(db, a.ID); err != nil {
		return err
	}
	for i := range a.Actions {
		// if child id is not given, try to load by name
		if a.Actions[i].ID == 0 {
			ch, errl := LoadPublicAction(db, a.Actions[i].Name)
			if errl != nil {
				return errl
			}
			a.Actions[i].ID = ch.ID
		}

		if err := insertActionChild(db, a.ID, a.Actions[i], i+1); err != nil {
			return err
		}
	}

	if err := DeleteActionParameters(db, a.ID); err != nil {
		return err
	}
	for i := range a.Parameters {
		if err := InsertActionParameter(db, a.ID, a.Parameters[i]); err != nil {
			return sdk.WrapError(err, "InsertActionParameter for %s failed", a.Parameters[i].Name)
		}
	}

	if err := DeleteActionRequirements(db, a.ID); err != nil {
		return err
	}

	//TODO we don't need to compute all job requirements here, but only when running the job
	// Requirements of children are requirement of parent
	computeRequirements(a)

	// Checks if multiple requirements have the same name
	if err := isRequirementsValid(a.Requirements); err != nil {
		return err
	}

	for i := range a.Requirements {
		if err := InsertActionRequirement(db, a.ID, a.Requirements[i]); err != nil {
			return err
		}
	}

	query := `UPDATE action SET name=$1, description=$2, type=$3, enabled=$4, deprecated=$5 WHERE id=$6`
	_, errdb := db.Exec(query, a.Name, a.Description, string(a.Type), a.Enabled, a.Deprecated, a.ID)
	return errdb
}

// DeleteAction remove action from database
func DeleteAction(db gorp.SqlExecutor, actionID, userID int64) error {

	if err := insertAudit(db, actionID, userID, "Action delete"); err != nil {
		return err
	}

	if err := deleteActionChildren(db, actionID); err != nil {
		return err
	}

	query := `DELETE FROM pipeline_action WHERE action_id = $1`
	if _, err := db.Exec(query, actionID); err != nil {
		return err
	}

	query = `DELETE FROM action_parameter WHERE action_id = $1`
	if _, err := db.Exec(query, actionID); err != nil {
		return err
	}

	query = `DELETE FROM action_requirement WHERE action_id = $1`
	if _, err := db.Exec(query, actionID); err != nil {
		return err
	}

	query = `DELETE FROM action WHERE action.id = $1`
	if _, err := db.Exec(query, actionID); err != nil {
		return err
	}
	return nil
}

// Used checks if action is used in another action or in a pipeline
func Used(db gorp.SqlExecutor, actionID int64) (bool, error) {
	var count int

	query := `SELECT COUNT(id) FROM pipeline_action WHERE pipeline_action.action_id = $1`
	if err := db.QueryRow(query, actionID).Scan(&count); err != nil {
		return false, err
	}

	if count > 0 {
		return true, nil
	}

	query = `SELECT COUNT(id) FROM action_edge WHERE child_id = $1`
	if err := db.QueryRow(query, actionID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func isTreeLoopFree(db gorp.SqlExecutor, a *sdk.Action, parents []int64) (bool, error) {
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

func insertAudit(db gorp.SqlExecutor, actionID, userID int64, change string) error {
	a, errLoad := LoadActionByID(db, actionID)
	if errLoad != nil {
		return errLoad
	}

	query := `INSERT INTO action_audit (action_id, user_id, change, versionned, action_json)
			VALUES ($1, $2, $3, NOW(), $4)`

	b, errJSON := json.Marshal(a)
	if errJSON != nil {
		return errJSON
	}

	if _, err := db.Exec(query, actionID, userID, change, b); err != nil {
		return err
	}

	return nil
}

func isRequirementsValid(requirements sdk.RequirementList) error {
	nbModelReq, nbHostnameReq := 0, 0
	for i := range requirements {
		for j := range requirements {
			if requirements[i].Name == requirements[j].Name && requirements[i].Type == requirements[j].Type && i != j {
				return sdk.WrapError(sdk.ErrInvalidJobRequirement, "For requirement name %s and type %s", requirements[i].Name, requirements[i].Type)
			}
		}
		switch requirements[i].Type {
		case sdk.ModelRequirement:
			nbModelReq++
		case sdk.HostnameRequirement:
			nbHostnameReq++
		}
	}
	if nbModelReq > 1 {
		return sdk.ErrInvalidJobRequirementDuplicateModel
	}
	if nbHostnameReq > 1 {
		return sdk.ErrInvalidJobRequirementDuplicateHostname
	}
	return nil
}

// PipelineUsingAction represent a pipeline using an action
type PipelineUsingAction struct {
	ActionID         int    `json:"action_id"`
	ActionType       string `json:"type"`
	ActionName       string `json:"action_name"`
	PipName          string `json:"pipeline_name"`
	AppName          string `json:"application_name"`
	EnvID            int64  `json:"environment_id"`
	ProjName         string `json:"project_name"`
	ProjKey          string `json:"key"`
	StageID          int64  `json:"stage_id"`
	WorkflowName     string `json:"workflow_name"`
	WorkflowNodeName string `json:"workflow_node_name"`
	WorkflowNodeID   int64  `json:"workflow_node_id"`
}

// GetPipelineUsingAction returns the list of pipelines using an action
func GetPipelineUsingAction(db gorp.SqlExecutor, name string) ([]PipelineUsingAction, error) {
	query := `
		SELECT
			action.type, action.name as actionName, action.id as actionId,
			pipeline_stage.id as stageId,
			pipeline.name as pipName, project.name, project.projectkey,
			workflow.name as wName, workflow_node.id as nodeId,  workflow_node.name as nodeName
		FROM action_edge
		LEFT JOIN action on action.id = parent_id
		LEFT OUTER JOIN pipeline_action ON pipeline_action.action_id = action.id
		LEFT OUTER JOIN pipeline_stage ON pipeline_stage.id = pipeline_action.pipeline_stage_id
		LEFT OUTER JOIN pipeline ON pipeline.id = pipeline_stage.pipeline_id
		LEFT OUTER JOIN project ON pipeline.project_id = project.id
		LEFT OUTER JOIN workflow_node ON workflow_node.pipeline_id = pipeline.id
		LEFT OUTER JOIN workflow ON workflow_node.workflow_id = workflow.id
		LEFT JOIN action as actionChild ON  actionChild.id = child_id
		WHERE actionChild.name = $1 and actionChild.public = true AND pipeline.name IS NOT NULL
		ORDER BY projectkey, pipName, actionName;
	`
	rows, errq := db.Query(query, name)
	if errq != nil {
		return nil, sdk.WrapError(errq, "getPipelineUsingAction> Cannot load pipelines using action %s", name)
	}
	defer rows.Close()

	response := []PipelineUsingAction{}

	for rows.Next() {
		var a PipelineUsingAction
		var pipName, projName, projKey, wName, wnodeName sql.NullString
		var stageID, nodeID sql.NullInt64
		if err := rows.Scan(&a.ActionType, &a.ActionName, &a.ActionID, &stageID,
			&pipName, &projName, &projKey,
			&wName, &nodeID, &wnodeName,
		); err != nil {
			return nil, sdk.WrapError(err, "Cannot read sql response")
		}
		if stageID.Valid {
			a.StageID = stageID.Int64
		}
		if pipName.Valid {
			a.PipName = pipName.String
		}
		if projName.Valid {
			a.ProjName = projName.String
		}
		if projKey.Valid {
			a.ProjKey = projKey.String
		}
		if wName.Valid {
			a.WorkflowName = wName.String
		}
		if wnodeName.Valid {
			a.WorkflowNodeName = wnodeName.String
		}
		if nodeID.Valid {
			a.WorkflowNodeID = nodeID.Int64
		}

		response = append(response, a)
	}

	return response, nil
}

func computeRequirements(a *sdk.Action) {
	if a.Enabled {
		// Requirements of children are requirement of parent
		for _, c := range a.Actions {
			if !c.Enabled { // If action is not enabled we don't need their requirements
				continue
			}
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
	} else {
		a.Requirements = []sdk.Requirement{}
	}
}
