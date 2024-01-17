package sdk

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-gorp/gorp"
)

// HTTP Headers
const (
	HeaderXVCSURL           = "X-CDS-VCS-URL"
	HeaderXVCSURLApi        = "X-CDS-VCS-URL-API"
	HeaderXVCSType          = "X-CDS-VCS-TYPE"
	HeaderXVCSToken         = "X-CDS-VCS-TOKEN"
	HeaderXVCSUsername      = "X-CDS-VCS-USERNAME"
	HeaderXVCSSSHUsername   = "X-CDS-VCS-SSH-USERNAME"
	HeaderXVCSSSHPort       = "X-CDS-VCS-SSH-PORT"
	HeaderXVCSSSHPrivateKey = "X-CDS-VCS-SSH-PRIVATE-KEY"

	VCSTypeGitea           = "gitea"
	VCSTypeGerrit          = "gerrit"
	VCSTypeGitlab          = "gitlab"
	VCSTypeBitbucketServer = "bitbucketserver"
	VCSTypeBitbucketCloud  = "bitbucketcloud"
	VCSTypeGithub          = "github"
)

var (
	BitbucketEvents = []string{
		"repo:refs_changed",
		"repo:modified",
		"repo:forked",
		"repo:comment:added",
		"repo:comment:edited",
		"repo:comment:deleted",
		"pr:from_ref_updated",
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
		"pull_request_review_comment",
		"pull_request_review",
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

type VCSProject struct {
	ID           string            `json:"id" db:"id"`
	Name         string            `json:"name" db:"name" cli:"name,key"`
	Type         string            `json:"type" db:"type"`
	Created      time.Time         `json:"created" db:"created"`
	LastModified time.Time         `json:"last_modified" db:"last_modified"`
	CreatedBy    string            `json:"created_by" db:"created_by"`
	ProjectID    int64             `json:"-" db:"project_id"`
	Description  string            `json:"description" db:"description"`
	URL          string            `json:"url" db:"url"`
	Auth         VCSAuthProject    `json:"auth" db:"auth" gorpmapping:"encrypted,ProjectID"`
	Options      VCSOptionsProject `json:"options" db:"options"`
}

type VCSAuthProject struct {
	// VCS Authentication
	Username   string `json:"username,omitempty" db:"-"`
	Token      string `json:"token,omitempty" db:"-"`
	SSHKeyName string `json:"sshKeyName,omitempty" db:"-"`

	// Used by gerrit
	SSHUsername   string `json:"sshUsername,omitempty" db:"-"`
	SSHPort       int    `json:"sshPort,omitempty" db:"-"`
	SSHPrivateKey string `json:"sshPrivateKey,omitempty" db:"-"`
}

type VCSOptionsProject struct {
	DisableWebhooks      bool   `json:"disableWebhooks,omitempty" db:"-"`
	DisableStatus        bool   `json:"disableStatus,omitempty" db:"-"`
	DisableStatusDetails bool   `json:"disableStatusDetails,omitempty" db:"-"`
	DisablePolling       bool   `json:"disablePolling,omitempty" db:"-"`
	URLAPI               string `json:"urlApi,omitempty" db:"-"` // optional
}

func (v VCSProject) Lint(prj Project) error {
	// If it's not a gerrit vcs
	if v.Auth.SSHUsername == "" {
		if v.Auth.Username == "" {
			return NewErrorFrom(ErrInvalidData, "missing auth username")
		}
		if v.Auth.Token == "" {
			return NewErrorFrom(ErrInvalidData, "missing auth token")
		}
	}

	if v.Auth.SSHKeyName != "" {
		found := false
		for _, k := range prj.Keys {
			if k.Name == v.Auth.SSHKeyName {
				found = true
				break
			}
		}
		if !found {
			return NewErrorFrom(ErrNotFound, "unable to find ssh key %s on project", v.Auth.SSHKeyName)
		}
	}

	return nil
}

func (v VCSOptionsProject) Value() (driver.Value, error) {
	j, err := json.Marshal(v)
	return j, WrapError(err, "cannot marshal VCSOptionsProject")
}

func (v *VCSOptionsProject) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	source, ok := src.([]byte)
	if !ok {
		return WithStack(fmt.Errorf("type assertion .([]byte) failed (%T)", src))
	}
	return WrapError(JSONUnmarshal(source, v), "cannot unmarshal VCSOptionsProject")
}

// VCSConfiguration represent a small vcs configuration
type VCSConfiguration struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type VCSGerritConfiguration struct {
	SSHUsername   string `json:"sshUsername"`
	SSHPrivateKey string `json:"sshPrivateKey"`
	URL           string `json:"url"`
	SSHPort       int    `json:"sshport"`
}

// VCSAuth contains tokens (oauth2 tokens or personalAccessToken)
type VCSAuth struct {
	Type     string
	URL      string
	URLApi   string // optional
	Username string
	Token    string

	SSHUsername string
	SSHPort     int
}

// VCSServer is an interface for a OAuth VCS Server. The goal of this interface is to return a VCSAuthorizedClient.
type VCSServer interface {
	GetAuthorizedClient(context.Context, VCSAuth) (VCSAuthorizedClient, error)
}

type VCSServerService interface {
	GetAuthorizedClient(context.Context, string, string, int64) (VCSAuthorizedClientService, error)
}

type VCSBranchFilters struct {
	BranchName string
	Default    bool
}

type VCSBranchesFilter struct {
	Limit int64
}

type VCSArchiveRequest struct {
	Path   string `json:"path"`
	Format string `json:"format"`
	Commit string `json:"commit"`
}

// VCSAuthorizedClientCommon is an interface for a connected client on a VCS Server.
type VCSAuthorizedClientCommon interface {
	//Repos
	Repos(context.Context) ([]VCSRepo, error)
	RepoByFullname(ctx context.Context, fullname string) (VCSRepo, error)

	//Branches
	Branches(context.Context, string, VCSBranchesFilter) ([]VCSBranch, error)
	Branch(ctx context.Context, repo string, filters VCSBranchFilters) (*VCSBranch, error)

	//Tags
	Tags(ctx context.Context, repo string) ([]VCSTag, error)
	Tag(ctx context.Context, repo string, tagName string) (VCSTag, error)

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
	SetStatus(ctx context.Context, event Event) error
	SetDisableStatusDetails(disableStatusDetails bool)
	ListStatuses(ctx context.Context, repo string, ref string) ([]VCSCommitStatus, error)

	// Release
	Release(ctx context.Context, repo, tagName, releaseTitle, releaseDescription string) (*VCSRelease, error)
	UploadReleaseFile(ctx context.Context, repo string, releaseName string, uploadURL string, artifactName string, r io.Reader, length int) error

	// Forks
	ListForks(ctx context.Context, repo string) ([]VCSRepo, error)

	// File
	GetArchive(ctx context.Context, repo string, dir string, format string, commit string) (io.Reader, http.Header, error)
	ListContent(ctx context.Context, repo string, commit, dir string) ([]VCSContent, error)
	GetContent(ctx context.Context, repo string, commit, dir string) (VCSContent, error)

	// Search
	SearchPullRequest(ctx context.Context, repoFullName, commit, state string) (*VCSPullRequest, error)
}

type VCSAuthorizedClient interface {
	VCSAuthorizedClientCommon
	PullRequests(ctx context.Context, repo string, opts VCSPullRequestOptions) ([]VCSPullRequest, error)
}

type VCSAuthorizedClientService interface {
	VCSAuthorizedClientCommon
	PullRequests(ctx context.Context, repo string, mods ...VCSRequestModifier) ([]VCSPullRequest, error)
	IsGerrit(ctx context.Context, db gorp.SqlExecutor) (bool, error)
	IsBitbucketCloud() bool
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
