package hooks

import (
	"strconv"
	"time"

	"github.com/ovh/cds/sdk"
)

// GithubWebHookEvent represents payload send by github on a push event
type GithubWebHookEvent struct {
	Ref        string            `json:"ref"`
	Before     string            `json:"before"`
	After      string            `json:"after"`
	Created    bool              `json:"created"`
	Deleted    bool              `json:"deleted"`
	Forced     bool              `json:"forced"`
	BaseRef    interface{}       `json:"base_ref"`
	Compare    string            `json:"compare"`
	Commits    []GithubCommit    `json:"commits"`
	HeadCommit *GithubCommit     `json:"head_commit"`
	Repository *GithubRepository `json:"repository"`
	Pusher     GithubOwner       `json:"pusher"`
	Sender     GithubSender      `json:"sender"`
}

type GithubSender struct {
	Login             string `json:"login"`
	ID                int    `json:"id"`
	AvatarURL         string `json:"avatar_url"`
	GravatarID        string `json:"gravatar_id"`
	URL               string `json:"url"`
	HTMLURL           string `json:"html_url"`
	FollowersURL      string `json:"followers_url"`
	FollowingURL      string `json:"following_url"`
	GistsURL          string `json:"gists_url"`
	StarredURL        string `json:"starred_url"`
	SubscriptionsURL  string `json:"subscriptions_url"`
	OrganizationsURL  string `json:"organizations_url"`
	ReposURL          string `json:"repos_url"`
	EventsURL         string `json:"events_url"`
	ReceivedEventsURL string `json:"received_events_url"`
	Type              string `json:"type"`
	SiteAdmin         bool   `json:"site_admin"`
}

type GithubCommit struct {
	ID        string       `json:"id"`
	TreeID    string       `json:"tree_id"`
	Distinct  bool         `json:"distinct"`
	Message   string       `json:"message"`
	Timestamp time.Time    `json:"timestamp"`
	URL       string       `json:"url"`
	Author    GithubAuthor `json:"author"`
	Committer GithubAuthor `json:"committer"`
	Added     []string     `json:"added"`
	Removed   []string     `json:"removed"`
	Modified  []string     `json:"modified"`
}

type GithubAuthor struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type GithubOwner struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type GithubRepository struct {
	ID               int         `json:"id"`
	Name             string      `json:"name"`
	FullName         string      `json:"full_name"`
	Owner            GithubOwner `json:"owner"`
	Private          bool        `json:"private"`
	HTMLURL          string      `json:"html_url"`
	Description      string      `json:"description"`
	Fork             bool        `json:"fork"`
	URL              string      `json:"url"`
	ForksURL         string      `json:"forks_url"`
	KeysURL          string      `json:"keys_url"`
	CollaboratorsURL string      `json:"collaborators_url"`
	TeamsURL         string      `json:"teams_url"`
	HooksURL         string      `json:"hooks_url"`
	IssueEventsURL   string      `json:"issue_events_url"`
	EventsURL        string      `json:"events_url"`
	AssigneesURL     string      `json:"assignees_url"`
	BranchesURL      string      `json:"branches_url"`
	TagsURL          string      `json:"tags_url"`
	BlobsURL         string      `json:"blobs_url"`
	GitTagsURL       string      `json:"git_tags_url"`
	GitRefsURL       string      `json:"git_refs_url"`
	TreesURL         string      `json:"trees_url"`
	StatusesURL      string      `json:"statuses_url"`
	LanguagesURL     string      `json:"languages_url"`
	StargazersURL    string      `json:"stargazers_url"`
	ContributorsURL  string      `json:"contributors_url"`
	SubscribersURL   string      `json:"subscribers_url"`
	SubscriptionURL  string      `json:"subscription_url"`
	CommitsURL       string      `json:"commits_url"`
	GitCommitsURL    string      `json:"git_commits_url"`
	CommentsURL      string      `json:"comments_url"`
	IssueCommentURL  string      `json:"issue_comment_url"`
	ContentsURL      string      `json:"contents_url"`
	CompareURL       string      `json:"compare_url"`
	MergesURL        string      `json:"merges_url"`
	ArchiveURL       string      `json:"archive_url"`
	DownloadsURL     string      `json:"downloads_url"`
	IssuesURL        string      `json:"issues_url"`
	PullsURL         string      `json:"pulls_url"`
	MilestonesURL    string      `json:"milestones_url"`
	NotificationsURL string      `json:"notifications_url"`
	LabelsURL        string      `json:"labels_url"`
	ReleasesURL      string      `json:"releases_url"`
	CreateAt         GithubDate  `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
	PushedAt         GithubDate  `json:"pushed_at"`
	GitURL           string      `json:"git_url"`
	SSHURL           string      `json:"ssh_url"`
	CloneURL         string      `json:"clone_url"`
	SvnURL           string      `json:"svn_url"`
	Homepage         interface{} `json:"homepage"`
	Size             int         `json:"size"`
	StargazersCount  int         `json:"stargazers_count"`
	WatchersCount    int         `json:"watchers_count"`
	Language         interface{} `json:"language"`
	HasIssues        bool        `json:"has_issues"`
	HasDownloads     bool        `json:"has_downloads"`
	HasWiki          bool        `json:"has_wiki"`
	HasPages         bool        `json:"has_pages"`
	ForksCount       int         `json:"forks_count"`
	MirrorURL        interface{} `json:"mirror_url"`
	OpenIssuesCount  int         `json:"open_issues_count"`
	Forks            int         `json:"forks"`
	OpenIssues       int         `json:"open_issues"`
	Watchers         int         `json:"watchers"`
	DefaultBranch    string      `json:"default_branch"`
	Stargazers       int         `json:"stargazers"`
	MasterBranch     string      `json:"master_branch"`
}

func (g *GithubWebHookEvent) GetCommits() []sdk.VCSCommit {
	commits := []sdk.VCSCommit{}
	for _, c := range g.Commits {
		commit := sdk.VCSCommit{
			Hash: c.ID,
			Author: sdk.VCSAuthor{
				Name:        c.Author.Username,
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

type GithubDate time.Time

func (g *GithubDate) UnmarshalJSON(data []byte) error {
	var d time.Time

	dateInt, err := strconv.Atoi(string(data))
	if err == nil {
		d = time.Unix(int64(dateInt), 0)
	} else {
		if err := sdk.JSONUnmarshal(data, &d); err != nil {
			return err
		}
	}
	*g = GithubDate(d)
	return nil
}
