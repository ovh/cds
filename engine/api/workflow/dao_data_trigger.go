package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func insertNodeTriggerData(db gorp.SqlExecutor, w *sdk.Workflow, n *sdk.Node) error {
	if n.Triggers == nil || len(n.Triggers) == 0 {
		return nil
	}

	for i := range n.Triggers {
		t := &n.Triggers[i]
		t.ID = 0

		// Is child already exist
		if t.ChildNode.ID == 0 {
			// Create child to get its ID
			if err := insertNodeData(db, w, &t.ChildNode, false); err != nil {
				return sdk.WrapError(err, "Unable to insert destination node")
			}
		}
		t.ChildNodeID = t.ChildNode.ID
		t.ParentNodeID = n.ID

		// Create Trigger
		dbTrigger := dbNodeTriggerData(*t)
		if err := db.Insert(&dbTrigger); err != nil {
			return sdk.WrapError(err, "insertNodeTriggerData> Unable to insert workflow node trigger")
		}
		t.ID = dbTrigger.ID

		if _, err := db.Exec("UPDATE w_node SET trigger_id = $1 WHERE id = $2", t.ID, t.ChildNodeID); err != nil {
			return sdk.WrapError(err, "insertNodeTriggerData> Unable to update trigger parent")
		}
	}
	return nil
}
