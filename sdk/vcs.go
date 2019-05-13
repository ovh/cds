package sdk

import (
	"context"
	"fmt"
	"io"
	"time"
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

// VCSServer is an interce for a OAuth VCS Server. The goal of this interface is to return a VCSAuthorizedClient
type VCSServer interface {
	AuthorizeRedirect(context.Context) (string, string, error)
	AuthorizeToken(context.Context, string, string) (string, string, error)
	GetAuthorizedClient(context.Context, string, string) (VCSAuthorizedClient, error)
}

// VCSAuthorizedClient is an interface for a connected client on a VCS Server.
type VCSAuthorizedClient interface {
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
	PullRequest(context.Context, string, int) (VCSPullRequest, error)
	PullRequests(context.Context, string) ([]VCSPullRequest, error)
	PullRequestComment(context.Context, string, int, string) error
	PullRequestCreate(context.Context, string, VCSPullRequest) (VCSPullRequest, error)

	//Hooks
	CreateHook(ctx context.Context, repo string, hook *VCSHook) error
	GetHook(ctx context.Context, repo, url string) (VCSHook, error)
	UpdateHook(ctx context.Context, repo, url string, hook VCSHook) error
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
