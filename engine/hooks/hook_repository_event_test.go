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

func Test_isRefDeletion(t *testing.T) {
	s := &Service{}

	githubBody := `{"ref": "refs/heads/my-branch", "before": "1111111111111111111111111111111111111111", "after": "0000000000000000000000000000000000000000", "deleted": true, "repository": {"full_name": "ovh/cds"}, "commits": []}`
	_, data, err := s.extractDataFromGithubRequest([]byte(githubBody), "push")
	require.NoError(t, err)
	require.True(t, isRefDeletion(data))

	gitlabBody := `{"ref": "refs/heads/my-branch", "before": "1111111111111111111111111111111111111111", "after": "0000000000000000000000000000000000000000", "project": {"path_with_namespace": "ovh/cds"}, "commits": []}`
	_, data, err = s.extractDataFromGitlabRequest([]byte(gitlabBody), "Push Hook")
	require.NoError(t, err)
	require.True(t, isRefDeletion(data))

	bitbucketBody := `{"eventKey": "repo:refs_changed", "repository": {"slug": "cds", "project": {"key": "OVH"}}, "changes": [{"refId": "refs/heads/my-branch", "fromHash": "1111111111111111111111111111111111111111", "toHash": "0000000000000000000000000000000000000000", "type": "DELETE"}]}`
	_, data, err = s.extractDataFromBitbucketRequest([]byte(bitbucketBody))
	require.NoError(t, err)
	require.True(t, isRefDeletion(data))

	require.False(t, isRefDeletion(sdk.HookRepositoryEventExtractData{CDSEventName: sdk.WorkflowHookEventNamePush, Commit: "2222222222222222222222222222222222222222"}))
	require.False(t, isRefDeletion(sdk.HookRepositoryEventExtractData{CDSEventName: sdk.WorkflowHookEventNamePullRequest, Commit: NoCommit}))
}

func Test_extractDataFromForgejoPushEvent_NoHeadCommit(t *testing.T) {
	body := `{"ref": "refs/heads/my-branch", "before": "0000000000000000000000000000000000000000", "after": "2222222222222222222222222222222222222222", "repository": {"full_name": "ovh/cds"}, "commits": [], "head_commit": null}`

	s := &Service{}
	repoName, data, err := s.extractDataFromForgejoPushEvent(context.TODO(), []byte(body))
	require.NoError(t, err)
	require.Equal(t, "ovh/cds", repoName)
	require.Equal(t, "2222222222222222222222222222222222222222", data.Commit)
	require.False(t, data.CommitVerified)
}

// Forgejo PR fixture whose database id (4242) differs from its number (7).
const testForgejoPR = `{
    "id": 4242,
    "number": 7,
    "head": {"ref": "my-feature", "sha": "2222222222222222222222222222222222222222"},
    "base": {"ref": "master", "sha": "1111111111111111111111111111111111111111"}
  }`

func Test_extractDataFromForgejoIssueCommentPREvent(t *testing.T) {
	s := &Service{}
	for action, eventType := range map[HookIssueCommentAction]sdk.WorkflowHookEventType{
		HookIssueCommentCreated: sdk.WorkflowHookEventTypePullRequestCommentCreated,
		HookIssueCommentEdited:  sdk.WorkflowHookEventTypePullRequestCommentEdited,
		HookIssueCommentDeleted: sdk.WorkflowHookEventTypePullRequestCommentDeleted,
	} {
		body := `{
  "action": "` + string(action) + `",
  "is_pull": true,
  "issue": {"id": 4241, "number": 7},
  "pull_request": ` + testForgejoPR + `,
  "comment": {"id": 99, "body": "/retest"},
  "repository": {"full_name": "ovh/cds"}
}`
		repoName, data, err := s.extractDataFromForgejoIssueCommentPREvent([]byte(body))
		require.NoError(t, err)
		require.Equal(t, "ovh/cds", repoName)
		require.Equal(t, sdk.WorkflowHookEventNamePullRequestComment, data.CDSEventName)
		require.Equal(t, eventType, data.CDSEventType)
		require.Equal(t, int64(7), data.PullRequestID)
		require.Equal(t, "refs/heads/my-feature", data.Ref)
		require.Equal(t, "2222222222222222222222222222222222222222", data.Commit)
		require.Equal(t, "refs/heads/master", data.PullRequestRefTo)
		require.Equal(t, "/retest", data.Comment)
	}
}

func Test_extractDataFromForgejoPullRequestEvents_UsePullRequestNumber(t *testing.T) {
	s := &Service{}

	prBody := `{"action": "opened", "number": 7, "pull_request": ` + testForgejoPR + `, "repository": {"full_name": "ovh/cds"}}`
	_, data, err := s.extractDataFromForgejoPullRequestEvent([]byte(prBody), string(ForgejoEventTypePullRequest))
	require.NoError(t, err)
	require.Equal(t, int64(7), data.PullRequestID)

	reviewBody := `{"action": "reviewed", "number": 7, "pull_request": ` + testForgejoPR + `, "review": {"type": "pull_request_review_comment", "content": "lgtm"}, "repository": {"full_name": "ovh/cds"}}`
	_, data, err = s.extractDataFromForgejoPullRequestCommentEvent([]byte(reviewBody))
	require.NoError(t, err)
	require.Equal(t, int64(7), data.PullRequestID)
	require.Equal(t, "lgtm", data.Comment)
}

func Test_extractDataFromForgejoPullRequestEvents_MissingPullRequest(t *testing.T) {
	s := &Service{}

	_, _, err := s.extractDataFromForgejoIssueCommentPREvent([]byte(`{"action": "created", "comment": {"body": "/retest"}, "repository": {"full_name": "ovh/cds"}}`))
	require.Error(t, err)

	_, _, err = s.extractDataFromForgejoPullRequestEvent([]byte(`{"action": "opened", "repository": {"full_name": "ovh/cds"}}`), string(ForgejoEventTypePullRequest))
	require.Error(t, err)

	_, _, err = s.extractDataFromForgejoPullRequestCommentEvent([]byte(`{"action": "reviewed", "repository": {"full_name": "ovh/cds"}}`))
	require.Error(t, err)
}
