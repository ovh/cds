package hooks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

// Three commits of one push: shared.go is touched three times, new.go twice.
const testPushCommits = `[
    {"id": "aaaa", "added": ["new.go"], "modified": ["shared.go"], "removed": []},
    {"id": "bbbb", "added": [], "modified": ["shared.go", "new.go"], "removed": ["old.go"]},
    {"id": "cccc", "added": [], "modified": [], "removed": ["shared.go"]}
  ]`

var testPushExpectedPaths = []string{"new.go", "old.go", "shared.go"}

func Test_sortedUniquePaths(t *testing.T) {
	require.Equal(t, []string{}, sortedUniquePaths([]string{}))
	require.Nil(t, sortedUniquePaths(nil))
	require.Equal(t,
		[]string{"a/x.go", "b.go", "c.go"},
		sortedUniquePaths([]string{"c.go", "b.go", "c.go", "a/x.go", "b.go", "c.go"}),
	)
}

func Test_extractDataFromGithubRequest_PathsAreSortedAndUnique(t *testing.T) {
	body := `{
  "ref": "refs/heads/my-branch",
  "before": "1111111111111111111111111111111111111111",
  "after": "2222222222222222222222222222222222222222",
  "repository": {"full_name": "ovh/cds"},
  "commits": ` + testPushCommits + `
}`

	s := &Service{}
	repoName, data, err := s.extractDataFromGithubRequest([]byte(body), "push")
	require.NoError(t, err)
	require.Equal(t, "ovh/cds", repoName)
	require.Equal(t, sdk.WorkflowHookEventNamePush, data.CDSEventName)
	require.Equal(t, "2222222222222222222222222222222222222222", data.Commit)
	require.Equal(t, "1111111111111111111111111111111111111111", data.CommitFrom)
	require.Equal(t, testPushExpectedPaths, data.Paths)
}

func Test_extractDataFromGitlabRequest_PathsAreSortedAndUnique(t *testing.T) {
	body := `{
  "object_kind": "push",
  "ref": "refs/heads/my-branch",
  "before": "1111111111111111111111111111111111111111",
  "after": "2222222222222222222222222222222222222222",
  "project": {"path_with_namespace": "ovh/cds"},
  "commits": ` + testPushCommits + `
}`

	s := &Service{}
	repoName, data, err := s.extractDataFromGitlabRequest([]byte(body), "Push Hook")
	require.NoError(t, err)
	require.Equal(t, "ovh/cds", repoName)
	require.Equal(t, sdk.WorkflowHookEventNamePush, data.CDSEventName)
	require.Equal(t, "2222222222222222222222222222222222222222", data.Commit)
	require.Equal(t, "1111111111111111111111111111111111111111", data.CommitFrom)
	require.Equal(t, testPushExpectedPaths, data.Paths)
}

func Test_extractDataFromGiteaRequest_PathsAreSortedAndUnique(t *testing.T) {
	body := `{
  "ref": "refs/heads/my-branch",
  "before": "1111111111111111111111111111111111111111",
  "after": "2222222222222222222222222222222222222222",
  "repository": {"full_name": "ovh/cds"},
  "commits": ` + testPushCommits + `
}`

	s := &Service{}
	repoName, data, err := s.extractDataFromGiteaRequest([]byte(body), "push")
	require.NoError(t, err)
	require.Equal(t, "ovh/cds", repoName)
	require.Equal(t, sdk.WorkflowHookEventNamePush, data.CDSEventName)
	require.Equal(t, "2222222222222222222222222222222222222222", data.Commit)
	require.Equal(t, "1111111111111111111111111111111111111111", data.CommitFrom)
	require.Equal(t, testPushExpectedPaths, data.Paths)
}

func Test_extractDataFromForgejoPushEvent_PathsAreSortedAndUnique(t *testing.T) {
	body := `{
  "ref": "refs/heads/my-branch",
  "before": "1111111111111111111111111111111111111111",
  "after": "2222222222222222222222222222222222222222",
  "repository": {"full_name": "ovh/cds"},
  "head_commit": {"id": "cccc"},
  "commits": ` + testPushCommits + `
}`

	s := &Service{}
	repoName, data, err := s.extractDataFromForgejoPushEvent(context.TODO(), []byte(body))
	require.NoError(t, err)
	require.Equal(t, "ovh/cds", repoName)
	require.Equal(t, sdk.WorkflowHookEventNamePush, data.CDSEventName)
	require.Equal(t, "2222222222222222222222222222222222222222", data.Commit)
	require.Equal(t, "1111111111111111111111111111111111111111", data.CommitFrom)
	require.Equal(t, testPushExpectedPaths, data.Paths)
}
