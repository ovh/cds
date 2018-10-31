package workflow

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func insertNodeJoinData(db gorp.SqlExecutor, w *sdk.Workflow, n *sdk.Node) error {
	if n.Type != sdk.NodeTypeJoin {
		return nil
	}

	if n.JoinContext == nil || len(n.JoinContext) == 0 {
		return sdk.WrapError(sdk.ErrWorkflowNodeRef, "insertNodeJoinData> Invalid joins references")
	}

	for i := range n.JoinContext {
		j := &n.JoinContext[i]
		foundRef := w.WorkflowData.NodeByRef(j.ParentName)
		if foundRef == nil {
			return sdk.WrapError(sdk.ErrWorkflowNodeRef, "insertNodeJoinData> Invalid joins references %s", j.ParentName)
		}
		log.Debug("insertNodeJoinData> Found reference %s: %d", j.ParentName, foundRef.ID)
		if foundRef.ID == 0 {
			log.Debug("insertNodeJoinData> insertreference node (%d) %s", foundRef.ID, foundRef.Name)
			if errN := insertNodeData(db, w, foundRef, true); errN != nil {
				return sdk.WrapError(errN, "insertNodeJoinData> Unable to insert or update source node %s", foundRef.Name)
			}
		}
		j.ParentID = foundRef.ID
		j.NodeID = n.ID

		dbJoin := dbNodeJoinData(*j)
		if err := db.Insert(&dbJoin); err != nil {
			return sdk.WrapError(err, "insertNodeJoinData> Unable to insert workflow node join")
		}
		j.ID = dbJoin.ID
	}
	return nil
}
