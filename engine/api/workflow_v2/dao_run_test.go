package workflow_v2

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunQueryOrderBy(t *testing.T) {
	for sort, expected := range map[string]string{
		"":                   "ORDER BY started DESC",
		"started:desc":       "ORDER BY started DESC",
		"started:asc":        "ORDER BY started ASC",
		"last_modified:desc": "ORDER BY last_modified DESC",
		"last_modified:asc":  "ORDER BY last_modified ASC",
	} {
		orderBy, err := runQueryOrderBy(sort)
		require.NoError(t, err, "sort %q", sort)
		require.Equal(t, expected, orderBy, "sort %q", sort)
	}

	for _, sort := range []string{"started", "started:", "started:up", "id:desc", "started:desc;DROP TABLE v2_workflow_run", "started:desc, id"} {
		_, err := runQueryOrderBy(sort)
		require.Error(t, err, "sort %q should be refused", sort)
	}
}

// annotation_strings only exists next to one of the annotations joins. A filter reading it from
// runQueryFilters would go unnoticed until a search leaving the join out ran against a database.
func TestRunQueryFilters_ReadTheAnnotationsOnlyWhereTheyAreJoined(t *testing.T) {
	require.NotContains(t, runQueryFilters, "annotation_strings")
	require.Contains(t, runQueryAnnotationFilters, "annotation_strings")
	require.Contains(t, runAnnotationsJoin, `as "annotation_strings"`)
	require.Contains(t, runAnnotationsJoinByProject, `as "annotation_strings"`)
}

// The two joins only differ by the project they are bounded to.
func TestRunAnnotationsJoin_ByProjectOnlyAddsTheProjectFilter(t *testing.T) {
	require.Contains(t, runAnnotationsJoinByProject, "WHERE run.project_key = :projKey")
	require.NotContains(t, runAnnotationsJoin, ":projKey")
	require.Equal(t,
		strings.Join(strings.Fields(runAnnotationsJoin), " "),
		strings.Replace(strings.Join(strings.Fields(runAnnotationsJoinByProject), " "), "WHERE run.project_key = :projKey ", "", 1),
	)
}
