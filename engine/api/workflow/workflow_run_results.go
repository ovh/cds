package workflow

import (
	"context"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

var (
	KeyResult = cache.Key("run", "result")
)

func GetRunResultKey(runID int64, t sdk.WorkflowRunResultType, fileName string) string {
	return cache.Key(KeyResult, string(t), strconv.Itoa(int(runID)), fileName)
}

func CanUploadRunResult(ctx context.Context, db *gorp.DbMap, store cache.Store, wr sdk.WorkflowRun, apiRef sdk.CDNRunResultAPIRef) (bool, error) {
	// Check run
	if wr.ID != apiRef.RunID {
		return false, sdk.WrapError(sdk.ErrInvalidData, "unable to upload and artifact for this run")
	}
	if sdk.StatusIsTerminated(wr.Status) {
		return false, sdk.WrapError(sdk.ErrInvalidData, "unable to upload artifact on a terminated run")
	}

	// Check node run
	var nrs []sdk.WorkflowNodeRun
	for _, nodeRuns := range wr.WorkflowNodeRuns {
		if len(nodeRuns) < 1 {
			continue
		}
		// Get last noderun
		nodeRun := nodeRuns[0]
		if nodeRun.ID != apiRef.RunNodeID {
			continue
		}
		nrs = nodeRuns
		if sdk.StatusIsTerminated(nodeRun.Status) {
			return false, sdk.WrapError(sdk.ErrInvalidData, "unable to upload artifact on a terminated node run")
		}
	}
	if len(nrs) == 0 {
		return false, sdk.WrapError(sdk.ErrNotFound, "unable to find node run: %d", apiRef.RunNodeID)
	}

	// Check job data
	nodeRunJob, err := LoadNodeJobRun(ctx, db, store, apiRef.RunJobID)
	if err != nil {
		return false, err
	}
	if nodeRunJob.WorkflowNodeRunID != apiRef.RunNodeID {
		return false, sdk.WrapError(sdk.ErrInvalidData, "invalid node run %d", apiRef.RunNodeID)
	}
	if sdk.StatusIsTerminated(nodeRunJob.Status) {
		return false, sdk.WrapError(sdk.ErrInvalidData, "unable to upload artifact on a terminated job")
	}

	// Check File Name
	runResults, err := LoadRunResultsByRunIDAndType(ctx, db, apiRef.RunID, apiRef.RunResultType)
	if err != nil {
		return false, sdk.WrapError(err, "unable to load run results for run %d", apiRef.RunID)
	}
	for _, result := range runResults {
		var fileName string
		switch apiRef.RunResultType {
		case sdk.WorkflowRunResultTypeArtifact:
			refArt, err := result.GetArtifact()
			if err != nil {
				return false, err
			}
			fileName = refArt.Name
		case sdk.WorkflowRunResultTypeCoverage:
			refCov, err := result.GetCoverage()
			if err != nil {
				return false, err
			}
			fileName = refCov.Name
		}

		if fileName != apiRef.ToFilename() {
			continue
		}

		// If we find a run result with same check, check subnumber
		var previousNodeRunUpload *sdk.WorkflowNodeRun
		for _, nr := range nrs {
			if nr.ID != result.WorkflowNodeRunID {
				continue
			}
			previousNodeRunUpload = &nr
			break
		}
		if previousNodeRunUpload == nil {
			return false, sdk.WrapError(sdk.ErrConflictData, "artifact %s has already been uploaded from another pipeline", apiRef.ArtifactName)
		}

		// Check Sub num
		if result.SubNum == nrs[0].SubNumber {
			return false, sdk.WrapError(sdk.ErrConflictData, "artifact %s has already been uploaded", apiRef.ArtifactName)
		}
		if result.SubNum > nrs[0].SubNumber {
			return false, sdk.WrapError(sdk.ErrConflictData, "artifact %s cannot be uploaded into a previous run", apiRef.ArtifactName)
		}
	}
	return true, nil
}

func AddResult(db *gorp.DbMap, store cache.Store, runResult *sdk.WorkflowRunResult) error {
	var cacheKey string
	switch runResult.Type {
	case sdk.WorkflowRunResultTypeArtifact:
		var err error
		cacheKey, err = verifyAddResultArtifact(store, runResult)
		if err != nil {
			return err
		}
	case sdk.WorkflowRunResultTypeCoverage:
		var err error
		cacheKey, err = verifyAddResultCoverage(store, runResult)
		if err != nil {
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
	if err := tx.Commit(); err != nil {
		return err
	}
	return sdk.WithStack(store.Delete(cacheKey))
}

func verifyAddResultCoverage(store cache.Store, runResult *sdk.WorkflowRunResult) (string, error) {
	coverageRunResult, err := runResult.GetCoverage()
	if err != nil {
		return "", err
	}
	if err := coverageRunResult.IsValid(); err != nil {
		return "", err
	}

	cacheKey := GetRunResultKey(runResult.WorkflowRunID, runResult.Type, coverageRunResult.Name)
	b, err := store.Exist(cacheKey)
	if err != nil {
		return cacheKey, err
	}
	if !b {
		return cacheKey, sdk.WrapError(sdk.ErrForbidden, "unable to upload an unchecked coverage")
	}
	return cacheKey, nil
}

func verifyAddResultArtifact(store cache.Store, runResult *sdk.WorkflowRunResult) (string, error) {
	artifactRunResult, err := runResult.GetArtifact()
	if err != nil {
		return "", err
	}
	if err := artifactRunResult.IsValid(); err != nil {
		return "", err
	}

	cacheKey := GetRunResultKey(runResult.WorkflowRunID, runResult.Type, artifactRunResult.Name)
	b, err := store.Exist(cacheKey)
	if err != nil {
		return cacheKey, err
	}
	if !b {
		return cacheKey, sdk.WrapError(sdk.ErrForbidden, "unable to upload an unchecked artifact")
	}
	return cacheKey, nil
}

func insertResult(tx gorpmapper.SqlExecutorWithTx, runResult *sdk.WorkflowRunResult) error {
	runResult.ID = sdk.UUID()
	runResult.Created = time.Now()
	dbRunResult := dbRunResult(*runResult)
	if err := gorpmapping.Insert(tx, &dbRunResult); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

func getAll(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.WorkflowRunResult, error) {
	var dbResults []dbRunResult
	if err := gorpmapping.GetAll(ctx, db, query, &dbResults); err != nil {
		return nil, err
	}
	results := make([]sdk.WorkflowRunResult, 0, len(dbResults))
	for _, r := range dbResults {
		results = append(results, sdk.WorkflowRunResult(r))
	}
	return results, nil
}

func LoadRunResultsByRunID(ctx context.Context, db gorp.SqlExecutor, runID int64) ([]sdk.WorkflowRunResult, error) {
	query := gorpmapping.NewQuery("SELECT * FROM workflow_run_result where workflow_run_id = $1").Args(runID)
	return getAll(ctx, db, query)
}

func LoadRunResultsByNodeRunID(ctx context.Context, db gorp.SqlExecutor, nodeRunID int64) ([]sdk.WorkflowRunResult, error) {
	query := gorpmapping.NewQuery("SELECT * FROM workflow_run_result where workflow_node_run_id = $1").Args(nodeRunID)
	return getAll(ctx, db, query)
}

func LoadRunResultsByRunIDAndType(ctx context.Context, db gorp.SqlExecutor, runID int64, t sdk.WorkflowRunResultType) ([]sdk.WorkflowRunResult, error) {
	query := gorpmapping.NewQuery("SELECT * FROM workflow_run_result where workflow_run_id = $1 AND type = $2").Args(runID, t)
	return getAll(ctx, db, query)

}
