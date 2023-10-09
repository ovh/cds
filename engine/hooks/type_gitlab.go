package hooks

import (
	"time"

	"github.com/ovh/cds/sdk"
)

// GitlabEvent represents payload send by gitlab on a push event
type GitlabEvent struct {
	ObjectKind        string            `json:"object_kind"`
	Before            string            `json:"before"`
	After             string            `json:"after"`
	Ref               string            `json:"ref"`
	CheckoutSha       string            `json:"checkout_sha"`
	UserID            int               `json:"user_id"`
	UserName          string            `json:"user_name"`
	UserUsername      string            `json:"user_username"`
	UserEmail         string            `json:"user_email"`
	UserAvatar        string            `json:"user_avatar"`
	ProjectID         int               `json:"project_id"`
	Project           *GitlabProject    `json:"project"`
	Repository        *GitlabRepository `json:"repository"`
	Commits           []GitlabCommit    `json:"commits"`
	TotalCommitsCount int               `json:"total_commits_count"`
}

type GitlabCommit struct {
	ID        string       `json:"id"`
	Message   string       `json:"message"`
	Timestamp time.Time    `json:"timestamp"`
	URL       string       `json:"url"`
	Author    GitlabAuthor `json:"author"`
	Added     []string     `json:"added"`
	Modified  []string     `json:"modified"`
	Removed   []string     `json:"removed"`
}

type GitlabAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type GitlabRepository struct {
	Name            string `json:"name"`
	URL             string `json:"url"`
	Description     string `json:"description"`
	Homepage        string `json:"homepage"`
	GitHTTPURL      string `json:"git_http_url"`
	GitSSHURL       string `json:"git_ssh_url"`
	VisibilityLevel int    `json:"visibility_level"`
}

type GitlabProject struct {
	ID                int         `json:"id"`
	Name              string      `json:"name"`
	Description       string      `json:"description"`
	WebURL            string      `json:"web_url"`
	AvatarURL         interface{} `json:"avatar_url"`
	GitSSHURL         string      `json:"git_ssh_url"`
	GitHTTPURL        string      `json:"git_http_url"`
	Namespace         string      `json:"namespace"`
	VisibilityLevel   int         `json:"visibility_level"`
	PathWithNamespace string      `json:"path_with_namespace"`
	DefaultBranch     string      `json:"default_branch"`
	Homepage          string      `json:"homepage"`
	URL               string      `json:"url"`
	SSHURL            string      `json:"ssh_url"`
	HTTPURL           string      `json:"http_url"`
}

func (g *GitlabEvent) GetCommits() []sdk.VCSCommit {
	commits := []sdk.VCSCommit{}
	for _, c := range g.Commits {
		commit := sdk.VCSCommit{
			Hash: c.ID,
			Author: sdk.VCSAuthor{
				Name:        c.Author.Name,
				DisplayName: c.Author.Name,
				Email:       c.Author.Email,
			},
			Message:   c.Message,
			URL:       c.URL,
			Timestamp: c.Timestamp.Unix(),
		}
		commits = append(commits, commit)
	}
	return commits
}
