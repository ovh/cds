package workflow

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// CountPipeline Count the number of workflow that use the given pipeline
func CountPipeline(db gorp.SqlExecutor, pipelineID int64) (bool, error) {
	query := `SELECT count(1) FROM w_node_context WHERE pipeline_id= $1`
	nbWorkfow := -1
	err := db.QueryRow(query, pipelineID).Scan(&nbWorkfow)
	return nbWorkfow != 0, err
}

// DeleteWorkflowData delete the relation representation of the workflow
func DeleteWorkflowData(db gorp.SqlExecutor, wf sdk.Workflow) error {
	log.Debug("DeleteWorkflowData> deleting workflow data %d", wf.ID)

	// Delete all JOINs
	for _, j := range wf.WorkflowData.Joins {
		if err := deleteJoinData(db, j); err != nil {
			return sdk.WrapError(err, "DeleteWorkflowData> unable to delete all join on workflow(%d)", wf.ID)
		}
	}

	//Delete root
	if err := deleteNodeData(db, wf.WorkflowData.Node); err != nil {
		return sdk.WrapError(err, "DeleteWorkflowData> Unable to delete workflow root")
	}

	return nil
}

func deleteJoinData(db gorp.SqlExecutor, n sdk.Node) error {
	j := dbNodeData(n)
	if _, err := db.Delete(&j); err != nil {
		return sdk.WrapError(err, "deleteJoinData> Unable to delete join %d", j.ID)
	}
	return nil
}

//deleteNode deletes nodes and all its children
func deleteNodeData(db gorp.SqlExecutor, node sdk.Node) error {
	dbwn := dbNodeData(node)
	if _, err := db.Delete(&dbwn); err != nil {
		return sdk.WrapError(err, "deleteNodeData> Unable to delete node %d", dbwn.ID)
	}
	return nil
}

// InsertWorkflowData insert workflow data
func InsertWorkflowData(db gorp.SqlExecutor, w *sdk.Workflow) error {
	if errIN := insertNodeData(db, w, &w.WorkflowData.Node, false); errIN != nil {
		return sdk.WrapError(errIN, "InsertWorkflowData> Unable to insert workflow node %s", w.WorkflowData.Node.Name)
	}

	for i := range w.WorkflowData.Joins {
		j := &w.WorkflowData.Joins[i]
		if err := insertNodeData(db, w, j, false); err != nil {
			return sdk.WrapError(err, "InsertWorkflowData> Unable to insert workflow(%d) join (%#v)", w.ID, j)
		}
	}

	dbWorkflow := Workflow(*w)
	if _, err := db.Update(&dbWorkflow); err != nil {
		return sdk.WrapError(err, "InsertWorkflowData> unable to update workflow data")
	}

	return nil
}

func insertNodeData(db gorp.SqlExecutor, w *sdk.Workflow, n *sdk.Node, skipDependencies bool) error {
	log.Debug("insertNodeData> insert node %s %d (%s)", n.Name, n.ID, n.Ref)

	if !nodeNamePattern.MatchString(n.Name) {
		return sdk.WrapError(sdk.ErrInvalidNodeNamePattern, "insertNodeData> node has a wrong name %s", n.Name)
	}

	n.ID = 0
	n.WorkflowID = w.ID

	//Insert new node
	dbwn := dbNodeData(*n)
	if err := db.Insert(&dbwn); err != nil {
		return sdk.WrapError(err, "insertNodeData> Unable to insert workflow node %s-%s", n.Name, n.Ref)
	}
	n.ID = dbwn.ID

	if err := insertNodeJoinData(db, w, n); err != nil {
		return sdk.WrapError(err, "insertNodeData> Unable to insert workflow node join data")
	}

	if skipDependencies {
		return nil
	}

	if err := insertNodeContextData(db, w, n); err != nil {
		return sdk.WrapError(err, "insertNodeData> Unable to insertNodeContextData %s-%s", n.Name, n.Ref)
	}

	if err := insertNodeHookData(db, w, n); err != nil {
		return sdk.WrapError(err, "insertNodeData> Unable to insertNodeHookData %s-%s", n.Name, n.Ref)
	}

	if err := insertNodeOutGoingHookData(db, w, n); err != nil {
		return sdk.WrapError(err, "insertNodeData> Unable to insertNodeOutGoingHook %s-%s", n.Name, n.Ref)
	}

	if err := insertNodeTriggerData(db, w, n); err != nil {
		return sdk.WrapError(err, "insertNodeData> Unable to insertNodeTriggerData %s-%s", n.Name, n.Ref)
	}

	return nil
}

func (node *dbNodeData) PostDelete(db gorp.SqlExecutor) error {
	return group.DeleteAllGroupFromNode(db, node.ID)
}

func (node *dbNodeData) PostInsert(db gorp.SqlExecutor) error {
	if len(node.Groups) == 0 {
		return nil
	}

	for i, grp := range node.Groups {
		var grDB *sdk.Group
		var err error

		switch {
		case grp.Group.ID == 0:
			grDB, err = group.LoadByName(context.Background(), db, grp.Group.Name)
		case grp.Group.Name == "":
			grDB, err = group.LoadByID(context.Background(), db, grp.Group.ID)
		default:
			grDB = &grp.Group
		}
		if err != nil {
			return sdk.WrapError(err, "cannot load group %s for node %d : %s", grp.Group.Name, node.ID, node.Name)
		}
		if grDB == nil {
			return sdk.WithStack(sdk.ErrNotFound)
		}
		node.Groups[i].Group = *grDB
	}

	return group.InsertGroupsInNode(db, node.Groups, node.ID)
}
