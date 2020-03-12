package migrate

import (
	"context"
	"database/sql"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"

	"github.com/go-gorp/gorp"
)

// CleanDuplicateNodes .
func CleanDuplicateNodes(ctx context.Context, db *gorp.DbMap) error {
	if err := cleanDuplicateNodesWNodeTrigger(ctx, db); err != nil {
		return sdk.WithStack(err)
	}
	if err := cleanDuplicateNodesWNode(ctx, db); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

func cleanDuplicateNodesWNodeTrigger(ctx context.Context, db *gorp.DbMap) error {
	query := `WITH workflowInfo AS (
		SELECT id, name, CAST(workflow_data->'node'->>'id' AS BIGINT) as rootNodeID
		FROM workflow
	),
	oldNode as (
		SELECT w_node.id as nodeID, w_node.name as nodeName, workflowInfo.id as wID, workflowInfo.name as WName
		FROM w_node
		JOIN workflowInfo ON workflowInfo.id = w_node.workflow_id
		WHERE w_node.id < workflowInfo.rootNodeID
	)
	SELECT id FROM w_node_trigger where child_node_id IN (SELECT nodeID FROM oldNode);`
	rows, err := db.Query(query)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return sdk.WithStack(err)
	}

	var idsWNodeTrigger []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close() // nolint
			return sdk.WithStack(err)
		}
		idsWNodeTrigger = append(idsWNodeTrigger, id)
	}

	if err := rows.Close(); err != nil {
		return sdk.WithStack(err)
	}

	var mError = new(sdk.MultiError)
	for _, idWNodeTrigger := range idsWNodeTrigger {
		if err := deleteFromWNodeTrigger(ctx, db, idWNodeTrigger); err != nil {
			mError.Append(err)
			log.Error(ctx, "migrate.cleanDuplicateNodesWNodeTrigger> unable to delete from wNodeTrigger %d: %v", idWNodeTrigger, err)
		}
	}

	if mError.IsEmpty() {
		return nil
	}
	return mError
}

func deleteFromWNodeTrigger(ctx context.Context, db *gorp.DbMap, idWNodeTrigger int64) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	defer tx.Rollback() // nolint

	query := "DELETE FROM w_node_trigger where id = $1"
	if _, err := db.Exec(query, idWNodeTrigger); err != nil {
		log.Error(ctx, "migrate.deleteFromWNodeTrigger> unable to delete w_node %d: %v", idWNodeTrigger, err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func cleanDuplicateNodesWNode(ctx context.Context, db *gorp.DbMap) error {
	query := `WITH workflowInfo AS (
		SELECT id, name, CAST(workflow_data->'node'->>'id' AS BIGINT) as rootNodeID
		FROM workflow
	),
	oldNode as (
		SELECT w_node.id as nodeID, w_node.name as nodeName, workflowInfo.id as wID, workflowInfo.name as WName
		FROM w_node
		JOIN workflowInfo ON workflowInfo.id = w_node.workflow_id
		WHERE w_node.id < workflowInfo.rootNodeID
	)
	SELECT id FROM w_node where id IN (SELECT nodeID FROM oldNode);`
	rows, err := db.Query(query)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return sdk.WithStack(err)
	}

	var idsWNode []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close() // nolint
			return sdk.WithStack(err)
		}
		idsWNode = append(idsWNode, id)
	}

	if err := rows.Close(); err != nil {
		return sdk.WithStack(err)
	}

	var mError = new(sdk.MultiError)
	for _, idWNode := range idsWNode {
		if err := deleteFromWNode(ctx, db, idWNode); err != nil {
			mError.Append(err)
			log.Error(ctx, "migrate.cleanDuplicateNodesWNode> unable to delete from WNode %d: %v", idWNode, err)
		}
	}

	if mError.IsEmpty() {
		return nil
	}
	return mError
}

func deleteFromWNode(ctx context.Context, db *gorp.DbMap, idWNode int64) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	defer tx.Rollback() // nolint

	query := "DELETE FROM w_node where id = $1"
	if _, err := db.Exec(query, idWNode); err != nil {
		log.Error(ctx, "migrate.deleteFromWNode> unable to delete w_node %d: %v", idWNode, err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}
