package workflow_v2

import (
	"context"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/user"
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
		if dbWkfRun.Initiator == nil {
			dbWkfRun.Initiator = &sdk.V2Initiator{
				UserID:         dbWkfRun.DeprecatedUserID,
				IsAdminWithMFA: dbWkfRun.DeprecatedAdminMFA,
			}
		}
		if dbWkfRun.Initiator.UserID != "" && dbWkfRun.Initiator.User == nil { // Compatibility code
			u, err := user.LoadByID(ctx, db, dbWkfRun.Initiator.UserID, user.LoadOptions.WithContacts)
			if err != nil {
				return nil, err
			}
			dbWkfRun.Initiator.User = u.Initiator()
		}

		dbWkfRun.DeprecatedUsername = dbWkfRun.Initiator.Username()
		dbWkfRun.DeprecatedAdminMFA = dbWkfRun.Initiator.IsAdminWithMFA

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

	if dbWkfRun.Initiator == nil {
		dbWkfRun.Initiator = &sdk.V2Initiator{
			UserID:         dbWkfRun.DeprecatedUserID,
			IsAdminWithMFA: dbWkfRun.DeprecatedAdminMFA,
		}
	}

	if dbWkfRun.Initiator.UserID != "" && dbWkfRun.Initiator.User == nil { // Compatibility code
		u, err := user.LoadByID(ctx, db, dbWkfRun.Initiator.UserID, user.LoadOptions.WithContacts)
		if err != nil {
			return nil, err
		}
		dbWkfRun.Initiator.User = u.Initiator()
	}

	dbWkfRun.DeprecatedUsername = dbWkfRun.Initiator.Username()
	dbWkfRun.DeprecatedAdminMFA = dbWkfRun.Initiator.IsAdminWithMFA

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

	if wr.Initiator == nil {
		wr.Initiator = &sdk.V2Initiator{
			UserID:         wr.DeprecatedUserID,
			IsAdminWithMFA: wr.DeprecatedAdminMFA,
		}
	}

	wr.DeprecatedAdminMFA = wr.Initiator.IsAdminWithMFA
	wr.DeprecatedUserID = wr.Initiator.UserID
	if wr.Initiator.UserID != "" && wr.Initiator.User == nil { // Compat code
		u, err := user.LoadByID(ctx, db, wr.Initiator.UserID, user.LoadOptions.WithContacts)
		if err != nil {
			return err
		}
		wr.Initiator.User = u.Initiator()
	}
	wr.DeprecatedUsername = wr.Initiator.Username()

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

	if wr.Initiator == nil {
		wr.Initiator = &sdk.V2Initiator{
			UserID:         wr.DeprecatedUserID,
			IsAdminWithMFA: wr.DeprecatedAdminMFA,
		}
	}

	wr.DeprecatedAdminMFA = wr.Initiator.IsAdminWithMFA
	wr.DeprecatedUserID = wr.Initiator.UserID
	if wr.Initiator.UserID != "" && wr.Initiator.User == nil { // Compat code
		u, err := user.LoadByID(ctx, db, wr.Initiator.UserID, user.LoadOptions.WithContacts)
		if err != nil {
			return err
		}
		wr.Initiator.User = u.Initiator()
	}
	wr.DeprecatedUsername = wr.Initiator.Username()

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

func LoadRunsWorkflowRefs(ctx context.Context, db gorp.SqlExecutor, projKey string) ([]string, error) {
	var refs []string
	_, next := telemetry.Span(ctx, "LoadRunsWorkflowRefs")
	defer next()
	if _, err := db.Select(&refs, `
		SELECT DISTINCT workflow_ref
		FROM v2_workflow_run
		WHERE project_key = $1
	`, projKey); err != nil {
		return nil, sdk.WithStack(err)
	}
	return refs, nil
}

func LoadRunsGitRepositories(ctx context.Context, db gorp.SqlExecutor, projKey string) ([]string, error) {
	var names []string
	_, next := telemetry.Span(ctx, "LoadRunsGitRepositories")
	defer next()
	if _, err := db.Select(&names, `
		SELECT DISTINCT ((contexts -> 'git' ->> 'server') || '/' || (contexts -> 'git' ->> 'repository'))
		FROM v2_workflow_run
		WHERE project_key = $1 AND contexts -> 'git' ->> 'repository' IS NOT NULL
	`, projKey); err != nil {
		return nil, sdk.WithStack(err)
	}
	return names, nil
}

func LoadRunsWorkflowRepositories(ctx context.Context, db gorp.SqlExecutor, projKey string) ([]string, error) {
	var names []string
	_, next := telemetry.Span(ctx, "LoadRunsWorkflowRepositories")
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

func LoadRunsTemplates(ctx context.Context, db gorp.SqlExecutor, projKey string) ([]string, error) {
	var names []string
	_, next := telemetry.Span(ctx, "LoadRunsTemplates")
	defer next()
	if _, err := db.Select(&names, `
		SELECT DISTINCT ((contexts -> 'cds' ->> 'workflow_template_vcs_server') || '/' || (contexts -> 'cds' ->> 'workflow_template_repository') || '/' || (contexts -> 'cds' ->> 'workflow_template'))
		FROM v2_workflow_run
		WHERE project_key = $1 AND contexts -> 'cds' ->> 'workflow_template' IS NOT NULL
	`, projKey); err != nil {
		return nil, sdk.WithStack(err)
	}
	return names, nil
}

type AnnotationsFilter struct {
	Key    string         `db:"key"`
	Values pq.StringArray `db:"values"`
}

type AnnotationsFilters []AnnotationsFilter

func LoadRunsAnnotations(ctx context.Context, db gorp.SqlExecutor, projKey string) (AnnotationsFilters, error) {
	var annotations AnnotationsFilters
	_, next := telemetry.Span(ctx, "LoadRunsAnnotations")
	defer next()

	if _, err := db.Select(&annotations, `
	select annotations.key as "key", array_agg(annotations.value) as "values"
	from 
		v2_workflow_run,
		jsonb_each_text(annotations) as annotations
	where project_key = $1 AND annotations IS NOT NULL
	group  by annotations.key;
	`, projKey); err != nil {
		return nil, sdk.WithStack(err)
	}
	return annotations, nil
}

const runQueryFilters = `
	(array_length(:workflows::text[], 1) IS NULL OR (vcs_server || '/' || repository || '/' || workflow_name) = ANY(:workflows))
	AND (array_length(:actors::text[], 1) IS NULL OR username = ANY(:actors))
	AND (array_length(:status::text[], 1) IS NULL OR status = ANY(:status))
	AND (array_length(:refs::text[], 1) IS NULL OR contexts -> 'git' ->> 'ref' = ANY(:refs))
	AND (array_length(:workflow_refs::text[], 1) IS NULL OR workflow_ref = ANY(:workflow_refs))
	AND (array_length(:repositories::text[], 1) IS NULL OR ((contexts -> 'git' ->> 'server') || '/' || (contexts -> 'git' ->> 'repository')) = ANY(:repositories))
	AND (array_length(:workflow_repositories::text[], 1) IS NULL OR (vcs_server || '/' || repository) = ANY(:workflow_repositories))
	AND (array_length(:commits::text[], 1) IS NULL OR contexts -> 'git' ->> 'sha' = ANY(:commits))
	AND (array_length(:templates::text[], 1) IS NULL OR ((contexts -> 'cds' ->> 'workflow_template_vcs_server') || '/' || (contexts -> 'cds' ->> 'workflow_template_repository') || '/' || (contexts -> 'cds' ->> 'workflow_template')) = ANY(:templates))
	AND (array_length(:annotation_keys::text[], 1) IS NULL OR annotation_keys @> :annotation_keys)
	AND (array_length(:annotation_values::text[], 1) IS NULL OR annotation_values @> :annotation_values)
`

func CountAllRuns(ctx context.Context, db gorp.SqlExecutor, filters SearchRunsFilters) (int64, error) {
	_, next := telemetry.Span(ctx, "CountAllRuns")
	defer next()

	query := `SELECT COUNT(1) 
	FROM v2_workflow_run
	LEFT JOIN (
		SELECT v2_workflow_run.id, array_agg(annotation_object.key) as "annotation_keys", array_agg(annotation_object.value) as "annotation_values"
		FROM v2_workflow_run, jsonb_each_text(COALESCE(annotations, '{}'::jsonb)) as annotation_object
		GROUP BY v2_workflow_run.id
	) v2_workflow_run_annotations 
	ON  
		v2_workflow_run.id = v2_workflow_run_annotations.id 
	WHERE ` + runQueryFilters

	params := map[string]interface{}{
		"workflows":             pq.StringArray(filters.Workflows),
		"actors":                pq.StringArray(filters.Actors),
		"status":                pq.StringArray(filters.Status),
		"refs":                  pq.StringArray(filters.Refs),
		"workflow_refs":         pq.StringArray(filters.WorkflowRefs),
		"repositories":          pq.StringArray(filters.Repositories),
		"workflow_repositories": pq.StringArray(filters.WorkflowRepositories),
		"commits":               pq.StringArray(filters.Commits),
		"templates":             pq.StringArray(filters.Templates),
		"annotation_keys":       pq.StringArray(filters.AnnotationKeys),
		"annotation_values":     pq.StringArray(filters.AnnotationValues),
	}

	count, err := db.SelectInt(query, params)
	return count, sdk.WithStack(err)
}

func CountRuns(ctx context.Context, db gorp.SqlExecutor, projKey string, filters SearchRunsFilters) (int64, error) {
	_, next := telemetry.Span(ctx, "CountRuns")
	defer next()

	query := `SELECT COUNT(1) 
	FROM v2_workflow_run
	LEFT JOIN (
		SELECT v2_workflow_run.id, array_agg(annotation_object.key) as "annotation_keys", array_agg(annotation_object.value) as "annotation_values"
		FROM v2_workflow_run, jsonb_each_text(COALESCE(annotations, '{}'::jsonb)) as annotation_object
		GROUP BY v2_workflow_run.id
	) v2_workflow_run_annotations 
	ON  
		v2_workflow_run.id = v2_workflow_run_annotations.id 
	WHERE 
		project_key = :projKey AND ` + runQueryFilters

	params := map[string]interface{}{
		"projKey":               projKey,
		"workflows":             pq.StringArray(filters.Workflows),
		"actors":                pq.StringArray(filters.Actors),
		"status":                pq.StringArray(filters.Status),
		"refs":                  pq.StringArray(filters.Refs),
		"workflow_refs":         pq.StringArray(filters.WorkflowRefs),
		"repositories":          pq.StringArray(filters.Repositories),
		"workflow_repositories": pq.StringArray(filters.WorkflowRepositories),
		"commits":               pq.StringArray(filters.Commits),
		"templates":             pq.StringArray(filters.Templates),
		"annotation_keys":       pq.StringArray(filters.AnnotationKeys),
		"annotation_values":     pq.StringArray(filters.AnnotationValues),
	}

	count, err := db.SelectInt(query, params)
	return count, sdk.WithStack(err)
}

type SearchRunsFilters struct {
	Workflows            []string
	Actors               []string
	Status               []string
	Refs                 []string
	WorkflowRefs         []string
	Repositories         []string
	WorkflowRepositories []string
	Commits              []string
	Templates            []string
	AnnotationKeys       []string
	AnnotationValues     []string
}

func (s SearchRunsFilters) Lower() {
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

func SearchAllRuns(ctx context.Context, db gorp.SqlExecutor, filters SearchRunsFilters, offset, limit uint, sort string, opts ...gorpmapper.GetOptionFunc) ([]sdk.V2WorkflowRun, error) {
	ctx, next := telemetry.Span(ctx, "LoadRuns")
	defer next()

	if limit == 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	sort, err := parseSortFilter(sort)
	if err != nil {
		return nil, err
	}

	query := gorpmapping.NewQuery(`
    SELECT v2_workflow_run.*
    FROM v2_workflow_run
		LEFT JOIN (
			SELECT v2_workflow_run.id, array_agg(annotation_object.key) as "annotation_keys", array_agg(annotation_object.value) as "annotation_values"
			FROM v2_workflow_run, jsonb_each_text(COALESCE(annotations, '{}'::jsonb)) as annotation_object
			GROUP BY v2_workflow_run.id
		) v2_workflow_run_annotations 
		ON  
			v2_workflow_run.id = v2_workflow_run_annotations.id 
		WHERE ` + runQueryFilters + `			
		ORDER BY 
			CASE WHEN :sort = 'last_modified:asc' THEN last_modified END asc,
			CASE WHEN :sort = 'last_modified:desc' THEN last_modified END desc,
			CASE WHEN :sort = 'started:asc' THEN started END asc,
			CASE WHEN :sort = 'started:desc' THEN started END desc
    LIMIT :limit OFFSET :offset
	`).Args(
		map[string]interface{}{
			"workflows":             pq.StringArray(filters.Workflows),
			"actors":                pq.StringArray(filters.Actors),
			"status":                pq.StringArray(filters.Status),
			"refs":                  pq.StringArray(filters.Refs),
			"workflow_refs":         pq.StringArray(filters.WorkflowRefs),
			"repositories":          pq.StringArray(filters.Repositories),
			"workflow_repositories": pq.StringArray(filters.WorkflowRepositories),
			"commits":               pq.StringArray(filters.Commits),
			"templates":             pq.StringArray(filters.Templates),
			"annotation_keys":       pq.StringArray(filters.AnnotationKeys),
			"annotation_values":     pq.StringArray(filters.AnnotationValues),
			"sort":                  sort,
			"limit":                 limit,
			"offset":                offset,
		})

	return getRuns(ctx, db, query, opts...)
}

func SearchRuns(ctx context.Context, db gorp.SqlExecutor, projKey string, filters SearchRunsFilters, offset, limit uint, sort string, opts ...gorpmapper.GetOptionFunc) ([]sdk.V2WorkflowRun, error) {
	ctx, next := telemetry.Span(ctx, "LoadRuns")
	defer next()

	if limit == 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	sort, err := parseSortFilter(sort)
	if err != nil {
		return nil, err
	}

	query := gorpmapping.NewQuery(`
    SELECT v2_workflow_run.*
    FROM v2_workflow_run
		LEFT JOIN (
			SELECT v2_workflow_run.id, array_agg(annotation_object.key) as "annotation_keys", array_agg(annotation_object.value) as "annotation_values"
			FROM v2_workflow_run, jsonb_each_text(COALESCE(annotations, '{}'::jsonb)) as annotation_object
			GROUP BY v2_workflow_run.id
		) v2_workflow_run_annotations 
		ON  
			v2_workflow_run.id = v2_workflow_run_annotations.id 
		WHERE project_key = :projKey AND ` + runQueryFilters + `
		ORDER BY 
			CASE WHEN :sort = 'last_modified:asc' THEN last_modified END asc,
			CASE WHEN :sort = 'last_modified:desc' THEN last_modified END desc,
			CASE WHEN :sort = 'started:asc' THEN started END asc,
			CASE WHEN :sort = 'started:desc' THEN started END desc
		LIMIT :limit OFFSET :offset
	`).Args(map[string]interface{}{
		"projKey":               projKey,
		"workflows":             pq.StringArray(filters.Workflows),
		"actors":                pq.StringArray(filters.Actors),
		"status":                pq.StringArray(filters.Status),
		"refs":                  pq.StringArray(filters.Refs),
		"workflow_refs":         pq.StringArray(filters.WorkflowRefs),
		"repositories":          pq.StringArray(filters.Repositories),
		"workflow_repositories": pq.StringArray(filters.WorkflowRepositories),
		"commits":               pq.StringArray(filters.Commits),
		"templates":             pq.StringArray(filters.Templates),
		"annotation_keys":       pq.StringArray(filters.AnnotationKeys),
		"annotation_values":     pq.StringArray(filters.AnnotationValues),
		"sort":                  sort,
		"limit":                 limit,
		"offset":                offset,
	})

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

func LoadRunsUnsafeWithPagination(ctx context.Context, db gorp.SqlExecutor, offset, limit int) ([]sdk.V2WorkflowRun, error) {
	query := gorpmapping.NewQuery(`SELECT * from v2_workflow_run ORDER BY started OFFSET $1 LIMIT $2`).Args(offset, limit)
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

func DeleteRunByID(db gorp.SqlExecutor, id string) error {
	_, err := db.Exec("DELETE FROM v2_workflow_run WHERE id = $1", id)
	return sdk.WrapError(err, "unable to delete workflow run v2 with id %s", id)
}
