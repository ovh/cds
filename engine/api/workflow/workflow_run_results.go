package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/jfrog/jfrog-client-go/artifactory/services/utils"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	art "github.com/ovh/cds/contrib/integrations/artifactory"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/artifact_manager"
	"github.com/ovh/cds/sdk/telemetry"
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

// Check validity of the request + complete runResult with md5,size,type
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

	repoDetails, err := artifactClient.GetRepository(artNewResult.RepoName)
	if err != nil {
		return "", err
	}

	filePath := artNewResult.Path
	// To get FileInfo for a docker image, we have to check the manifest file
	if repoDetails.PackageType == "docker" && !strings.HasSuffix(filePath, "manifest.json") {
		filePath = path.Join(filePath, "manifest.json")
	}

	fileInfo, err := artifactClient.GetFileInfo(artNewResult.RepoName, filePath)
	if err != nil {
		return "", err
	}
	artNewResult.Size = fileInfo.Size
	artNewResult.MD5 = fileInfo.Checksums.Md5
	artNewResult.RepoType = repoDetails.PackageType
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

func LoadRunResultsByRunIDFilterByIDs(ctx context.Context, db gorp.SqlExecutor, runID int64, resultIDs ...string) (sdk.WorkflowRunResults, error) {
	ctx, end := telemetry.Span(ctx, "workflow.LoadRunResultsByRunIDFilterByIDs")
	defer end()
	query := gorpmapping.NewQuery("SELECT * FROM workflow_run_result WHERE workflow_run_id = $1 AND id = ANY($2) ORDER BY sub_num DESC").Args(runID, pq.StringArray(resultIDs))
	return getAll(ctx, db, query)
}

func LoadRunResultsByRunID(ctx context.Context, db gorp.SqlExecutor, runID int64) (sdk.WorkflowRunResults, error) {
	ctx, end := telemetry.Span(ctx, "workflow.LoadRunResultsByRunIDFilterByIDs")
	defer end()
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

func ResyncWorkflowRunResultsRoutine(ctx context.Context, DBFunc func() *gorp.DbMap, store cache.Store, delay time.Duration) {
	tick := time.NewTicker(delay)
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
				ids, err := FindOldestWorkflowRunsWithResultToSync(ctx, DBFunc())
				if err != nil {
					log.ErrorWithStackTrace(ctx, err)
					continue
				}
				for _, id := range ids {
					lockKey := cache.Key("api:resyncWorkflowRunResults", fmt.Sprintf("%d", id))
					b, err := store.Lock(lockKey, 5*time.Minute, 0, 1)
					if err != nil {
						log.ErrorWithStackTrace(ctx, err)
						continue
					}
					if !b {
						log.Debug(ctx, "api.resyncWorkflowRunResults> workflow run %d is locked in cache", id)
						continue
					}
					tx, err := DBFunc().Begin()
					if err != nil {
						log.ErrorWithStackTrace(ctx, sdk.WithStack(err))
						_ = store.Unlock(lockKey)
						continue
					}
					if err := SyncRunResultArtifactManagerByRunID(ctx, tx, id); err != nil {
						log.ErrorWithStackTrace(ctx, err)
						tx.Rollback()
						_ = store.Unlock(lockKey)
						continue
					}
					if err := tx.Commit(); err != nil {
						log.ErrorWithStackTrace(ctx, sdk.WithStack(err))
						tx.Rollback()
						_ = store.Unlock(lockKey)
						continue
					}
					_ = store.Unlock(lockKey)
				}
			}
		}
	}
}

type ArtifactSignature map[string]string

func FindOldestWorkflowRunsWithResultToSync(ctx context.Context, dbmap *gorp.DbMap) ([]int64, error) {
	var results []int64
	query := `
    select distinct workflow_run_result.workflow_run_id
    from workflow_run_result
    join workflow_node_run on workflow_node_run.id = workflow_run_result.workflow_node_run_id
    where sync is NULL
    and workflow_node_run.status in ('Success', 'Fail', 'Stopped')
    order by workflow_run_result.workflow_run_id asc
    limit 100`
	_, err := dbmap.Select(&results, query)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	return results, nil
}

func UpdateRunResult(ctx context.Context, db gorp.SqlExecutor, result *sdk.WorkflowRunResult) error {
	_, end := telemetry.Span(ctx, "workflow.UpdateRunResult")
	defer end()
	dbResult := dbRunResult(*result)
	if err := gorpmapping.Update(db, &dbResult); err != nil {
		return err
	}
	return nil
}

func SyncRunResultArtifactManagerByRunID(ctx context.Context, db gorpmapper.SqlExecutorWithTx, workflowRunID int64) error {
	ctx, end := telemetry.Span(ctx, "workflow.SyncRunResultArtifactManagerByRunID")
	defer end()
	log.Info(ctx, "Sync run results for workflow run id %d", workflowRunID)

	wr, err := LoadRunByID(ctx, db, workflowRunID, LoadRunOptions{})
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

	var runResults sdk.WorkflowRunResults
	for i := range allRunResults {
		result := allRunResults[i]
		// If the result is not an artifact manager, we do nothing but we consider it as synchronized
		if result.Type != sdk.WorkflowRunResultTypeArtifactManager {
			if result.DataSync == nil {
				result.DataSync = new(sdk.WorkflowRunResultSync)
			}
			result.DataSync.Link = ""
			result.DataSync.Sync = true
			result.DataSync.Error = ""
			if err := UpdateRunResult(ctx, db, &result); err != nil {
				return err
			}
		} else {
			runResults = append(runResults, result)
		}
	}

	// Nothing more to do with artifact manager
	if len(runResults) == 0 {
		return nil
	}

	log.Debug(ctx, "%d run results to sync on run %d", len(runResults), workflowRunID)

	handleSyncError := func(err error) error {
		log.ErrorWithStackTrace(ctx, err)
		for i := range runResults {
			result := runResults[i]
			// If the result is not an artifact manager, we do nothing but we consider it as synchronized
			if result.DataSync == nil {
				result.DataSync = new(sdk.WorkflowRunResultSync)
			}
			result.DataSync.Sync = false
			result.DataSync.Error = err.Error()
			if err := UpdateRunResult(ctx, db, &result); err != nil {
				return err
			}
		}
		return nil
	}

	var artifactManagerInteg *sdk.WorkflowProjectIntegration
	for i := range wr.Workflow.Integrations {
		if wr.Workflow.Integrations[i].ProjectIntegration.Model.ArtifactManager {
			artifactManagerInteg = &wr.Workflow.Integrations[i]
			break
		}
	}
	if artifactManagerInteg == nil {
		return handleSyncError(sdk.Errorf("artifact manager integration is not found for workflow %s/%s", wr.Workflow.ProjectKey, wr.Workflow.Name))
	}

	log.Info(ctx, "artifact manager %q found for workflow run", artifactManagerInteg.ProjectIntegration.Name)

	var (
		rtName                      = artifactManagerInteg.ProjectIntegration.Config[sdk.ArtifactoryConfigPlatform].Value
		rtURL                       = artifactManagerInteg.ProjectIntegration.Config[sdk.ArtifactoryConfigURL].Value
		buildInfoPrefix             = artifactManagerInteg.ProjectIntegration.Config[sdk.ArtifactoryConfigBuildInfoPrefix].Value
		tokenName                   = artifactManagerInteg.ProjectIntegration.Config[sdk.ArtifactoryConfigTokenName].Value
		lowMaturitySuffixFromConfig = artifactManagerInteg.ProjectIntegration.Config[sdk.ArtifactoryConfigPromotionLowMaturity].Value
		artifactoryProjectKey       = artifactManagerInteg.ProjectIntegration.Config[sdk.ArtifactoryConfigProjectKey].Value
	)

	// Load the token from secrets
	secrets, err := LoadDecryptSecrets(ctx, db, wr, wr.RootRun())
	if err != nil {
		return err
	}

	var rtToken string
	for _, s := range secrets {
		if s.Name == fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigToken) {
			rtToken = string(s.Value)
			break
		}
	}
	if rtToken == "" {
		return handleSyncError(sdk.Errorf("unable to find artifact manager %q token", artifactManagerInteg.ProjectIntegration.Name))
	}

	version := fmt.Sprintf("%d", wr.Number)
	if wr.Version != nil {
		version = *wr.Version
	}

	parameters := wr.GetAllParameters()
	// Compute git url
	var gitUrl, gitBranch, gitMessage, gitHash string
	gitUrlParam, has := parameters["git.url"]
	if has {
		gitUrl = gitUrlParam[0]
		if gitUrl == "" {
			gitUrl = parameters["git.http_url"][0]
		}
	}
	gitBranchParam, has := parameters["git.branch"]
	if has {
		gitBranch = gitBranchParam[0]
	}
	gitMessageParam, has := parameters["git.message"]
	if has {
		gitMessage = gitMessageParam[0]
	}
	gitHashParam, has := parameters["git.hash"]
	if has {
		gitHash = gitHashParam[0]
	}

	nodeRunURL := parameters["cds.ui.pipeline.run"][0]
	runURL := nodeRunURL[0:strings.Index(nodeRunURL, "/node/")]

	artiClient, err := artifact_manager.NewClient(rtName, rtURL, rtToken)
	if err != nil {
		return err
	}

	buildInfoRequest, err := art.PrepareBuildInfo(ctx, artiClient, art.BuildInfoRequest{
		BuildInfoPrefix:          buildInfoPrefix,
		ProjectKey:               wr.Workflow.ProjectKey,
		WorkflowName:             wr.Workflow.Name,
		Version:                  version,
		AgentName:                "cds-api",
		TokenName:                tokenName,
		RunURL:                   runURL,
		GitBranch:                gitBranch,
		GitMessage:               gitMessage,
		GitURL:                   gitUrl,
		GitHash:                  gitHash,
		RunResults:               runResults,
		DefaultLowMaturitySuffix: lowMaturitySuffixFromConfig,
	})
	if err != nil {
		ctx = log.ContextWithStackTrace(ctx, err)
		log.Warn(ctx, err.Error())
		return handleSyncError(sdk.Errorf("unable to prepare build info for artifact manager"))
	}

	log.Info(ctx, "Creating Artifactory Build %s %s on project %s...\n", buildInfoRequest.Name, buildInfoRequest.Number, artifactoryProjectKey)

	// Instanciate artifactory client
	artifactClient, err := artifact_manager.NewClient(rtName, rtURL, rtToken)
	if err != nil {
		return err
	}

	ctxDelete, endDelete := telemetry.Span(ctx, "artifactClient.DeleteBuild")
	if err := artifactClient.DeleteBuild(artifactoryProjectKey, buildInfoRequest.Name, buildInfoRequest.Number); err != nil {
		ctx = log.ContextWithStackTrace(ctxDelete, err)
		log.Warn(ctx, err.Error())
		endDelete()
		return handleSyncError(sdk.Errorf("unable to delete previous build info on artifact manager"))
	}
	endDelete()

	var nbAttempts int
	for {
		nbAttempts++
		_, endPublishBuildInfo := telemetry.Span(ctx, "artifactClient.PublishBuildInfo")
		err := artifactClient.PublishBuildInfo(artifactoryProjectKey, buildInfoRequest)
		endPublishBuildInfo()
		if err == nil {
			break
		} else if nbAttempts >= 3 {
			ctx = log.ContextWithStackTrace(ctx, err)
			log.Warn(ctx, err.Error())
			return handleSyncError(sdk.Errorf("unable to publish build info on artifact manager"))
		} else {
			log.Error(ctx, "error while pushing buildinfo %s %s. Retrying...\n", buildInfoRequest.Name, buildInfoRequest.Number)
		}
	}

	// Push git info as properties
	for _, result := range runResults {
		if result.Type != sdk.WorkflowRunResultTypeArtifactManager {
			continue
		}

		artifact, err := result.GetArtifactManager()
		if err != nil {
			ctx := log.ContextWithStackTrace(ctx, err)
			log.Error(ctx, "unable to get artifact from result %s: %v", result.ID, err)
			continue
		}

		maturity := lowMaturitySuffixFromConfig
		if result.DataSync != nil && result.DataSync.LatestPromotionOrRelease() != nil {
			maturity = result.DataSync.LatestPromotionOrRelease().ToMaturity
		}
		localRepository := fmt.Sprintf("%s-%s", artifact.RepoName, maturity)

		repoDetails, err := artifactClient.GetRepository(localRepository)
		if err != nil {
			log.Error(ctx, "unable to get repository %q fror result %s: %v", localRepository, result.ID, err)
			continue
		}

		// To get FileInfo for a docker image, we have to check the manifest file
		filePath := artifact.Path
		if repoDetails.PackageType == "docker" && !strings.HasSuffix(filePath, "manifest.json") {
			filePath = path.Join(filePath, "manifest.json")
		}

		fi, err := artifactClient.GetFileInfo(artifact.RepoName, filePath)
		if err != nil {
			ctx := log.ContextWithStackTrace(ctx, err)
			log.Error(ctx, "unable to get artifact info from result %s: %v", result.ID, err)
			continue
		}

		existingProperties, err := artifactClient.GetProperties(localRepository, filePath)
		if err != nil {
			ctx := log.ContextWithStackTrace(ctx, err)
			log.Error(ctx, "unable to get artifact properties from result %s: %v", result.ID, err)
			continue
		}

		if sdk.MapHasKeys(existingProperties, "cds.signature") {
			log.Debug(ctx, "artifact is already signed by cds")
			continue
		}

		// Push git properties as artifact properties
		props := utils.NewProperties()
		signedProps := make(ArtifactSignature)

		props.AddProperty("cds.project", wr.Workflow.ProjectKey)
		signedProps["cds.project"] = wr.Workflow.ProjectKey
		props.AddProperty("cds.workflow", wr.Workflow.Name)
		signedProps["cds.workflow"] = wr.Workflow.Name
		if wr.Version != nil {
			props.AddProperty("cds.version", *wr.Version)
			signedProps["cds.version"] = *wr.Version
		}
		props.AddProperty("cds.run", strconv.FormatInt(wr.Number, 10))
		signedProps["cds.run"] = strconv.FormatInt(wr.Number, 10)

		if gitUrl != "" {
			props.AddProperty("git.url", gitUrl)
		}
		signedProps["git.url"] = gitUrl
		if gitHash != "" {
			props.AddProperty("git.hash", gitHash)
		}
		signedProps["git.hash"] = gitHash
		if gitBranch != "" {
			props.AddProperty("git.branch", gitBranch)
			signedProps["git.branch"] = gitBranch
		}

		// Prepare artifact signature
		signedProps["repository"] = artifact.RepoName
		signedProps["type"] = artifact.RepoType
		signedProps["path"] = artifact.Path
		signedProps["name"] = artifact.Name
		if fi.Checksums == nil {
			log.Error(ctx, "unable to get checksums for artifact %s %s", artifact.RepoName, artifact.Path)
		} else {
			signedProps["md5"] = fi.Checksums.Md5
			signedProps["sha1"] = fi.Checksums.Sha1
			signedProps["sha256"] = fi.Checksums.Sha256
		}

		// Sign the properties with main CDS authentication key pair
		signature, err := authentication.SignJWS(signedProps, time.Now(), 0)
		if err != nil {
			ctx := log.ContextWithStackTrace(ctx, err)
			log.Error(ctx, "unable to get artifact properties from result %s: %v", result.ID, err)
			continue
		}

		log.Info(ctx, "artifact %s%s signature: %s", localRepository, artifact.Path, signature)

		props.AddProperty("cds.signature", signature)
		if err := artifactClient.SetProperties(localRepository, artifact.Path, props); err != nil {
			ctx := log.ContextWithStackTrace(ctx, err)
			log.Error(ctx, "unable to set artifact properties from result %s: %v", result.ID, err)
			continue
		}
	}

	for _, result := range runResults {
		if result.DataSync == nil {
			result.DataSync = new(sdk.WorkflowRunResultSync)
		}
		result.DataSync.Link = buildInfoRequest.Name + "/" + buildInfoRequest.Number
		result.DataSync.Sync = true
		result.DataSync.Error = ""
		if err := UpdateRunResult(ctx, db, &result); err != nil {
			return err
		}
	}

	return nil
}

func ProcessRunResultPromotionByRunID(ctx context.Context, db gorpmapper.SqlExecutorWithTx, workflowRunID int64, promotionType sdk.WorkflowRunResultPromotionType, promotionRequest sdk.WorkflowRunResultPromotionRequest) error {
	ctx, end := telemetry.Span(ctx, "workflow.ProcessRunResultPromotionByRunID")
	defer end()
	log.Info(ctx, "Process promotion for run results %v and workflow run with id %d to maturity %s",
		promotionRequest.IDs, workflowRunID, promotionRequest.ToMaturity)

	wr, err := LoadRunByID(ctx, db, workflowRunID, LoadRunOptions{})
	if err != nil {
		return err
	}

	// Retrieve results to promote
	rs, err := LoadRunResultsByRunIDFilterByIDs(ctx, db, wr.ID, promotionRequest.IDs...)
	if err != nil {
		return err
	}
	var filteredRunResults sdk.WorkflowRunResults
	for i := range rs {
		if rs[i].Type == sdk.WorkflowRunResultTypeArtifactManager {
			filteredRunResults = append(filteredRunResults, rs[i])
		}
	}
	if len(filteredRunResults) == 0 {
		log.Info(ctx, "nothing to process on workflow run %d", workflowRunID)
		return nil
	}

	// Retrieve artifact manager integration for the workflow
	var artifactManagerInteg *sdk.WorkflowProjectIntegration
	for i := range wr.Workflow.Integrations {
		if wr.Workflow.Integrations[i].ProjectIntegration.Model.ArtifactManager {
			artifactManagerInteg = &wr.Workflow.Integrations[i]
			break
		}
	}

	// If no integration was found and there are existing run results of type ArtifactManager, set an error on this results
	if artifactManagerInteg == nil {
		var err = sdk.Errorf("artifact manager integration is not found for workflow %s/%s", wr.Workflow.ProjectKey, wr.Workflow.Name)
		log.ErrorWithStackTrace(ctx, err)
		for i := range filteredRunResults {
			result := filteredRunResults[i]
			// If the result is not an artifact manager, we do nothing but we consider it as synchronized
			if result.DataSync == nil {
				result.DataSync = new(sdk.WorkflowRunResultSync)
			}
			result.DataSync.Sync = false
			result.DataSync.Error = err.Error()
			if err := UpdateRunResult(ctx, db, &result); err != nil {
				return err
			}
		}
		return nil
	}

	// If no release or promotion can be found on an run result, consider that the current maturity equals to the default low maturity from config
	lowMaturitySuffixFromConfig := artifactManagerInteg.ProjectIntegration.Config[sdk.ArtifactoryConfigPromotionLowMaturity].Value

	// Set a new promotion on each run result
	now := time.Now()
	for i := range filteredRunResults {
		result := filteredRunResults[i]
		if result.DataSync == nil {
			result.DataSync = new(sdk.WorkflowRunResultSync)
		}
		currentMaturity := lowMaturitySuffixFromConfig
		latestPromotion := result.DataSync.LatestPromotionOrRelease()
		if latestPromotion != nil {
			currentMaturity = latestPromotion.ToMaturity
		}
		switch promotionType {
		case sdk.WorkflowRunResultPromotionTypeRelease:
			result.DataSync.Releases = append(result.DataSync.Releases, sdk.WorkflowRunResultPromotion{
				Date:         now,
				FromMaturity: currentMaturity,
				ToMaturity:   promotionRequest.ToMaturity,
			})
			log.Info(ctx, "updating run result %s with release %+v", result.DataSync.Releases[len(result.DataSync.Releases)-1])
		case sdk.WorkflowRunResultPromotionTypePromote:
			result.DataSync.Promotions = append(result.DataSync.Promotions, sdk.WorkflowRunResultPromotion{
				Date:         now,
				FromMaturity: currentMaturity,
				ToMaturity:   promotionRequest.ToMaturity,
			})
			log.Info(ctx, "updating run result %s with promotion %+v", result.DataSync.Promotions[len(result.DataSync.Promotions)-1])
		}
		if err := UpdateRunResult(ctx, db, &result); err != nil {
			return err
		}
	}

	return nil
}
