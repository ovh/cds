package workflow_v2

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"
)

func getRun(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) (*sdk.V2WorkflowRun, error) {
	var dbWkfRun dbWorkflowRun
	found, err := gorpmapping.Get(ctx, db, query, &dbWkfRun)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.ErrNotFound
	}
	isValid, err := gorpmapping.CheckSignature(dbWkfRun, dbWkfRun.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "run %s: data corrupted", dbWkfRun.ID)
		return nil, sdk.ErrNotFound
	}
	return &dbWkfRun.V2WorkflowRun, nil
}

func WorkflowRunNextNumber(db gorp.SqlExecutor, repoID, workflowName string) (int64, error) {
	i, err := db.SelectInt("select v2_workflow_run_sequences_nextval($1, $2)", repoID, workflowName)
	if err != nil {
		return 0, sdk.WrapError(err, "nextRunNumber")
	}
	return i, nil

}

func InsertRun(ctx context.Context, db gorpmapper.SqlExecutorWithTx, wr *sdk.V2WorkflowRun) error {
	ctx, next := telemetry.Span(ctx, "workflow_v2.InsertRun")
	defer next()
	wr.ID = sdk.UUID()
	wr.Started = time.Now()
	wr.LastModified = time.Now()

	dbWkfRun := &dbWorkflowRun{V2WorkflowRun: *wr}
	if err := gorpmapping.InsertAndSign(ctx, db, dbWkfRun); err != nil {
		return err
	}
	*wr = dbWkfRun.V2WorkflowRun
	return nil
}

func UpdateRun(ctx context.Context, db gorpmapper.SqlExecutorWithTx, wr *sdk.V2WorkflowRun) error {
	ctx, next := telemetry.Span(ctx, "workflow_v2.UpdateRun")
	defer next()
	wr.LastModified = time.Now()
	dbWkfRun := &dbWorkflowRun{V2WorkflowRun: *wr}
	if err := gorpmapping.UpdateAndSign(ctx, db, dbWkfRun); err != nil {
		return err
	}
	*wr = dbWkfRun.V2WorkflowRun
	return nil
}

func LoadRunByID(ctx context.Context, db gorp.SqlExecutor, id string) (*sdk.V2WorkflowRun, error) {
	ctx, next := telemetry.Span(ctx, "LoadRunByID")
	defer next()
	query := gorpmapping.NewQuery("SELECT * from v2_workflow_run WHERE id = $1").Args(id)
	return getRun(ctx, db, query)
}

func LoadRunByRunNumber(ctx context.Context, db gorp.SqlExecutor, projectKey, vcsServerID, repositoryID, wfName string, runNumber int64) (*sdk.V2WorkflowRun, error) {
	query := gorpmapping.NewQuery(`
    SELECT * from v2_workflow_run
    WHERE project_key = $1 AND vcs_server_id = $2
    AND repository_id = $3 AND workflow_name = $4 AND run_number = $5`).
		Args(projectKey, vcsServerID, repositoryID, wfName, runNumber)
	return getRun(ctx, db, query)
}

func LoadCratingWorkflowRunIDs(db gorp.SqlExecutor) ([]string, error) {
	query := `
		SELECT id
		FROM v2_workflow_run
		WHERE status = $1
		LIMIT 10
	`
	var ids []string
	_, err := db.Select(&ids, query, sdk.StatusCrafting)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to load crafting v2 workflow runs")
	}
	return ids, nil
}
