package workflow

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func AddResult(ctx context.Context, db *gorp.DbMap, store cache.Store, wr sdk.WorkflowRun, runResult *sdk.WorkflowRunResult) error {
	runJob, err := LoadNodeJobRun(ctx, db, store, runResult.WorkflowJobID)
	if err != nil {
		return err
	}
	if sdk.StatusIsTerminated(runJob.Status) {
		return sdk.WrapError(sdk.ErrInvalidData, "unable to add result on a terminated job")
	}

	switch runResult.Type {
	case sdk.WorkflowRunResultTypeArtifact:
		if err := verifyAddResultArtifact(runResult, wr); err != nil {
			return err
		}
	default:
		return sdk.WrapError(sdk.ErrInvalidData, "unkonwn result type %s", runResult.Type)
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() //nolint

	if err := insertResult(tx, runResult); err != nil {
		return err
	}

	return sdk.WithStack(tx.Commit())
}

func verifyAddResultArtifact(runResult *sdk.WorkflowRunResult, wr sdk.WorkflowRun) error {
	artifactRunResult, err := runResult.GetArtifact()
	if err != nil {
		return err
	}
	if err := artifactRunResult.IsValid(); err != nil {
		return err
	}
	nodeFound := false
	for _, nrs := range wr.WorkflowNodeRuns {
		if len(nrs) == 0 {
			continue
		}
		nr := nrs[0]
		if nr.ID != runResult.WorkflowNodeRunID {
			continue
		}
		nodeFound = true
		if sdk.StatusIsTerminated(nr.Status) {
			return sdk.WrapError(sdk.ErrInvalidData, "unable to add result on a terminated node run")
		}
		for _, art := range nr.Artifacts {
			if art.Name == artifactRunResult.Name {
				return sdk.WrapError(sdk.ErrConflictData, "this artifact has already been uploaded %s", art.Name)
			}
		}
		break
	}
	if !nodeFound {
		return sdk.WrapError(sdk.ErrNotFound, "unable to add result on an unknown node run")
	}
	return nil
}

func insertResult(tx gorpmapper.SqlExecutorWithTx, runResult *sdk.WorkflowRunResult) error {
	runResult.ID = sdk.UUID()
	runResult.Created = time.Now()
	dbRunResult := dbRunResult(*runResult)
	if err := tx.Insert(&dbRunResult); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}
