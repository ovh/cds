package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/integration/artifact_manager"
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

func CanUploadRunResult(ctx context.Context, db *gorp.DbMap, store cache.Store, wr sdk.WorkflowRun, runResultCheck sdk.WorkflowRunResultCheck) (bool, error) {
	// Check run
	if wr.ID != runResultCheck.RunID {
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
		if nodeRun.ID != runResultCheck.RunNodeID {
			continue
		}
		nrs = nodeRuns
		if sdk.StatusIsTerminated(nodeRun.Status) {
			return false, sdk.WrapError(sdk.ErrInvalidData, "unable to upload artifact on a terminated node run")
		}
	}
	if len(nrs) == 0 {
		return false, sdk.WrapError(sdk.ErrNotFound, "unable to find node run: %d", runResultCheck.RunNodeID)
	}

	// Check job data
	nodeRunJob, err := LoadNodeJobRun(ctx, db, store, runResultCheck.RunJobID)
	if err != nil {
		return false, err
	}
	if nodeRunJob.WorkflowNodeRunID != runResultCheck.RunNodeID {
		return false, sdk.WrapError(sdk.ErrInvalidData, "invalid node run %d", runResultCheck.RunNodeID)
	}
	if sdk.StatusIsTerminated(nodeRunJob.Status) {
		return false, sdk.WrapError(sdk.ErrInvalidData, "unable to upload artifact on a terminated job")
	}

	// Check File Name
	runResults, err := LoadRunResultsByRunIDAndType(ctx, db, runResultCheck.RunID, runResultCheck.ResultType)
	if err != nil {
		return false, sdk.WrapError(err, "unable to load run results for run %d", runResultCheck.RunID)
	}
	for _, result := range runResults {
		var fileName string
		switch runResultCheck.ResultType {
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
		case sdk.WorkflowRunResultTypeArtifactManager:
			refArt, err := result.GetArtifactManager()
			if err != nil {
				return false, err
			}
			fileName = refArt.Name
		case sdk.WorkflowRunResultTypeStaticFile:
			refArt, err := result.GetStaticFile()
			if err != nil {
				return false, err
			}
			fileName = refArt.Name
		}

		if fileName != runResultCheck.Name {
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
			return false, sdk.WrapError(sdk.ErrConflictData, "artifact %s has already been uploaded from another pipeline", runResultCheck.Name)
		}

		// Check Sub num
		if result.SubNum == nrs[0].SubNumber {
			return false, sdk.WrapError(sdk.ErrConflictData, "artifact %s has already been uploaded", runResultCheck.Name)
		}
		if result.SubNum > nrs[0].SubNumber {
			return false, sdk.WrapError(sdk.ErrConflictData, "artifact %s cannot be uploaded into a previous run", runResultCheck.Name)
		}
	}
	return true, nil
}

func AddResult(ctx context.Context, db *gorp.DbMap, store cache.Store, wr *sdk.WorkflowRun, runResult *sdk.WorkflowRunResult) error {
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
	case sdk.WorkflowRunResultTypeArtifactManager:
		var err error
		cacheKey, err = verifyAddResultArtifactManager(ctx, db, store, wr, runResult)
		if err != nil {
			return err
		}
	case sdk.WorkflowRunResultTypeStaticFile:
		var err error
		cacheKey, err = verifyAddResultStaticFile(store, runResult)
		if err != nil {
			return err
		}
	default:
		return sdk.WrapError(sdk.ErrInvalidData, "unknown result type %s", runResult.Type)
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

// Check validity of the request + complete runResuklt with md5,size,type
func verifyAddResultArtifactManager(ctx context.Context, db gorp.SqlExecutor, store cache.Store, wr *sdk.WorkflowRun, runResult *sdk.WorkflowRunResult) (string, error) {
	artResult, err := runResult.GetArtifactManager()
	if err != nil {
		return "", err
	}

	// Check file in integration
	var artiInteg *sdk.WorkflowProjectIntegration
	for i := range wr.Workflow.Integrations {
		if !wr.Workflow.Integrations[i].ProjectIntegration.Model.ArtifactManager {
			continue
		}
		artiInteg = &wr.Workflow.Integrations[i]
	}
	if artiInteg == nil {
		return "", sdk.NewErrorFrom(sdk.ErrInvalidData, "you cannot add a artifact manager run result without an integration")
	}
	secrets, err := loadRunSecretWithDecryption(ctx, db, wr.ID, []string{fmt.Sprintf(SecretProjIntegrationContext, artiInteg.ProjectIntegrationID)})
	if err != nil {
		return "", err
	}
	var artifactManagerToken string
	for _, s := range secrets {
		if s.Name == fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigToken) {
			artifactManagerToken = string(s.Value)
			break
		}
	}
	if artifactManagerToken == "" {
		return "", sdk.NewErrorFrom(sdk.ErrNotFound, "unable to find artifact manager token")
	}
	artifactClient, err := artifact_manager.NewClient(artiInteg.ProjectIntegration.Config[sdk.ArtifactoryConfigPlatform].Value, artiInteg.ProjectIntegration.Config[sdk.ArtifactoryConfigURL].Value, artifactManagerToken)
	if err != nil {
		return "", err
	}
	fileInfo, err := artifactClient.GetFileInfo(artResult.RepoName, artResult.Path)
	if err != nil {
		return "", err
	}
	artResult.Size = fileInfo.Size
	artResult.MD5 = fileInfo.Md5
	artResult.RepoType = fileInfo.Type
	if artResult.FileType == "" {
		artResult.FileType = artResult.RepoType
	}

	if err := artResult.IsValid(); err != nil {
		return "", err
	}
	dataUpdated, _ := json.Marshal(artResult)
	runResult.DataRaw = dataUpdated

	cacheKey := GetRunResultKey(runResult.WorkflowRunID, runResult.Type, artResult.Name)
	b, err := store.Exist(cacheKey)
	if err != nil {
		return cacheKey, err
	}
	if !b {
		return cacheKey, sdk.WrapError(sdk.ErrForbidden, "unable to upload an unchecked artifact manager file")
	}
	return cacheKey, nil
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

func verifyAddResultStaticFile(store cache.Store, runResult *sdk.WorkflowRunResult) (string, error) {
	staticFileRunResult, err := runResult.GetStaticFile()
	if err != nil {
		return "", err
	}
	if err := staticFileRunResult.IsValid(); err != nil {
		return "", err
	}

	cacheKey := GetRunResultKey(runResult.WorkflowRunID, runResult.Type, staticFileRunResult.Name)
	b, err := store.Exist(cacheKey)
	if err != nil {
		return cacheKey, err
	}
	if !b {
		return cacheKey, sdk.WrapError(sdk.ErrForbidden, "unable to upload an unchecked static-file")
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
	dbQuery := `
	WITH allResults AS (
		SELECT data->>'name' AS name, sub_num, id
		FROM workflow_run_result
		WHERE workflow_run_id = $1
	),
	deduplication AS (
		SELECT distinct on (name) *
		FROM allResults
		ORDER BY name, sub_num DESC
	)
	SELECT * FROM workflow_run_result WHERE id IN (SELECT id FROM deduplication);`
	query := gorpmapping.NewQuery(dbQuery).Args(runID)
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
