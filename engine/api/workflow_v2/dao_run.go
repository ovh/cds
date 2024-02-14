package workflow_v2

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"
)

func WithRunResults(ctx context.Context, _ *gorpmapper.Mapper, db gorp.SqlExecutor, i interface{}) error {
	switch target := i.(type) {
	case *[]dbWorkflowRun:
		var ids []string
		for _, r := range *target {
			ids = append(ids, r.ID)
		}
		allResults, err := loadRunResultsByRunIDs(ctx, db, ids...)
		if err != nil {
			return err
		}
		for i := range *target {
			r := &(*target)[i]
			if results, has := allResults[r.ID]; has {
				var runResults []sdk.V2WorkflowRunResult
				for _, runResult := range results {
					if r.RunAttempt != runResult.RunAttempt {
						continue
					}
					runResults = append(runResults, runResult.V2WorkflowRunResult)
				}
				r.Results = runResults
			}
		}
	case []sdk.V2WorkflowRun:
		var ids []string
		for _, r := range target {
			ids = append(ids, r.ID)
		}
		allResults, err := loadRunResultsByRunIDs(ctx, db, ids...)
		if err != nil {
			return err
		}
		for i := range target {
			r := &target[i]
			if results, has := allResults[r.ID]; has {
				var runResults []sdk.V2WorkflowRunResult
				for _, runResult := range results {
					if r.RunAttempt != runResult.RunAttempt {
						continue
					}
					runResults = append(runResults, runResult.V2WorkflowRunResult)
				}
				r.Results = runResults
			}
		}
	case *sdk.V2WorkflowRun:
		allResults, err := loadRunResultsByRunIDs(ctx, db, target.ID)
		if err != nil {
			return err
		}
		if results, has := allResults[target.ID]; has {
			var runResults []sdk.V2WorkflowRunResult
			for _, r := range results {
				if target.RunAttempt != r.RunAttempt {
					continue
				}
				runResults = append(runResults, r.V2WorkflowRunResult)
			}
			target.Results = runResults
		}
	case *dbWorkflowRun:
		allResults, err := loadRunResultsByRunIDs(ctx, db, target.ID)
		if err != nil {
			return err
		}
		if results, has := allResults[target.ID]; has {
			var runResult []sdk.V2WorkflowRunResult
			for _, r := range results {
				if target.RunAttempt != r.RunAttempt {
					continue
				}
				runResult = append(runResult, r.V2WorkflowRunResult)
			}
			target.Results = runResult
		}
	default:
		panic(fmt.Sprintf("WithRunResults: unsupported target %T", i))
	}

	return nil
}

func LoadRunResults(ctx context.Context, db gorp.SqlExecutor, runID string) ([]sdk.V2WorkflowRunResult, error) {
	results, err := loadRunResultsByRunIDs(ctx, db, runID)
	if err != nil {
		return nil, err
	}
	if results, has := results[runID]; has {
		var runResults []sdk.V2WorkflowRunResult
		for _, r := range results {
			runResults = append(runResults, r.V2WorkflowRunResult)
		}

		return runResults, nil
	}
	return nil, nil
}

func LoadRunResult(ctx context.Context, db gorp.SqlExecutor, runID string, id string) (*sdk.V2WorkflowRunResult, error) {
	query := gorpmapping.NewQuery(`select * from v2_workflow_run_result where id = $1 AND workflow_run_id = $2`).Args(id, runID)
	var result dbV2WorkflowRunResult
	found, err := gorpmapping.Get(ctx, db, query, &result)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to load run result %v", id)
	}
	if !found {
		return nil, sdk.WrapError(sdk.ErrNotFound, "unable to run load result id=%s workflow_run_id=%s", id, runID)
	}
	return &result.V2WorkflowRunResult, nil
}

func LoadRunResultsByRunJobID(ctx context.Context, db gorp.SqlExecutor, runJobID string) ([]sdk.V2WorkflowRunResult, error) {
	query := gorpmapping.NewQuery(`select * from v2_workflow_run_result where workflow_run_job_id = $1`).Args(runJobID)
	var result []dbV2WorkflowRunResult
	if err := gorpmapping.GetAll(ctx, db, query, &result); err != nil {
		return nil, sdk.WrapError(err, "unable to load run results for run job %s", runJobID)
	}
	runResults := make([]sdk.V2WorkflowRunResult, 0, len(result))
	for _, r := range result {
		runResults = append(runResults, r.V2WorkflowRunResult)
	}
	return runResults, nil
}

func InsertRunResult(ctx context.Context, db gorp.SqlExecutor, runResult *sdk.V2WorkflowRunResult) error {
	entity := dbV2WorkflowRunResult{*runResult}
	if err := gorpmapping.Insert(db, &entity); err != nil {
		return err
	}
	log.Info(ctx, "run result %+v inserted", runResult)
	return nil
}

func UpdateRunResult(ctx context.Context, db gorp.SqlExecutor, runResult *sdk.V2WorkflowRunResult) error {
	entity := dbV2WorkflowRunResult{*runResult}
	return gorpmapping.Update(db, &entity)
}

func loadRunResultsByRunIDs(ctx context.Context, db gorp.SqlExecutor, runIDs ...string) (map[string][]dbV2WorkflowRunResult, error) {
	query := gorpmapping.NewQuery(`
  select * from v2_workflow_run_result where workflow_run_id = ANY($1::uuid[]) order by workflow_run_id, issued_at ASC
  `).Args(pq.StringArray(runIDs))
	var result []dbV2WorkflowRunResult
	if err := gorpmapping.GetAll(ctx, db, query, &result); err != nil {
		return nil, sdk.WrapError(err, "unable to load run results for %v", runIDs)
	}
	var mapRes = make(map[string][]dbV2WorkflowRunResult)
	for _, r := range result {
		res := mapRes[r.WorkflowRunID]
		res = append(res, r)
		mapRes[r.WorkflowRunID] = res
	}
	return mapRes, nil
}

func getRuns(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query, opts ...gorpmapper.GetOptionFunc) ([]sdk.V2WorkflowRun, error) {
	var dbWkfRuns []dbWorkflowRun
	if err := gorpmapping.GetAll(ctx, db, query, &dbWkfRuns, opts...); err != nil {
		return nil, err
	}
	runs := make([]sdk.V2WorkflowRun, 0, len(dbWkfRuns))
	for _, dbWkfRun := range dbWkfRuns {
		isValid, err := gorpmapping.CheckSignature(dbWkfRun, dbWkfRun.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "run %s: data corrupted", dbWkfRun.ID)
			continue
		}
		runs = append(runs, dbWkfRun.V2WorkflowRun)
	}

	return runs, nil
}

func getRun(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query, opts ...gorpmapper.GetOptionFunc) (*sdk.V2WorkflowRun, error) {
	var dbWkfRun dbWorkflowRun
	found, err := gorpmapping.Get(ctx, db, query, &dbWkfRun, opts...)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	isValid, err := gorpmapping.CheckSignature(dbWkfRun, dbWkfRun.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "run %s: data corrupted", dbWkfRun.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
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
	wr.RunAttempt = 1

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

func LoadRunsWorkflowNames(ctx context.Context, db gorp.SqlExecutor, projKey string) ([]string, error) {
	var names []string
	_, next := telemetry.Span(ctx, "LoadRunsWorkflowNames")
	defer next()
	if _, err := db.Select(&names, `
		SELECT DISTINCT (vcs_server || '/' || repository || '/' || workflow_name)
		FROM v2_workflow_run
		WHERE project_key = $1
	`, projKey); err != nil {
		return nil, sdk.WithStack(err)
	}
	return names, nil
}

func LoadRunsActors(ctx context.Context, db gorp.SqlExecutor, projKey string) ([]string, error) {
	var actors []string
	_, next := telemetry.Span(ctx, "LoadRunsActors")
	defer next()
	if _, err := db.Select(&actors, `
		SELECT DISTINCT username
		FROM v2_workflow_run
		WHERE project_key = $1
	`, projKey); err != nil {
		return nil, sdk.WithStack(err)
	}
	return actors, nil
}

func LoadRunsGitRefs(ctx context.Context, db gorp.SqlExecutor, projKey string) ([]string, error) {
	var refs []string
	_, next := telemetry.Span(ctx, "LoadRunsGitRefs")
	defer next()
	if _, err := db.Select(&refs, `
		SELECT DISTINCT contexts -> 'git' ->> 'ref'
		FROM v2_workflow_run
		WHERE project_key = $1
	`, projKey); err != nil {
		return nil, sdk.WithStack(err)
	}
	return refs, nil
}

func CountRuns(ctx context.Context, db gorp.SqlExecutor, projKey string, filters SearchsRunsFilters) (int64, error) {
	_, next := telemetry.Span(ctx, "CountRuns")
	defer next()
	count, err := db.SelectInt(`
		SELECT COUNT(1)
    FROM v2_workflow_run
    WHERE 
			project_key = $1 
			AND (
				array_length($2::text[], 1) IS NULL OR (vcs_server || '/' || repository || '/' || workflow_name) = ANY($2)
			)
			AND (
				array_length($3::text[], 1) IS NULL OR username = ANY($3)
			)
			AND (
				array_length($4::text[], 1) IS NULL OR status = ANY($4)
			)
			AND (
				array_length($5::text[], 1) IS NULL OR contexts -> 'git' ->> 'ref' = ANY($5)
			)
	`, projKey, pq.StringArray(filters.Workflows), pq.StringArray(filters.Actors), pq.StringArray(filters.Status), pq.StringArray(filters.Branches))
	return count, sdk.WithStack(err)
}

type SearchsRunsFilters struct {
	Workflows []string
	Actors    []string
	Status    []string
	Branches  []string
}

func SearchRuns(ctx context.Context, db gorp.SqlExecutor, projKey string, filters SearchsRunsFilters, offset, limit uint, opts ...gorpmapper.GetOptionFunc) ([]sdk.V2WorkflowRun, error) {
	ctx, next := telemetry.Span(ctx, "LoadRuns")
	defer next()

	if limit == 0 {
		limit = 10
	}

	query := gorpmapping.NewQuery(`
    SELECT *
    FROM v2_workflow_run
    WHERE 
			project_key = $1
			AND (
				array_length($2::text[], 1) IS NULL OR (vcs_server || '/' || repository || '/' || workflow_name) = ANY($2)
			)
			AND (
				array_length($3::text[], 1) IS NULL OR username = ANY($3)
			)
			AND (
				array_length($4::text[], 1) IS NULL OR status = ANY($4)
			)
			AND (
				array_length($5::text[], 1) IS NULL OR contexts -> 'git' ->> 'ref' = ANY($5)
			)
		ORDER BY started desc
    LIMIT $6 OFFSET $7
	`).Args(projKey, pq.StringArray(filters.Workflows), pq.StringArray(filters.Actors), pq.StringArray(filters.Status), pq.StringArray(filters.Branches), limit, offset)

	return getRuns(ctx, db, query, opts...)
}

func LoadRuns(ctx context.Context, db gorp.SqlExecutor, projKey, vcsProjectID, repoID, workflowName string, opts ...gorpmapper.GetOptionFunc) ([]sdk.V2WorkflowRun, error) {
	ctx, next := telemetry.Span(ctx, "LoadRuns")
	defer next()
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM v2_workflow_run
    WHERE project_key = $1 AND vcs_server_id = $2 AND repository_id = $3 AND workflow_name = $4 ORDER BY run_number desc
    LIMIT 50`).Args(projKey, vcsProjectID, repoID, workflowName)
	return getRuns(ctx, db, query, opts...)
}

func LoadRunByID(ctx context.Context, db gorp.SqlExecutor, id string, opts ...gorpmapper.GetOptionFunc) (*sdk.V2WorkflowRun, error) {
	ctx, next := telemetry.Span(ctx, "LoadRunByID")
	defer next()
	query := gorpmapping.NewQuery("SELECT * from v2_workflow_run WHERE id = $1").Args(id)
	return getRun(ctx, db, query, opts...)
}

func LoadRunByProjectKeyAndID(ctx context.Context, db gorp.SqlExecutor, projectKey, id string, opts ...gorpmapper.GetOptionFunc) (*sdk.V2WorkflowRun, error) {
	ctx, next := telemetry.Span(ctx, "LoadRunByProjectKeyAndID")
	defer next()
	query := gorpmapping.NewQuery("SELECT * from v2_workflow_run WHERE project_key = $1 AND id = $2").Args(projectKey, id)
	return getRun(ctx, db, query, opts...)
}

func LoadRunByRunNumber(ctx context.Context, db gorp.SqlExecutor, projectKey, vcsServerID, repositoryID, wfName string, runNumber int64, opts ...gorpmapper.GetOptionFunc) (*sdk.V2WorkflowRun, error) {
	query := gorpmapping.NewQuery(`
    SELECT * from v2_workflow_run
    WHERE project_key = $1 AND vcs_server_id = $2
    AND repository_id = $3 AND workflow_name = $4 AND run_number = $5`).
		Args(projectKey, vcsServerID, repositoryID, wfName, runNumber)
	return getRun(ctx, db, query, opts...)
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

func LoadBuildingRunWithEndedJobs(ctx context.Context, db gorp.SqlExecutor, opts ...gorpmapper.GetOptionFunc) ([]sdk.V2WorkflowRun, error) {
	query := gorpmapping.NewQuery(`
  SELECT v2_workflow_run.*
  FROM v2_workflow_run
  WHERE status = $1
  AND (
    SELECT count(1) FROM v2_workflow_run_job
    WHERE v2_workflow_run_job.workflow_run_id = v2_workflow_run.id AND v2_workflow_run_job.status = ANY($2)
  ) = 0
  LIMIT 100;
`).Args(sdk.StatusBuilding, pq.StringArray([]string{sdk.StatusBuilding, sdk.StatusScheduling, sdk.StatusWaiting}))

	return getRuns(ctx, db, query, opts...)
}

func LoadRunsUnsafe(ctx context.Context, db gorp.SqlExecutor) ([]sdk.V2WorkflowRun, error) {
	query := gorpmapping.NewQuery(`SELECT * from v2_workflow_run`)
	var dbWkfRuns []dbWorkflowRun
	if err := gorpmapping.GetAll(ctx, db, query, &dbWkfRuns); err != nil {
		return nil, err
	}
	runs := make([]sdk.V2WorkflowRun, 0, len(dbWkfRuns))
	for _, dbWkfRun := range dbWkfRuns {
		runs = append(runs, dbWkfRun.V2WorkflowRun)
	}

	return runs, nil
}
