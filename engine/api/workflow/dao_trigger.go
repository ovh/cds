package workflow

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// insertTrigger inserts a trigger
func insertTrigger(db gorp.SqlExecutor, w *sdk.Workflow, node *sdk.WorkflowNode, trigger *sdk.WorkflowNodeTrigger, u *sdk.User) error {
	defer func() {
		log.Debug("insertTrigger> insert or update node %d (%s) on %s trigger %d", node.ID, node.Ref, node.Pipeline.Name, trigger.ID)
	}()
	trigger.WorkflowNodeID = node.ID
	trigger.ID = 0
	trigger.WorkflowDestNodeID = 0

	//Setup destination node
	if err := insertNode(db, w, &trigger.WorkflowDestNode, u, false); err != nil {
		return sdk.WrapError(err, "insertTrigger> Unable to setup destination node %d on trigger %d", trigger.WorkflowDestNode.ID, trigger.ID)
	}
	trigger.WorkflowDestNodeID = trigger.WorkflowDestNode.ID

	//Insert trigger
	dbt := NodeTrigger(*trigger)
	if err := db.Insert(&dbt); err != nil {
		return sdk.WrapError(err, "insertTrigger> Unable to insert trigger")
	}
	trigger.ID = dbt.ID
	trigger.WorkflowDestNode.TriggerSrcID = trigger.ID

	// Update node trigger ID
	if err := updateWorkflowTriggerSrc(db, &trigger.WorkflowDestNode); err != nil {
		return sdk.WrapError(err, "insertTrigger> Unable to update node %d for trigger %d", trigger.WorkflowDestNode.ID, trigger.ID)
	}

	return nil
}

// LoadTriggers loads trigger from a node
func loadTriggers(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, node *sdk.WorkflowNode, u *sdk.User) ([]sdk.WorkflowNodeTrigger, error) {
	dbtriggers := []NodeTrigger{}
	if _, err := db.Select(&dbtriggers, "select * from workflow_node_trigger where workflow_node_id = $1 ORDER by workflow_node_trigger.id ASC", node.ID); err != nil {
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
			dest, err := loadNode(db, store, w, t.WorkflowDestNodeID, u)
			if err != nil {
				return nil, sdk.WrapError(err, "LoadTriggers> Unable to load destination node %d", t.WorkflowDestNodeID)
			}
			t.WorkflowDestNode = *dest
		}

		triggers = append(triggers, t)
	}
	return triggers, nil
}
