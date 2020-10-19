package sdk

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTP Headers
const (
	HeaderXAccessToken        = "X-CDS-ACCESS-TOKEN"
	HeaderXAccessTokenCreated = "X-CDS-ACCESS-TOKEN-CREATED"
	HeaderXAccessTokenSecret  = "X-CDS-ACCESS-TOKEN-SECRET"
)

var (
	BitbucketEvents = []string{
		"repo:refs_changed",
		"repo:modified",
		"repo:forked",
		"repo:comment:added",
		"repo:comment:edited",
		"repo:comment:deleted",
		"pr:opened",
		"pr:modified",
		"pr:reviewer:updated",
		"pr:reviewer:approved",
		"pr:reviewer:unapproved",
		"pr:reviewer:needs_work",
		"pr:merged",
		"pr:declined",
		"pr:deleted",
		"pr:comment:added",
		"pr:comment:edited",
		"pr:comment:deleted",
	}

	BitbucketEventsDefault = []string{
		"repo:refs_changed",
	}

	BitbucketCloudEvents = []string{
		"repo:push",
		"pullrequest:unapproved",
		"issue:comment_created",
		"pullrequest:approved",
		"repo:created",
		"repo:deleted",
		"repo:imported",
		"pullrequest:comment_updated",
		"issue:updated",
		"project:updated",
		"pullrequest:comment_created",
		"repo:commit_status_updated",
		"pullrequest:updated",
		"issue:created",
		"repo:fork",
		"pullrequest:comment_deleted",
		"repo:commit_status_created",
		"repo:updated",
		"pullrequest:rejected",
		"pullrequest:fulfilled",
		"pullrequest:created",
		"repo:transfer",
		"repo:commit_comment_created",
	}

	BitbucketCloudEventsDefault = []string{
		"repo:push",
	}

	GitHubEvents = []string{
		"push",
		"check_run",
		"check_suite",
		"commit_comment",
		"create",
		"delete",
		"deployment",
		"deployment_status",
		"fork",
		"github_app_authorization",
		"gollum",
		"installation",
		"installation_repositories",
		"issue_comment",
		"issues",
		"label",
		"marketplace_purchase",
		"member",
		"membership",
		"milestone",
		"organization",
		"org_block",
		"page_build",
		"project_card",
		"project_column",
		"project",
		"public",
		"pull-request_review_comment",
		"pull-request_review",
		"pull_request",
		"repository",
		"repository_import",
		"repository_vulnerability_alert",
		"release",
		"security_advisory",
		"status",
		"team",
		"team_add",
		"watch",
	}

	GitHubEventsDefault = []string{
		"push",
	}

	GitlabEventsDefault = []string{
		"Push Hook",
		"Tag Push Hook",
	}

	GerritEvents = []string{
		GerritEventTypePatchsetCreated,
		GerritEventTypeAssignedChanged,
		GerritEventTypeChangeAbandoned,
		GerritEventTypeChangeDeleted,
		GerritEventTypeChangeMerged,
		GerritEventTypeChangeRestored,
		GerritEventTypeCommentAdded,
		GerritEventTypeDrafPublished,
		GerritEventTypeDroppedOutput,
		GerritEventTypeHashTagsChanged,
		GerritEventTypeProjectCreated,
		GerritEventTypeRefUpdated,
		GerritEventTypeReviewerAdded,
		GerritEventTypeReviewerDelete,
		GerritEventTypeTopicChanged,
		GerritEventTypeWIPStateChanged,
		GerritEventTypePrivateStateChanged,
		GerritEventTypeVoteDeleted,
	}

	GerritEventTypeAssignedChanged     = "assignee-changed"
	GerritEventTypeChangeAbandoned     = "change-abandoned"
	GerritEventTypeChangeDeleted       = "change-deleted"
	GerritEventTypeChangeMerged        = "change-merged"
	GerritEventTypeChangeRestored      = "change-restored"
	GerritEventTypeCommentAdded        = "comment-added"
	GerritEventTypeDrafPublished       = "draft-published"
	GerritEventTypeDroppedOutput       = "dropped-output"
	GerritEventTypeHashTagsChanged     = "hashtags-changed"
	GerritEventTypeProjectCreated      = "project-created"
	GerritEventTypePatchsetCreated     = "patchset-created"
	GerritEventTypeRefUpdated          = "ref-updated"
	GerritEventTypeReviewerAdded       = "reviewer-added"
	GerritEventTypeReviewerDelete      = "reviewer-deleted"
	GerritEventTypeTopicChanged        = "topic-changed"
	GerritEventTypeWIPStateChanged     = "wip-state-changed"
	GerritEventTypePrivateStateChanged = "private-state-changed"
	GerritEventTypeVoteDeleted         = "vote-deleted"

	GerritEventsDefault = []string{
		GerritEventTypePatchsetCreated,
	}
)

// BuildNumberAndHash represents BuildNumber, Commit Hash and Branch for a Pipeline Build or Node Run
type BuildNumberAndHash struct {
	BuildNumber int64
	Hash        string
	Branch      string
	Tag         string
	Remote      string
	RemoteURL   string
}

// VCSConfiguration represent a small vcs configuration
type VCSConfiguration struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Password string `json:"password"`
	URL      string `json:"url"`
	SSHPort  int    `json:"sshport"`
}

type VCSServerCommon interface {
	AuthorizeRedirect(context.Context) (string, string, error)
	AuthorizeToken(context.Context, string, string) (string, string, error)
}

// VCSServer is an interface for a OAuth VCS Server. The goal of this interface is to return a VCSAuthorizedClient.
type VCSServer interface {
	VCSServerCommon
	GetAuthorizedClient(context.Context, string, string, int64) (VCSAuthorizedClient, error)
}

type VCSServerService interface {
	VCSServerCommon
	GetAuthorizedClient(context.Context, string, string, int64) (VCSAuthorizedClientService, error)
}

// VCSAuthorizedClientCommon is an interface for a connected client on a VCS Server.
type VCSAuthorizedClientCommon interface {
	//Repos
	Repos(context.Context) ([]VCSRepo, error)
	RepoByFullname(ctx context.Context, fullname string) (VCSRepo, error)

	//Branches
	Branches(context.Context, string) ([]VCSBranch, error)
	Branch(ctx context.Context, repo string, branch string) (*VCSBranch, error)

	//Tags
	Tags(ctx context.Context, repo string) ([]VCSTag, error)

	//Commits
	Commits(ctx context.Context, repo, branch, since, until string) ([]VCSCommit, error)
	Commit(ctx context.Context, repo, hash string) (VCSCommit, error)
	CommitsBetweenRefs(ctx context.Context, repo, base, head string) ([]VCSCommit, error)

	// PullRequests
	PullRequest(ctx context.Context, repo string, id string) (VCSPullRequest, error)
	PullRequestComment(ctx context.Context, repo string, c VCSPullRequestCommentRequest) error
	PullRequestCreate(ctx context.Context, repo string, pr VCSPullRequest) (VCSPullRequest, error)

	//Hooks
	CreateHook(ctx context.Context, repo string, hook *VCSHook) error
	UpdateHook(ctx context.Context, repo string, hook *VCSHook) error
	GetHook(ctx context.Context, repo, url string) (VCSHook, error)
	DeleteHook(ctx context.Context, repo string, hook VCSHook) error

	//Events
	GetEvents(ctx context.Context, repo string, dateRef time.Time) ([]interface{}, time.Duration, error)
	PushEvents(context.Context, string, []interface{}) ([]VCSPushEvent, error)
	CreateEvents(context.Context, string, []interface{}) ([]VCSCreateEvent, error)
	DeleteEvents(context.Context, string, []interface{}) ([]VCSDeleteEvent, error)
	PullRequestEvents(context.Context, string, []interface{}) ([]VCSPullRequestEvent, error)

	// Set build status on repository
	SetStatus(context.Context, Event) error
	ListStatuses(ctx context.Context, repo string, ref string) ([]VCSCommitStatus, error)

	// Release
	Release(ctx context.Context, repo, tagName, releaseTitle, releaseDescription string) (*VCSRelease, error)
	UploadReleaseFile(ctx context.Context, repo string, releaseName string, uploadURL string, artifactName string, r io.ReadCloser) error

	// Forks
	ListForks(ctx context.Context, repo string) ([]VCSRepo, error)

	// Permissions
	GrantWritePermission(ctx context.Context, repo string) error

	// Access Token
	GetAccessToken(ctx context.Context) string
}

type VCSAuthorizedClient interface {
	VCSAuthorizedClientCommon
	PullRequests(ctx context.Context, repo string, opts VCSPullRequestOptions) ([]VCSPullRequest, error)
}

type VCSAuthorizedClientService interface {
	VCSAuthorizedClientCommon
	PullRequests(ctx context.Context, repo string, mods ...VCSRequestModifier) ([]VCSPullRequest, error)
}

type VCSRequestModifier func(r *http.Request)

func VCSRequestModifierWithState(state VCSPullRequestState) VCSRequestModifier {
	return func(r *http.Request) {
		q := r.URL.Query()
		q.Set("state", string(state))
		r.URL.RawQuery = q.Encode()
	}
}

// GetDefaultBranch return the default branch
func GetDefaultBranch(branches []VCSBranch) VCSBranch {
	for _, branch := range branches {
		if branch.Default {
			return branch
		}
	}
	return VCSBranch{}
}

// VCSCommitStatusDescription return a node formated status description
func VCSCommitStatusDescription(projKey, workflowName string, evt EventRunWorkflowNode) string {
	key := fmt.Sprintf("%s-%s-%s",
		projKey,
		workflowName,
		evt.NodeName,
	)
	return fmt.Sprintf("CDS/%s", key)
}
