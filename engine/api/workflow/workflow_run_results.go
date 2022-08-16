package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	art "github.com/ovh/cds/contrib/integrations/artifactory"
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

	// We don't check duplicate filename duplicates for artifact manager
	if runResultCheck.ResultType == sdk.WorkflowRunResultTypeArtifactManager {
		return true, nil
	}

	// Check File Name
	runResults, err := LoadRunResultsByRunIDAndType(ctx, db, runResultCheck.RunID, runResultCheck.ResultType)
	if err != nil {
		return false, sdk.WrapError(err, "unable to load run results for run %d", runResultCheck.RunID)
	}
	for _, runResult := range runResults {
		var fileName string
		switch runResultCheck.ResultType {
		case sdk.WorkflowRunResultTypeArtifact:
			refArt, err := runResult.GetArtifact()
			if err != nil {
				return false, err
			}
			fileName = refArt.Name
		case sdk.WorkflowRunResultTypeCoverage:
			refCov, err := runResult.GetCoverage()
			if err != nil {
				return false, err
			}
			fileName = refCov.Name
		case sdk.WorkflowRunResultTypeStaticFile:
			refArt, err := runResult.GetStaticFile()
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
			if nr.ID != runResult.WorkflowNodeRunID {
				continue
			}
			previousNodeRunUpload = &nr
			break
		}
		if previousNodeRunUpload == nil {
			return false, sdk.WrapError(sdk.ErrConflictData, "artifact %s has already been uploaded from another pipeline", runResultCheck.Name)
		}

		// Check Sub num
		if runResult.SubNum == nrs[0].SubNumber {
			return false, sdk.WrapError(sdk.ErrConflictData, "artifact %s has already been uploaded", runResultCheck.Name)
		}
		if runResult.SubNum > nrs[0].SubNumber {
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
func verifyAddResultArtifactManager(ctx context.Context, db gorp.SqlExecutor, store cache.Store, wr *sdk.WorkflowRun, newRunResult *sdk.WorkflowRunResult) (string, error) {
	artNewResult, err := newRunResult.GetArtifactManager()
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
	fileInfo, err := artifactClient.GetFileInfo(artNewResult.RepoName, artNewResult.Path)
	if err != nil {
		return "", err
	}
	artNewResult.Size = fileInfo.Size
	artNewResult.MD5 = fileInfo.Checksums.Md5
	artNewResult.RepoType = fileInfo.Type
	if artNewResult.FileType == "" {
		artNewResult.FileType = artNewResult.RepoType
	}

	if err := artNewResult.IsValid(); err != nil {
		return "", err
	}
	dataUpdated, _ := json.Marshal(artNewResult)
	newRunResult.DataRaw = dataUpdated

	// Check existing run-result duplicates
	var nrs []sdk.WorkflowNodeRun
	for _, nodeRuns := range wr.WorkflowNodeRuns {
		if len(nodeRuns) < 1 {
			continue
		}
		// Get last noderun
		nodeRun := nodeRuns[0]
		if nodeRun.ID != newRunResult.WorkflowNodeRunID {
			continue
		}
		nrs = nodeRuns
	}
	runResults, err := LoadRunResultsByRunIDAndType(ctx, db, wr.ID, newRunResult.Type)
	if err != nil {
		return "", sdk.WrapError(err, "unable to load run results for run %d", wr.ID)
	}
	for _, runResult := range runResults {
		artRunResult, _ := runResult.GetArtifactManager()
		// if name is different: no problem
		if artRunResult.Name != artNewResult.Name {
			continue
		}
		// if name is the same but type is different: no problem
		if artRunResult.RepoType != artNewResult.RepoType {
			continue
		}
		// It can also be a new run
		var previousNodeRunUpload *sdk.WorkflowNodeRun
		for _, nr := range nrs {
			if nr.ID != runResult.WorkflowNodeRunID {
				continue
			}
			previousNodeRunUpload = &nr
			break
		}
		if previousNodeRunUpload == nil {
			return "", sdk.NewErrorFrom(sdk.ErrConflictData, "run-result %s has already been created from another pipeline", artNewResult.Name)
		}
		// Check Sub num
		if runResult.SubNum == nrs[0].SubNumber {
			return "", sdk.NewErrorFrom(sdk.ErrConflictData, "run-result %s has already been created", artNewResult.Name)
		}
		if runResult.SubNum > nrs[0].SubNumber {
			return "", sdk.NewErrorFrom(sdk.ErrConflictData, "run-result %s cannot be created into a previous run", artNewResult.Name)
		}
	}

	cacheKey := GetRunResultKey(newRunResult.WorkflowRunID, newRunResult.Type, artNewResult.Name)
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

func getAll(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) (sdk.WorkflowRunResults, error) {
	var dbResults []dbRunResult
	if err := gorpmapping.GetAll(ctx, db, query, &dbResults); err != nil {
		return nil, err
	}
	results := make(sdk.WorkflowRunResults, 0, len(dbResults))
	for _, r := range dbResults {
		results = append(results, sdk.WorkflowRunResult(r))
	}
	return results, nil
}

func LoadRunResultsByRunID(ctx context.Context, db gorp.SqlExecutor, runID int64) (sdk.WorkflowRunResults, error) {
	query := gorpmapping.NewQuery("SELECT * FROM workflow_run_result WHERE workflow_run_id = $1 ORDER BY sub_num DESC").Args(runID)
	return getAll(ctx, db, query)
}

func LoadRunResultsByRunIDUnique(ctx context.Context, db gorp.SqlExecutor, runID int64) (sdk.WorkflowRunResults, error) {
	query := gorpmapping.NewQuery("SELECT * FROM workflow_run_result WHERE workflow_run_id = $1 ORDER BY sub_num DESC").Args(runID)
	rs, err := getAll(ctx, db, query)
	if err != nil {
		return nil, err
	}
	return rs.Unique()
}

func LoadRunResultsByNodeRunID(ctx context.Context, db gorp.SqlExecutor, nodeRunID int64) (sdk.WorkflowRunResults, error) {
	query := gorpmapping.NewQuery("SELECT * FROM workflow_run_result WHERE workflow_node_run_id = $1").Args(nodeRunID)
	return getAll(ctx, db, query)
}

func LoadRunResultsByRunIDAndType(ctx context.Context, db gorp.SqlExecutor, runID int64, t sdk.WorkflowRunResultType) (sdk.WorkflowRunResults, error) {
	query := gorpmapping.NewQuery("SELECT * FROM workflow_run_result WHERE workflow_run_id = $1 AND type = $2").Args(runID, t)
	return getAll(ctx, db, query)
}

func ResyncWorkflowRunResultsRoutine(ctx context.Context, DBFunc func() *gorp.DbMap) {
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting ResyncWorkflowRunResultsRoutine: %v", ctx.Err())
			}
			return
		case <-tick.C:
			db := DBFunc()
			if db != nil {
				id, err := FindOldestWorkflowRunWithResultToSync(ctx, DBFunc())
				if err != nil {
					log.ErrorWithStackTrace(ctx, err)
					continue
				}
				if id == 0 {
					continue
				}
				if err := SyncRunResultArtifactManagerByRunID(ctx, DBFunc(), id); err != nil {
					log.ErrorWithStackTrace(ctx, err)
					continue
				}
			}
		}
	}
}

func FindOldestWorkflowRunWithResultToSync(ctx context.Context, dbmap *gorp.DbMap) (int64, error) {
	result, err := dbmap.SelectInt("select distinct workflow_run_id from workflow_run_result where sync is NULL order by workflow_run_id asc limit 1")
	if err != nil {
		return 0, err
	}
	return result, nil
}

func SyncRunResultArtifactManagerByRunID(ctx context.Context, dbmap *gorp.DbMap, id int64) error {
	log.Info(ctx, "sync run result for workflow run id %d", id)
	db, err := dbmap.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer db.Rollback() // nolint

	wr, err := LoadAndLockRunByID(ctx, db, id, LoadRunOptions{})
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, log.Field("action_metadata_project_key"), wr.Workflow.ProjectKey)
	ctx = context.WithValue(ctx, log.Field("action_metadata_workflow_name"), wr.Workflow.Name)
	ctx = context.WithValue(ctx, log.Field("action_metadata_number"), wr.Number)

	allRunResults, err := LoadRunResultsByRunID(ctx, db, wr.ID)
	if err != nil {
		return err
	}

	log.Debug(ctx, "%d run results found", allRunResults)

	var runResults sdk.WorkflowRunResults
	for i := range allRunResults {
		result := allRunResults[i]
		// If the result is not an artifact manager, we do nothing but we consider it as synchronized
		if result.Type != sdk.WorkflowRunResultTypeArtifactManager {
			dbResult := dbRunResult(result)
			dbResult.DataSync = new(sdk.WorkflowRunResultSync)
			dbResult.DataSync.Link = ""
			dbResult.DataSync.Sync = true
			if err := gorpmapping.Update(db, &dbResult); err != nil {
				return err
			}
		} else {
			runResults = append(runResults, result)
		}
	}

	log.Debug(ctx, "%d run results to sync on run %d", len(runResults), id)

	var artifactManagerInteg *sdk.WorkflowProjectIntegration
	for i := range wr.Workflow.Integrations {
		if wr.Workflow.Integrations[i].ProjectIntegration.Model.ArtifactManager {
			artifactManagerInteg = &wr.Workflow.Integrations[i]
		}
	}
	if artifactManagerInteg == nil {
		return sdk.WrapError(sdk.ErrNotFound, "artifact manager integration is not found for this workflow")
	}

	log.Info(ctx, "artifact manager %q found for workflow run", artifactManagerInteg.ProjectIntegration.Name)

	var (
		rtName                = artifactManagerInteg.ProjectIntegration.Config[sdk.ArtifactoryConfigPlatform].Value
		rtURL                 = artifactManagerInteg.ProjectIntegration.Config[sdk.ArtifactoryConfigURL].Value
		buildInfoPrefix       = artifactManagerInteg.ProjectIntegration.Config[sdk.ArtifactoryConfigBuildInfoPrefix].Value
		tokenName             = artifactManagerInteg.ProjectIntegration.Config[sdk.ArtifactoryConfigTokenName].Value
		lowMaturitySuffix     = artifactManagerInteg.ProjectIntegration.Config[sdk.ArtifactoryConfigRepositoryPrefix].Value
		artifactoryProjectKey = artifactManagerInteg.ProjectIntegration.Config[sdk.ArtifactoryConfigProjectKey].Value
	)

	// Load the token from secrets
	secrets, err := LoadDecryptSecrets(ctx, db, wr, nil)
	if err != nil {
		return err
	}

	var rtToken string
	for _, s := range secrets {
		log.Debug(ctx, "checking secret %q", s.Name)
		if s.Name == fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigToken) {
			rtToken = string(s.Value)
			break
		}
	}
	if rtToken == "" {
		return sdk.NewErrorFrom(sdk.ErrNotFound, "unable to find artifact manager token")
	}

	// Instanciate artifactory client
	artifactClient, err := artifact_manager.NewClient(rtName, rtURL, rtToken)
	if err != nil {
		return err
	}

	version := fmt.Sprintf("%d", wr.Number)
	if wr.Version != nil {
		version = *wr.Version
	}

	parameters := wr.GetAllParameters()

	// Compute git url
	gitUrl := parameters["git.url"][0]
	if gitUrl == "" {
		gitUrl = parameters["git.http_url"][0]
	}

	nodeRunURL := parameters["cds.ui.pipeline.run"][0]
	runURL := nodeRunURL[0:strings.Index(nodeRunURL, "/node/")]

	buildInfoRequest, err := art.PrepareBuildInfo(artifactClient, art.BuildInfoRequest{
		BuildInfoPrefix:   buildInfoPrefix,
		ProjectKey:        wr.Workflow.ProjectKey,
		WorkflowName:      wr.Workflow.Name,
		Version:           version,
		AgentName:         "cds-api",
		TokenName:         tokenName,
		RunURL:            runURL,
		GitBranch:         parameters["git.branch"][0],
		GitMessage:        parameters["git.message"][0],
		GitURL:            gitUrl,
		GitHash:           parameters["git.hash"][0],
		RunResults:        runResults,
		LowMaturitySuffix: lowMaturitySuffix,
	})
	if err != nil {
		return err
	}

	log.Debug(ctx, "artifact manager build info request: %+v", buildInfoRequest)
	log.Info(ctx, "Creating Artifactory Build %s %s on project %s...\n", buildInfoRequest.Name, buildInfoRequest.Number, artifactoryProjectKey)
	if err := artifactClient.DeleteBuild(artifactoryProjectKey, buildInfoRequest.Name, buildInfoRequest.Number); err != nil {
		return err
	}

	var nbAttempts int
	for {
		nbAttempts++
		err := artifactClient.PublishBuildInfo(artifactoryProjectKey, buildInfoRequest)
		if err == nil {
			break
		} else if nbAttempts >= 3 {
			return err
		} else {
			log.Error(ctx, "error while pushing buildinfo %s %s. Retrying...\n", buildInfoRequest.Name, buildInfoRequest.Number)
		}
	}

	for _, result := range runResults {
		dbResult := dbRunResult(result)
		dbResult.DataSync = new(sdk.WorkflowRunResultSync)
		dbResult.DataSync.Link = buildInfoRequest.Name + "/" + buildInfoRequest.Number
		dbResult.DataSync.Sync = true
		if err := gorpmapping.Update(db, &dbResult); err != nil {
			return err
		}
	}

	return sdk.WithStack(db.Commit())
}
