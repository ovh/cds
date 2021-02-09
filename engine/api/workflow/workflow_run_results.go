package workflow

import (
	"context"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

var (
	KeyResult = cache.Key("run", "result")
)

func GetArtifactResultKey(runID int64, artifactName string) string {
	return cache.Key(KeyResult, string(sdk.WorkflowRunResultTypeArtifact), strconv.Itoa(int(runID)), artifactName)
}

func CanUploadArtifact(ctx context.Context, db *gorp.DbMap, store cache.Store, wr sdk.WorkflowRun, artifactRef sdk.CDNArtifactAPIRef) (bool, error) {
	// Check run
	if wr.ID != artifactRef.RunID {
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
		// get last noderun
		nodeRun := nodeRuns[0]
		if nodeRun.ID != artifactRef.RunNodeID {
			continue
		}
		nrs = nodeRuns
		if sdk.StatusIsTerminated(nodeRun.Status) {
			return false, sdk.WrapError(sdk.ErrInvalidData, "unable to upload artifact on a terminated node run")
		}
	}
	if len(nrs) == 0 {
		return false, sdk.WrapError(sdk.ErrNotFound, "unable to find node run: %d", artifactRef.RunNodeID)
	}

	// Check job data
	nodeRunJob, err := LoadNodeJobRun(ctx, db, store, artifactRef.RunJobID)
	if err != nil {
		return false, err
	}
	if nodeRunJob.WorkflowNodeRunID != artifactRef.RunNodeID {
		return false, sdk.WrapError(sdk.ErrInvalidData, "invalid node run %d", artifactRef.RunNodeID)
	}
	if sdk.StatusIsTerminated(nodeRunJob.Status) {
		return false, sdk.WrapError(sdk.ErrInvalidData, "unable to upload artifact on a terminated job")
	}

	// Check artifact name
	runResults, err := LoadRunResultsByRunIDAndType(db, artifactRef.RunID, sdk.WorkflowRunResultTypeArtifact)
	for _, result := range runResults {
		refArt, err := result.GetArtifact()
		if err != nil {
			return false, err
		}
		if refArt.Name != artifactRef.ToFilename() {
			continue
		}

		// find artifact in node run history
		var previousNodeRunUpload *sdk.WorkflowNodeRun
		for _, nr := range nrs {
			if nr.ID != result.WorkflowNodeRunID {
				continue
			}
			previousNodeRunUpload = &nr
			break
		}
		if previousNodeRunUpload == nil {
			return false, sdk.WrapError(sdk.ErrConflictData, "artifact %s has already been uploaded from another pipeline", artifactRef.ArtifactName)
		}

		// Check Sub num
		if result.SubNum == nrs[0].SubNumber {
			return false, sdk.WrapError(sdk.ErrConflictData, "artifact %s has already been uploaded", artifactRef.ArtifactName)
		}
		if result.SubNum > nrs[0].SubNumber {
			return false, sdk.WrapError(sdk.ErrConflictData, "artifact %s cannot be uploaded into a previous run", artifactRef.ArtifactName)
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

func verifyAddResultArtifact(store cache.Store, runResult *sdk.WorkflowRunResult) (string, error) {
	artifactRunResult, err := runResult.GetArtifact()
	if err != nil {
		return "", err
	}
	if err := artifactRunResult.IsValid(); err != nil {
		return "", err
	}

	cacheKey := GetArtifactResultKey(runResult.WorkflowRunID, artifactRunResult.Name)
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
	if err := tx.Insert(&dbRunResult); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

func LoadRunResultsByNodeRunID(db gorp.SqlExecutor, nodeRunID int64) ([]sdk.WorkflowRunResult, error) {
	var dbResults []dbRunResult
	if _, err := db.Select(&dbResults, "SELECT * FROM workflow_run_result where workflow_node_run_id = $1", nodeRunID); err != nil {
		return nil, err
	}
	results := make([]sdk.WorkflowRunResult, 0, len(dbResults))
	for _, r := range dbResults {
		results = append(results, sdk.WorkflowRunResult(r))
	}
	return results, nil
}

func LoadRunResultsByRunIDAndType(db gorp.SqlExecutor, runID int64, t sdk.WorkflowRunResultType) ([]sdk.WorkflowRunResult, error) {
	var dbResults []dbRunResult
	if _, err := db.Select(&dbResults, "SELECT * FROM workflow_run_result where workflow_run_id = $1 AND type = $2", runID, t); err != nil {
		return nil, err
	}
	results := make([]sdk.WorkflowRunResult, 0, len(dbResults))
	for _, r := range dbResults {
		results = append(results, sdk.WorkflowRunResult(r))
	}
	return results, nil
}
