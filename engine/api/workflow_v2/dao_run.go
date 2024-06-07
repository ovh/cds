package workflow_v2

import (
	"context"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"
)

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
		WHERE project_key = $1 AND contexts -> 'git' ->> 'ref' IS NOT NULL
	`, projKey); err != nil {
		return nil, sdk.WithStack(err)
	}
	return refs, nil
}

func LoadRunsRepositories(ctx context.Context, db gorp.SqlExecutor, projKey string) ([]string, error) {
	var names []string
	_, next := telemetry.Span(ctx, "LoadRunsRepositories")
	defer next()
	if _, err := db.Select(&names, `
		SELECT DISTINCT (vcs_server || '/' || repository)
		FROM v2_workflow_run
		WHERE project_key = $1
	`, projKey); err != nil {
		return nil, sdk.WithStack(err)
	}
	return names, nil
}

func CountAllRuns(ctx context.Context, db gorp.SqlExecutor, filters SearchsRunsFilters) (int64, error) {
	_, next := telemetry.Span(ctx, "CountAllRuns")
	defer next()
	count, err := db.SelectInt(`
		SELECT COUNT(1)
    FROM v2_workflow_run
    WHERE 
			(
				array_length($1::text[], 1) IS NULL OR (vcs_server || '/' || repository || '/' || workflow_name) = ANY($1)
			)
			AND (
				array_length($2::text[], 1) IS NULL OR username = ANY($2)
			)
			AND (
				array_length($3::text[], 1) IS NULL OR status = ANY($3)
			)
			AND (
				array_length($4::text[], 1) IS NULL OR contexts -> 'git' ->> 'ref' = ANY($4)
			)
			AND (
				array_length($5::text[], 1) IS NULL OR ( (contexts -> 'git' ->> 'server') || '/' || (contexts -> 'git' ->> 'repository') ) = ANY($5)
			)
			AND (
				array_length($6::text[], 1) IS NULL OR contexts -> 'git' ->> 'sha' = ANY($6)
			)
	`,
		pq.StringArray(filters.Workflows),
		pq.StringArray(filters.Actors),
		pq.StringArray(filters.Status),
		pq.StringArray(filters.Refs),
		pq.StringArray(filters.Repositories),
		pq.StringArray(filters.Commits))
	return count, sdk.WithStack(err)
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
			AND (
				array_length($6::text[], 1) IS NULL OR ( (contexts -> 'git' ->> 'server') || '/' || (contexts -> 'git' ->> 'repository') ) = ANY($6)
			)
			AND (
				array_length($7::text[], 1) IS NULL OR contexts -> 'git' ->> 'sha' = ANY($7)
			)
	`, projKey,
		pq.StringArray(filters.Workflows),
		pq.StringArray(filters.Actors),
		pq.StringArray(filters.Status),
		pq.StringArray(filters.Refs),
		pq.StringArray(filters.Repositories),
		pq.StringArray(filters.Commits))
	return count, sdk.WithStack(err)
}

type SearchsRunsFilters struct {
	Workflows    []string
	Actors       []string
	Status       []string
	Refs         []string
	Repositories []string
	Commits      []string
}

func (s SearchsRunsFilters) Lower() {
	for i := range s.Repositories {
		s.Repositories[i] = strings.ToLower(s.Repositories[i])
	}
}

func parseSortFilter(sort string) (string, error) {
	if sort == "" {
		return "started:desc", nil
	}
	splitted := strings.Split(sort, ":")
	if len(splitted) != 2 || (splitted[0] != "started" && splitted[0] != "last_modified") || (splitted[1] != "asc" && splitted[1] != "desc") {
		return "", sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given value for sort param: %q", sort)
	}
	return sort, nil
}

func SearchAllRuns(ctx context.Context, db gorp.SqlExecutor, filters SearchsRunsFilters, offset, limit uint, sort string, opts ...gorpmapper.GetOptionFunc) ([]sdk.V2WorkflowRun, error) {
	ctx, next := telemetry.Span(ctx, "LoadRuns")
	defer next()

	if limit == 0 {
		limit = 10
	}

	sort, err := parseSortFilter(sort)
	if err != nil {
		return nil, err
	}

	query := gorpmapping.NewQuery(`
    SELECT *
    FROM v2_workflow_run
    WHERE 
			(
				array_length($1::text[], 1) IS NULL OR (vcs_server || '/' || repository || '/' || workflow_name) = ANY($1)
			)
			AND (
				array_length($2::text[], 1) IS NULL OR username = ANY($2)
			)
			AND (
				array_length($3::text[], 1) IS NULL OR status = ANY($3)
			)
			AND (
				array_length($4::text[], 1) IS NULL OR contexts -> 'git' ->> 'ref' = ANY($4)
			)
			AND (
				array_length($5::text[], 1) IS NULL OR ( (contexts -> 'git' ->> 'server') || '/' || (contexts -> 'git' ->> 'repository') ) = ANY($5)
			)
			AND (
				array_length($6::text[], 1) IS NULL OR contexts -> 'git' ->> 'sha' = ANY($6)
			)
			ORDER BY 
				CASE WHEN $7 = 'last_modified:asc' THEN last_modified END asc,
				CASE WHEN $7 = 'last_modified:desc' THEN last_modified END desc,
				CASE WHEN $7 = 'started:asc' THEN started END asc,
				CASE WHEN $7 = 'started:desc' THEN started END desc
    LIMIT $8 OFFSET $9
	`).Args(pq.StringArray(filters.Workflows),
		pq.StringArray(filters.Actors),
		pq.StringArray(filters.Status),
		pq.StringArray(filters.Refs),
		pq.StringArray(filters.Repositories),
		pq.StringArray(filters.Commits),
		sort, limit, offset)

	return getRuns(ctx, db, query, opts...)
}

func SearchRuns(ctx context.Context, db gorp.SqlExecutor, projKey string, filters SearchsRunsFilters, offset, limit uint, sort string, opts ...gorpmapper.GetOptionFunc) ([]sdk.V2WorkflowRun, error) {
	ctx, next := telemetry.Span(ctx, "LoadRuns")
	defer next()

	if limit == 0 {
		limit = 10
	}

	sort, err := parseSortFilter(sort)
	if err != nil {
		return nil, err
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
			AND (
				array_length($6::text[], 1) IS NULL OR ( (contexts -> 'git' ->> 'server') || '/' || (contexts -> 'git' ->> 'repository') ) = ANY($6)
			)
			AND (
				array_length($7::text[], 1) IS NULL OR contexts -> 'git' ->> 'sha' = ANY($7)
			)
			ORDER BY 
				CASE WHEN $8 = 'last_modified:asc' THEN last_modified END asc,
				CASE WHEN $8 = 'last_modified:desc' THEN last_modified END desc,
				CASE WHEN $8 = 'started:asc' THEN started END asc,
				CASE WHEN $8 = 'started:desc' THEN started END desc
    	LIMIT $9 OFFSET $10
	`).Args(projKey,
		pq.StringArray(filters.Workflows),
		pq.StringArray(filters.Actors),
		pq.StringArray(filters.Status),
		pq.StringArray(filters.Refs),
		pq.StringArray(filters.Repositories),
		pq.StringArray(filters.Commits),
		sort, limit, offset)

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

func LoadRunIDsToDelete(ctx context.Context, db gorp.SqlExecutor) ([]string, error) {
	query := `SELECT id from v2_workflow_run WHERE retention_date < CURRENT_DATE ORDER BY started ASC LIMIT 500`
	var ids []string
	if _, err := db.Select(&ids, query); err != nil {
		return nil, err
	}
	return ids, nil
}

func LoadAndLockRunByID(ctx context.Context, db gorp.SqlExecutor, id string, opts ...gorpmapper.GetOptionFunc) (*sdk.V2WorkflowRun, error) {
	ctx, next := telemetry.Span(ctx, "LoadAndLockRunByID")
	defer next()
	query := gorpmapping.NewQuery("SELECT * from v2_workflow_run WHERE id = $1 FOR UPDATE SKIP LOCKED").Args(id)
	return getRun(ctx, db, query, opts...)
}
