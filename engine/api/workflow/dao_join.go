package workflow

import (
	"database/sql"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func loadJoins(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, u *sdk.User) ([]sdk.WorkflowNodeJoin, error) {
	joinIDs := []int64{}
	_, err := db.Select(&joinIDs, "select id from workflow_node_join where workflow_id = $1", w.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "loadJoins> Unable to load join IDs on workflow %d", w.ID)
	}

	joins := []sdk.WorkflowNodeJoin{}
	for _, id := range joinIDs {
		j, errJ := loadJoin(db, store, w, id, u)
		if errJ != nil {
			return nil, sdk.WrapError(errJ, "loadJoins> Unable to load join %d on workflow %d", id, w.ID)
		}
		joins = append(joins, *j)
	}

	return joins, nil
}

func loadJoin(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, id int64, u *sdk.User) (*sdk.WorkflowNodeJoin, error) {
	dbjoin := Join{}
	//Load the join
	if err := db.SelectOne(&dbjoin, "select * from workflow_node_join where id = $1 and workflow_id = $2", id, w.ID); err != nil {
		return nil, sdk.WrapError(err, "loadJoin> Unable to load join %d", id)
	}
	dbjoin.WorkflowID = w.ID

	//Load sources
	if _, err := db.Select(&dbjoin.SourceNodeIDs, "select workflow_node_id from workflow_node_join_source where workflow_node_join_id = $1", id); err != nil {
		return nil, sdk.WrapError(err, "loadJoin> Unable to load join %d sources", id)
	}
	j := sdk.WorkflowNodeJoin(dbjoin)

	for _, id := range j.SourceNodeIDs {
		j.SourceNodeRefs = append(j.SourceNodeRefs, fmt.Sprintf("%d", id))
	}

	//Select triggers id
	var triggerIDs []int64
	if _, err := db.Select(&triggerIDs, "select id from workflow_node_join_trigger where  workflow_node_join_id = $1", id); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WrapError(err, "loadJoin> Unable to load join triggers id for join %d", id)
		}
		return nil, sdk.WrapError(err, "loadJoin> Unable to load join triggers id for join %d", id)
	}

	//Load trigegrs
	for _, t := range triggerIDs {
		jt, err := loadJoinTrigger(db, store, w, &j, t, u)
		if err != nil {
			return nil, sdk.WrapError(err, "loadJoin> Unable to load join trigger %d", t)
		}
		j.Triggers = append(j.Triggers, *jt)
	}

	return &j, nil
}

func loadJoinTrigger(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, node *sdk.WorkflowNodeJoin, id int64, u *sdk.User) (*sdk.WorkflowNodeJoinTrigger, error) {
	dbtrigger := JoinTrigger{}
	//Load the trigger
	if err := db.SelectOne(&dbtrigger, "select * from workflow_node_join_trigger where workflow_node_join_id = $1 and id = $2", node.ID, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "loadJoinTrigger> Unable to load trigger %d", id)
	}

	t := sdk.WorkflowNodeJoinTrigger(dbtrigger)
	//Load node destination
	if t.WorkflowDestNodeID != 0 {
		dest, err := loadNode(db, store, w, t.WorkflowDestNodeID, u)
		if err != nil {
			return nil, sdk.WrapError(err, "loadJoinTrigger> Unable to load destination node %d", t.WorkflowDestNodeID)
		}
		t.WorkflowDestNode = *dest
	}

	return &t, nil
}

func findNodeByRefInWorkflow(ref string, w *sdk.Workflow) *sdk.WorkflowNode {
	r := findNodeByRef(ref, w.Root)
	if r != nil {
		return r
	}

	for i := range w.Joins {
		j := &w.Joins[i]
		for ti := range j.Triggers {
			t := &j.Triggers[ti]
			r := findNodeByRef(ref, &t.WorkflowDestNode)
			if r != nil {
				return r
			}
		}
	}

	return nil
}

func findNodeByRef(ref string, n *sdk.WorkflowNode) *sdk.WorkflowNode {
	log.Debug("findNodeByRef> finding node %s in node %d (%s) on %s", ref, n.ID, n.Ref, n.Pipeline.Name)
	if n.Ref == ref {
		return n
	}
	for _, t := range n.Triggers {
		n1 := findNodeByRef(ref, &t.WorkflowDestNode)
		if n1 != nil {
			return n1
		}
	}
	return nil
}

func insertJoin(db gorp.SqlExecutor, w *sdk.Workflow, n *sdk.WorkflowNodeJoin, u *sdk.User) error {
	log.Debug("insertOrUpdateJoin> %#v", n)
	n.WorkflowID = w.ID
	n.ID = 0
	n.SourceNodeIDs = nil
	dbJoin := Join(*n)

	//Check references to sources
	if len(n.SourceNodeIDs) == 0 {
		if len(n.SourceNodeRefs) == 0 {
			return sdk.WrapError(sdk.ErrWorkflowNodeRef, "insertOrUpdateJoin> Invalid joins references")
		}

		for _, s := range n.SourceNodeRefs {
			//Search references
			var foundRef = findNodeByRefInWorkflow(s, w)
			if foundRef == nil {
				return sdk.WrapError(sdk.ErrWorkflowNodeRef, "insertOrUpdateJoin> Invalid joins references")
			}
			log.Debug("insertOrUpdateJoin> Found reference %s : %d on %s", s, foundRef.ID, foundRef.Pipeline.Name)
			if foundRef.ID == 0 {
				log.Debug("insertOrUpdateJoin> insert or update reference node (%s) %d on %s", s, foundRef.ID, foundRef.Pipeline.Name)
				if err := insertNode(db, w, foundRef, u, true); err != nil {
					return sdk.WrapError(sdk.ErrWorkflowNodeRef, "insertOrUpdateJoin> Unable to insert or update source node")
				}
			}
			n.SourceNodeIDs = append(n.SourceNodeIDs, foundRef.ID)
		}
	}

	//Insert the join
	if err := db.Insert(&dbJoin); err != nil {
		return sdk.WrapError(err, "insertOrUpdateJoin> Unable to insert workflow node join")
	}
	n.ID = dbJoin.ID

	//Setup destination triggers
	for i := range n.Triggers {
		t := &n.Triggers[i]
		if err := insertJoinTrigger(db, w, n, t, u); err != nil {
			return sdk.WrapError(err, "insertOrUpdateJoin> Unable to insert or update join trigger")
		}
	}

	//Insert associations with sources
	query := "insert into workflow_node_join_source(workflow_node_id, workflow_node_join_id) values ($1, $2)"
	for _, source := range n.SourceNodeIDs {
		if _, err := db.Exec(query, source, n.ID); err != nil {
			return sdk.WrapError(err, "insertOrUpdateJoin> Unable to insert associations between node %d and join %d", source, n.ID)
		}
	}

	return nil
}

func insertJoinTrigger(db gorp.SqlExecutor, w *sdk.Workflow, j *sdk.WorkflowNodeJoin, trigger *sdk.WorkflowNodeJoinTrigger, u *sdk.User) error {
	trigger.WorkflowNodeJoinID = j.ID
	trigger.ID = 0

	//Setup destination node
	if err := insertNode(db, w, &trigger.WorkflowDestNode, u, false); err != nil {
		return sdk.WrapError(err, "insertOrUpdateJoinTrigger> Unable to setup destination node")
	}
	trigger.WorkflowDestNodeID = trigger.WorkflowDestNode.ID

	//Insert trigger
	dbt := JoinTrigger(*trigger)
	if err := db.Insert(&dbt); err != nil {
		return sdk.WrapError(err, "insertOrUpdateJoinTrigger> Unable to insert trigger")
	}
	trigger.ID = dbt.ID
	trigger.WorkflowDestNode.TriggerJoinSrcID = trigger.ID

	// Update node trigger ID
	if err := updateWorkflowTriggerJoinSrc(db, &trigger.WorkflowDestNode); err != nil {
		return sdk.WrapError(err, "insertTrigger> Unable to update node %d for trigger %d", trigger.WorkflowDestNode.ID, trigger.ID)
	}

	return nil
}

func deleteJoin(db gorp.SqlExecutor, n sdk.WorkflowNodeJoin) error {
	j := Join(n)
	if _, err := db.Delete(&j); err != nil {
		return sdk.WrapError(err, "deleteJoin> Unable to delete join %d", j.ID)
	}
	return nil
}
