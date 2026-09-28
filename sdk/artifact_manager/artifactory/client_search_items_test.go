package artifactory

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

// The filter part of a query, decoded, for a test to check it is valid json.
func filterOf(t *testing.T, query string) map[string]interface{} {
	t.Helper()
	start := strings.Index(query, "items.find(")
	require.NotEqual(t, -1, start, "query does not start with items.find")
	end := strings.LastIndex(query, ").include(")
	require.NotEqual(t, -1, end, "query does not end with an include")

	var filter map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(query[start+len("items.find("):end]), &filter),
		"the filter is not valid json: %s", query)
	return filter
}

func TestBuildItemsQueries(t *testing.T) {
	queries, err := buildItemsQueries([]sdk.ArtifactLocation{
		{Repository: "repo-a", Path: "dir/one", Name: "file.tgz"},
	}, "cds.signature")
	require.NoError(t, err)
	require.Len(t, queries, 1)

	require.Equal(t,
		`items.find({"$and":[{"repo":{"$eq":"repo-a"}},{"type":{"$eq":"any"}},`+
			`{"@cds.signature":{"$match":"*"}},{"$or":[`+
			`{"$and":[{"path":{"$eq":"dir/one"}},{"$or":[{"name":{"$eq":"file.tgz"}}]}]}`+
			`]}]}).include("repo","path","name","type","actual_md5","actual_sha1","sha256")`,
		queries[0].query)
	require.Equal(t, 1, queries[0].locations)
	filterOf(t, queries[0].query)
}

// The directory is the longest part of a branch, and a run puts most of its artifacts in the
// same one. Written once, a run fits in a single query.
func TestBuildItemsQueriesGroupsByDirectory(t *testing.T) {
	queries, err := buildItemsQueries([]sdk.ArtifactLocation{
		{Repository: "repo", Path: "same/dir", Name: "b.tgz"},
		{Repository: "repo", Path: "same/dir", Name: "a.tgz"},
		{Repository: "repo", Path: "other/dir", Name: "c.tgz"},
	}, "cds.signature")
	require.NoError(t, err)
	require.Len(t, queries, 1)
	require.Equal(t, 3, queries[0].locations)

	require.Equal(t, 1, strings.Count(queries[0].query, `"same/dir"`),
		"the directory must be written once for the two artifacts it holds")
	require.Equal(t, 1, strings.Count(queries[0].query, `"other/dir"`))
	filterOf(t, queries[0].query)
}

func TestBuildItemsQueriesSplitsPerRepository(t *testing.T) {
	queries, err := buildItemsQueries([]sdk.ArtifactLocation{
		{Repository: "repo-b", Path: "d", Name: "b.tgz"},
		{Repository: "repo-a", Path: "d", Name: "a.tgz"},
	}, "cds.signature")
	require.NoError(t, err)
	require.Len(t, queries, 2)

	// repositories come out sorted
	require.Contains(t, queries[0].query, `"repo-a"`)
	require.Contains(t, queries[1].query, `"repo-b"`)
}

// With a limit or an offset, artifactory counts database rows in place of items and returns
// a fraction of the matching artifacts, reported as the total.
func TestBuildItemsQueriesCarryNoPaging(t *testing.T) {
	queries, err := buildItemsQueries([]sdk.ArtifactLocation{
		{Repository: "repo", Path: "d", Name: "a.tgz"},
	}, "cds.signature")
	require.NoError(t, err)
	require.Len(t, queries, 1)

	require.NotContains(t, queries[0].query, ".limit(")
	require.NotContains(t, queries[0].query, ".offset(")
}

// A run result can be a folder, and the default of items.find is type file.
func TestBuildItemsQueriesMatchFolders(t *testing.T) {
	queries, err := buildItemsQueries([]sdk.ArtifactLocation{
		{Repository: "repo", Path: "d", Name: "a-folder"},
	}, "cds.signature")
	require.NoError(t, err)
	require.Contains(t, queries[0].query, `{"type":{"$eq":"any"}}`)
}

// The path and the name carry quotes and backslashes, and a percent sign that would trip a
// format string. The query must still be valid json.
func TestBuildItemsQueriesEscapeSpecialCharacters(t *testing.T) {
	queries, err := buildItemsQueries([]sdk.ArtifactLocation{
		{Repository: "repo", Path: `100%/di"r\x`, Name: `arti%fact"\.tgz`},
	}, "cds.signature")
	require.NoError(t, err)
	require.Len(t, queries, 1)

	filter := filterOf(t, queries[0].query)
	require.NotEmpty(t, filter)
	require.Contains(t, queries[0].query, `100%`)
}

func TestBuildItemsQueriesSplitOnLength(t *testing.T) {
	var locations []sdk.ArtifactLocation
	for i := 0; i < 500; i++ {
		locations = append(locations, sdk.ArtifactLocation{
			Repository: "a-repository-with-a-fairly-long-name-snapshot",
			Path:       "project/workflow/CD/application/0.1.0-121.sha.g8d0f769",
			Name:       fmt.Sprintf("application-ui_linux_amd64_%03d", i),
		})
	}

	queries, err := buildItemsQueries(locations, "cds.signature")
	require.NoError(t, err)
	require.Greater(t, len(queries), 1, "500 artifacts cannot fit in one query")

	var located int
	for i, q := range queries {
		require.LessOrEqual(t, len(q.query), aqlMaxQueryLength, "query %d is over the ceiling", i)
		filterOf(t, q.query)
		located += q.locations
	}
	require.Equal(t, len(locations), located, "every location must be asked about exactly once")
}

func TestBuildItemsQueriesOnEmptyInput(t *testing.T) {
	queries, err := buildItemsQueries(nil, "cds.signature")
	require.NoError(t, err)
	require.Empty(t, queries)
}

// Without a property key, every location comes back with its checksums: this is how the API
// reads what it signs without one api/storage call per artifact.
func TestBuildItemsQueriesWithoutPropertyKey(t *testing.T) {
	queries, err := buildItemsQueries([]sdk.ArtifactLocation{
		{Repository: "repo", Path: "d", Name: "a.tgz"},
	}, "")
	require.NoError(t, err)
	require.Len(t, queries, 1)

	require.NotContains(t, queries[0].query, `"@`, "an empty key must not become a property filter")
	require.Contains(t, queries[0].query, `.include("repo","path","name","type","actual_md5","actual_sha1","sha256")`)
	filterOf(t, queries[0].query)
}
