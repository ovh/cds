package workflow

import (
	"database/sql"
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// InsertOrUpdateTrigger inserts or updates a trigger
func insertOrUpdateTrigger(db gorp.SqlExecutor, w *sdk.Workflow, node *sdk.WorkflowNode, trigger *sdk.WorkflowNodeTrigger, u *sdk.User) error {
	trigger.WorkflowNodeID = node.ID
	var oldTrigger *sdk.WorkflowNodeTrigger

	//Try to load the trigger
	if trigger.ID != 0 {
		var err error
		oldTrigger, err = loadTrigger(db, w, node, trigger.ID, u)
		if err != nil {
			return sdk.WrapError(err, "InsertOrUpdateTrigger> Unable to load trigger %d", trigger.ID)
		}
	}

	//Delete the old trigger
	if oldTrigger != nil {
		if err := deleteTrigger(db, oldTrigger); err != nil {
			return sdk.WrapError(err, "InsertOrUpdateTrigger> Unable to delete trigger %d", trigger.ID)
		}
	}

	//Setup destination node
	if err := insertOrUpdateNode(db, w, &trigger.WorkflowDestNode, u); err != nil {
		return sdk.WrapError(err, "InsertOrUpdateTrigger> Unable to setup destination node")
	}
	trigger.WorkflowDestNodeID = trigger.WorkflowDestNode.ID

	//Insert trigger
	dbt := NodeTrigger(*trigger)
	if err := db.Insert(&dbt); err != nil {
		return sdk.WrapError(err, "InsertOrUpdateTrigger> Unable to insert trigger")
	}
	trigger.ID = dbt.ID

	//Manage conditions
	b, err := json.Marshal(trigger.Conditions)
	if err != nil {
		return sdk.WrapError(err, "InsertOrUpdateTrigger> Unable to marshal trigger conditions")
	}
	if _, err := db.Exec("UPDATE workflow_node_trigger SET conditions = $1 where id = $2", b, trigger.ID); err != nil {
		return sdk.WrapError(err, "InsertOrUpdateTrigger> Unable to set trigger conditions in database")
	}

	return nil
}

// DeleteTrigger deletes a trigger and all chrildren
func deleteTrigger(db gorp.SqlExecutor, trigger *sdk.WorkflowNodeTrigger) error {
	dbt := NodeTrigger(*trigger)
	if _, err := db.Delete(&dbt); err != nil {
		return sdk.WrapError(err, "DeleteTrigger> Unable to delete trigger %d", dbt.ID)
	}
	return nil
}

// LoadTriggers loads trigger from a node
func loadTriggers(db gorp.SqlExecutor, w *sdk.Workflow, node *sdk.WorkflowNode, u *sdk.User) ([]sdk.WorkflowNodeTrigger, error) {
	dbtriggers := []NodeTrigger{}
	if _, err := db.Select(&dbtriggers, "select * from workflow_node_trigger where workflow_node_id = $1", node.ID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "LoadTriggers> Unable to load triggers")
	}

	if len(dbtriggers) == 0 {
		return nil, nil
	}

	triggers := []sdk.WorkflowNodeTrigger{}
	for _, dbt := range dbtriggers {
		t := sdk.WorkflowNodeTrigger(dbt)
		if t.WorkflowDestNodeID != 0 {
			//Load destination node
			dest, err := loadNode(db, w, t.WorkflowDestNodeID, u)
			if err != nil {
				return nil, sdk.WrapError(err, "LoadTriggers> Unable to load destination node %d", t.WorkflowDestNodeID)
			}
			t.WorkflowDestNode = *dest
		}

		//Load conditions
		sqlConditions, err := db.SelectNullStr("select conditions from workflow_node_trigger where id = $1", t.ID)
		if err != nil {
			return nil, sdk.WrapError(err, "LoadTriggers> Unable to load conditions for trigger %d", t.ID)
		}
		if sqlConditions.Valid {
			if err := json.Unmarshal([]byte(sqlConditions.String), &t.Conditions); err != nil {
				return nil, sdk.WrapError(err, "LoadTriggers> Unable to unmarshall conditions for trigger %d", t.ID)
			}
		}

		triggers = append(triggers, t)
	}
	return triggers, nil
}

// LoadTrigger loads a specific trigger from a node
func loadTrigger(db gorp.SqlExecutor, w *sdk.Workflow, node *sdk.WorkflowNode, id int64, u *sdk.User) (*sdk.WorkflowNodeTrigger, error) {
	dbtrigger := NodeTrigger{}
	if err := db.SelectOne(&dbtrigger, "select * from workflow_node_trigger where workflow_node_id = $1 and id = $2", node.ID, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "LoadTriggers> Unable to load trigger %d", id)
	}

	t := sdk.WorkflowNodeTrigger(dbtrigger)

	if t.WorkflowDestNodeID != 0 {
		dest, err := loadNode(db, w, t.WorkflowDestNodeID, u)
		if err != nil {
			return nil, sdk.WrapError(err, "LoadTrigger> Unable to load destination node %d", t.WorkflowDestNodeID)
		}
		t.WorkflowDestNode = *dest
	}

	//Load conditions
	sqlConditions, err := db.SelectNullStr("select conditions from workflow_node_trigger where id = $1", t.ID)
	if err != nil {
		return nil, sdk.WrapError(err, "LoadTriggers> Unable to load conditions for trigger %d", t.ID)
	}
	if sqlConditions.Valid {
		if err := json.Unmarshal([]byte(sqlConditions.String), t.Conditions); err != nil {
			return nil, sdk.WrapError(err, "LoadTriggers> Unable to unmarshall conditions for trigger %d", t.ID)
		}
	}

	return &t, nil
}
