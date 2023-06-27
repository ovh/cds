package workflow_v2

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func getAllRunInfos(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.V2WorkflowRunInfo, error) {
	var dbWkfRunInfos []dbWorkflowRunInfo
	if err := gorpmapping.GetAll(ctx, db, query, &dbWkfRunInfos); err != nil {
		return nil, err
	}
	runInfos := make([]sdk.V2WorkflowRunInfo, 0, len(dbWkfRunInfos))
	for _, inf := range dbWkfRunInfos {
		runInfos = append(runInfos, inf.V2WorkflowRunInfo)
	}
	return runInfos, nil
}

func InsertRunInfo(ctx context.Context, db gorpmapper.SqlExecutorWithTx, info *sdk.V2WorkflowRunInfo) error {
	ctx, next := telemetry.Span(ctx, "workflow_v2.InsertRunInfo")
	defer next()
	info.ID = sdk.UUID()
	info.IssuedAt = time.Now()
	dbWkfRunInfos := &dbWorkflowRunInfo{V2WorkflowRunInfo: *info}

	if err := gorpmapping.Insert(db, dbWkfRunInfos); err != nil {
		return err
	}
	*info = dbWkfRunInfos.V2WorkflowRunInfo
	return nil
}

func LoadRunInfosByRunID(ctx context.Context, db gorp.SqlExecutor, runID string) ([]sdk.V2WorkflowRunInfo, error) {
	query := gorpmapping.NewQuery("SELECT * from v2_workflow_run_info WHERE workflow_run_id = $1").Args(runID)
	return getAllRunInfos(ctx, db, query)
}
