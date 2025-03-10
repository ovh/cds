package search

import (
	"context"
	"database/sql"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

type SearchFilters struct {
	Projects []string
	Types    []string
	Query    string
}

func CountAll(ctx context.Context, db gorp.SqlExecutor, filters SearchFilters) (int64, error) {
	_, next := telemetry.Span(ctx, "CountAll")
	defer next()

	query := `
		WITH 
			results AS (
				(
					SELECT 'project' AS type, projectkey AS id, name AS label
					FROM project
					WHERE
						projectkey = ANY(:projects)
						AND (array_length(:types::text[], 1) IS NULL OR 'project' = ANY(:types))
				)
				UNION
				(
					SELECT 'workflow' AS type, CONCAT(entity.project_key, '/', vcs_project.name, '/', project_repository.name, '/', entity.name) AS id, entity.name AS label
					FROM entity
					JOIN project_repository ON entity.project_repository_id = project_repository.id
					JOIN vcs_project ON project_repository.vcs_project_id = vcs_project.id
					WHERE
						entity.type = 'Workflow'
						AND entity.commit = 'HEAD'
						AND entity.project_key = ANY(:projects)
						AND (array_length(:types::text[], 1) IS NULL OR 'workflow' = ANY(:types))
					GROUP BY entity.project_key, vcs_project.name, project_repository.name, entity.type, entity.name
				)
				UNION
				(
					SELECT 'workflow-legacy' AS type, CONCAT(project.projectkey, '/', workflow.name) AS id, workflow.name AS label
					FROM workflow
					JOIN project ON project.id = workflow.project_id
					WHERE
						project.projectkey = ANY(:projects)
						AND (array_length(:types::text[], 1) IS NULL OR 'workflow-legacy' = ANY(:types))
				)
			)
		SELECT COUNT(1)
		FROM results
		WHERE LOWER(label) LIKE :query OR LOWER(id) LIKE :query
	`

	count, err := db.SelectInt(query, map[string]interface{}{
		"projects": pq.StringArray(filters.Projects),
		"types":    pq.StringArray(filters.Types),
		"query":    "%" + strings.ToLower(filters.Query) + "%",
	})
	return count, sdk.WithStack(err)
}

func SearchAll(ctx context.Context, db gorp.SqlExecutor, filters SearchFilters, offset, limit uint) (sdk.SearchResults, error) {
	_, next := telemetry.Span(ctx, "SearchAll")
	defer next()

	if limit == 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	query := `
		WITH 
			results AS (
				(
					SELECT 0 AS type_int, projectkey AS id, name AS label, description, null AS variants
					FROM project
					WHERE
						projectkey = ANY(:projects)
						AND (array_length(:types::text[], 1) IS NULL OR 'project' = ANY(:types))
				)
				UNION
				(
					WITH 
						entities AS (
							SELECT CONCAT(entity.project_key, '/', vcs_project.name, '/', project_repository.name, '/', entity.name) AS id, entity.name AS label, entity.ref AS ref, entity.last_update AS last_update
							FROM entity
							JOIN project_repository ON entity.project_repository_id = project_repository.id
							JOIN vcs_project ON project_repository.vcs_project_id = vcs_project.id
							WHERE
								entity.type = 'Workflow'
								AND entity.commit = 'HEAD'
								AND entity.project_key = ANY(:projects)
								AND (array_length(:types::text[], 1) IS NULL OR 'workflow' = ANY(:types))
							ORDER BY last_update DESC
						)
					SELECT 1 AS type_int, id, label, '' AS description, jsonb_agg(ref) AS variants
					FROM entities
					GROUP BY id, label
				)
				UNION
				(
					SELECT 2 AS type_int, CONCAT(project.projectkey, '/', workflow.name) AS id, workflow.name AS label, workflow.description AS description, null AS variants
					FROM workflow
					JOIN project ON project.id = workflow.project_id
					WHERE
						project.projectkey = ANY(:projects)
						AND (array_length(:types::text[], 1) IS NULL OR 'workflow-legacy' = ANY(:types))
				)
			)
		SELECT id, label, description, variants, CASE
				WHEN LOWER(label) LIKE :query THEN 1
				WHEN LOWER(id) LIKE :query THEN 2
			END AS priority, CASE
				WHEN type_int = 0 THEN 'project'
				WHEN type_int = 1 THEN 'workflow'
				WHEN type_int = 2 THEN 'workflow-legacy'
			END AS type
		FROM results
		WHERE LOWER(label) LIKE :query OR LOWER(id) LIKE :query
		ORDER BY priority ASC, CHAR_LENGTH(label) ASC, type_int ASC
		LIMIT :limit OFFSET :offset
	`

	res := make([]sdk.SearchResult, 0)

	if _, err := db.Select(&res, query, map[string]interface{}{
		"projects": pq.StringArray(filters.Projects),
		"types":    pq.StringArray(filters.Types),
		"query":    "%" + strings.ToLower(filters.Query) + "%",
		"limit":    limit,
		"offset":   offset,
	}); err != nil {
		if err == sql.ErrNoRows {
			return res, nil
		}
		return nil, sdk.WithStack(err)
	}

	return res, nil
}
