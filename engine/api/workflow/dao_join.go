package workflow

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func loadJoins(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.Workflow, u *sdk.User, opts LoadOptions) ([]sdk.WorkflowNodeJoin, error) {
	joinIDs := []int64{}
	_, err := db.Select(&joinIDs, "select id from workflow_node_join where workflow_id = $1", w.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Unable to load join IDs on workflow %d", w.ID)
	}

	joins := []sdk.WorkflowNodeJoin{}
	for _, id := range joinIDs {
		j, errJ := loadJoin(ctx, db, store, proj, w, id, u, opts)
		if errJ != nil {
			return nil, sdk.WrapError(errJ, "loadJoins> Unable to load join %d on workflow %d", id, w.ID)
		}
		joins = append(joins, *j)
	}

	return joins, nil
}

func loadJoin(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.Workflow, id int64, u *sdk.User, opts LoadOptions) (*sdk.WorkflowNodeJoin, error) {
	dbjoin := Join{}
	//Load the join
	if err := db.SelectOne(&dbjoin, "select * from workflow_node_join where id = $1 and workflow_id = $2", id, w.ID); err != nil {
		return nil, sdk.WrapError(err, "Unable to load join %d", id)
	}
	dbjoin.WorkflowID = w.ID

	//Load sources
	if _, err := db.Select(&dbjoin.SourceNodeIDs, "select workflow_node_id from workflow_node_join_source where workflow_node_join_id = $1", id); err != nil {
		return nil, sdk.WrapError(err, "Unable to load join %d sources", id)
	}
	j := sdk.WorkflowNodeJoin(dbjoin)

	for _, id := range j.SourceNodeIDs {
		j.SourceNodeRefs = append(j.SourceNodeRefs, fmt.Sprintf("%d", id))
	}

	//Select triggers id
	var triggerIDs []int64
	if _, err := db.Select(&triggerIDs, "select id from workflow_node_join_trigger where  workflow_node_join_id = $1", id); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WrapError(err, "Unable to load join triggers id for join %d", id)
		}
		return nil, sdk.WrapError(err, "Unable to load join triggers id for join %d", id)
	}

	//Load trigegrs
	for _, t := range triggerIDs {
		jt, err := loadJoinTrigger(ctx, db, store, proj, w, &j, t, u, opts)
		if err != nil {
			return nil, sdk.WrapError(err, "Unable to load join trigger %d", t)
		}
		//If the trigger has not been found, skip it
		if jt == nil {
			log.Warning("workflow.loadJoin> Trigger id=%d not found bu referenced by join_id %d", t, id)
			continue
		}
		j.Triggers = append(j.Triggers, *jt)
	}
	j.Ref = fmt.Sprintf("%d", j.ID)

	return &j, nil
}

func loadJoinTrigger(ctx context.Context, db gorp.SqlExecutor, store cache.Store, proj *sdk.Project, w *sdk.Workflow, node *sdk.WorkflowNodeJoin, id int64, u *sdk.User, opts LoadOptions) (*sdk.WorkflowNodeJoinTrigger, error) {
	dbtrigger := JoinTrigger{}
	//Load the trigger
	if err := db.SelectOne(&dbtrigger, "select * from workflow_node_join_trigger where workflow_node_join_id = $1 and id = $2", node.ID, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Unable to load trigger %d", id)
	}

	t := sdk.WorkflowNodeJoinTrigger(dbtrigger)
	//Load node destination
	if t.WorkflowDestNodeID != 0 {
		dest, err := loadNode(ctx, db, store, proj, w, t.WorkflowDestNodeID, u, opts)
		if err != nil {
			return nil, sdk.WrapError(err, "Unable to load destination node %d", t.WorkflowDestNodeID)
		}
		t.WorkflowDestNode = *dest
	}

	return &t, nil
}

func deleteJoin(db gorp.SqlExecutor, n sdk.WorkflowNodeJoin) error {
	j := Join(n)
	if _, err := db.Delete(&j); err != nil {
		return sdk.WrapError(err, "Unable to delete join %d", j.ID)
	}
	return nil
}
