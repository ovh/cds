package workflow_v2

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func getAllRunJobInfos(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.V2WorkflowRunJobInfo, error) {
	var dbWkfRunJobInfos []dbWorkflowRunJobInfo
	if err := gorpmapping.GetAll(ctx, db, query, &dbWkfRunJobInfos); err != nil {
		return nil, err
	}
	runJobInfos := make([]sdk.V2WorkflowRunJobInfo, 0, len(dbWkfRunJobInfos))
	for _, inf := range dbWkfRunJobInfos {
		runJobInfos = append(runJobInfos, inf.V2WorkflowRunJobInfo)
	}
	return runJobInfos, nil
}

func InsertRunJobInfo(ctx context.Context, db gorpmapper.SqlExecutorWithTx, info *sdk.V2WorkflowRunJobInfo) error {
	info.ID = sdk.UUID()
	info.IssuedAt = time.Now()
	dbWkfRunInfos := &dbWorkflowRunJobInfo{V2WorkflowRunJobInfo: *info}

	if err := gorpmapping.Insert(db, dbWkfRunInfos); err != nil {
		return err
	}
	*info = dbWkfRunInfos.V2WorkflowRunJobInfo
	return nil
}

func LoadRunJobInfosByRunID(ctx context.Context, db gorp.SqlExecutor, runJobID string) ([]sdk.V2WorkflowRunJobInfo, error) {
	query := gorpmapping.NewQuery("SELECT * from v2_workflow_run_job_info WHERE workflow_run_job_id = $1").Args(runJobID)
	return getAllRunJobInfos(ctx, db, query)
}
