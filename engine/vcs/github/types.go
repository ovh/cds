package github

import (
	"fmt"
	"strconv"
	"time"
)

// Timestamp represents a time that can be unmarshalled from a JSON string
// formatted as either an RFC3339 or Unix timestamp. This is necessary for some
// fields since the GitHub API is inconsistent in how it represents times. All
// exported methods of time.Time can be called on Timestamp.
type Timestamp struct {
	time.Time
}

func (t Timestamp) String() string {
	return t.Time.String()
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// Time is expected in RFC3339 or Unix format.
func (t *Timestamp) UnmarshalJSON(data []byte) (err error) {
	str := string(data)
	i, err := strconv.ParseInt(str, 10, 64)
	if err == nil {
		(*t).Time = time.Unix(i, 0)
	} else {
		(*t).Time, err = time.Parse(`"`+time.RFC3339+`"`, str)
	}
	return
}

// Equal reports whether t and u are equal based on time.Equal
func (t Timestamp) Equal(u Timestamp) bool {
	return t.Time.Equal(u.Time)
}

// WebhookCreate represent struct to create a webhook
type WebhookCreate struct {
	ID     int           `json:"id"`
	Name   string        `json:"name"`
	Active bool          `json:"active"`
	Events []string      `json:"events"`
	Config WebHookConfig `json:"config"`
}

// WebHookConfig represent the configuration of a webhook
type WebHookConfig struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
}

// User represents a GitHub user.
type User struct {
	Login             string    `json:"login,omitempty"`
	ID                int       `json:"id,omitempty"`
	AvatarURL         string    `json:"avatar_url,omitempty"`
	HTMLURL           string    `json:"html_url,omitempty"`
	GravatarID        string    `json:"gravatar_id,omitempty"`
	Name              string    `json:"name,omitempty"`
	Company           string    `json:"company,omitempty"`
	Blog              string    `json:"blog,omitempty"`
	Location          string    `json:"location,omitempty"`
	Email             string    `json:"email,omitempty"`
	Hireable          bool      `json:"hireable,omitempty"`
	Bio               string    `json:"bio,omitempty"`
	PublicRepos       int       `json:"public_repos,omitempty"`
	PublicGists       int       `json:"public_gists,omitempty"`
	Followers         int       `json:"followers,omitempty"`
	Following         int       `json:"following,omitempty"`
	CreatedAt         Timestamp `json:"created_at,omitempty"`
	UpdatedAt         Timestamp `json:"updated_at,omitempty"`
	SuspendedAt       Timestamp `json:"suspended_at,omitempty"`
	Type              string    `json:"type,omitempty"`
	SiteAdmin         bool      `json:"site_admin,omitempty"`
	TotalPrivateRepos int       `json:"total_private_repos,omitempty"`
	OwnedPrivateRepos int       `json:"owned_private_repos,omitempty"`
	PrivateGists      int       `json:"private_gists,omitempty"`
	DiskUsage         int       `json:"disk_usage,omitempty"`
	Collaborators     int       `json:"collaborators,omitempty"`
	Plan              Plan      `json:"plan,omitempty"`
	URL               string    `json:"url,omitempty"`
	EventsURL         string    `json:"events_url,omitempty"`
	FollowingURL      string    `json:"following_url,omitempty"`
	FollowersURL      string    `json:"followers_url,omitempty"`
	GistsURL          string    `json:"gists_url,omitempty"`
	OrganizationsURL  string    `json:"organizations_url,omitempty"`
	ReceivedEventsURL string    `json:"received_events_url,omitempty"`
	ReposURL          string    `json:"repos_url,omitempty"`
	StarredURL        string    `json:"starred_url,omitempty"`
	SubscriptionsURL  string    `json:"subscriptions_url,omitempty"`
}

// Repository represents a GitHub repository.
type Repository struct {
	ID               int             `json:"id,omitempty"`
	Owner            User            `json:"owner,omitempty"`
	Name             string          `json:"name,omitempty"`
	FullName         string          `json:"full_name,omitempty"`
	Description      string          `json:"description,omitempty"`
	Homepage         string          `json:"homepage,omitempty"`
	DefaultBranch    string          `json:"default_branch,omitempty"`
	MasterBranch     string          `json:"master_branch,omitempty"`
	CreatedAt        Timestamp       `json:"created_at,omitempty"`
	PushedAt         Timestamp       `json:"pushed_at,omitempty"`
	UpdatedAt        Timestamp       `json:"updated_at,omitempty"`
	HTMLURL          string          `json:"html_url,omitempty"`
	CloneURL         string          `json:"clone_url,omitempty"`
	GitURL           string          `json:"git_url,omitempty"`
	MirrorURL        string          `json:"mirror_url,omitempty"`
	SSHURL           string          `json:"ssh_url,omitempty"`
	SVNURL           string          `json:"svn_url,omitempty"`
	Language         string          `json:"language,omitempty"`
	Fork             bool            `json:"fork"`
	ForksCount       int             `json:"forks_count,omitempty"`
	NetworkCount     int             `json:"network_count,omitempty"`
	OpenIssuesCount  int             `json:"open_issues_count,omitempty"`
	StargazersCount  int             `json:"stargazers_count,omitempty"`
	SubscribersCount int             `json:"subscribers_count,omitempty"`
	WatchersCount    int             `json:"watchers_count,omitempty"`
	Size             int             `json:"size,omitempty"`
	AutoInit         bool            `json:"auto_init,omitempty"`
	Parent           *Repository     `json:"parent,omitempty"`
	Source           *Repository     `json:"source,omitempty"`
	Organization     Organization    `json:"organization,omitempty"`
	Permissions      map[string]bool `json:"permissions,omitempty"`
	URL              string          `json:"url,omitempty"`
	ArchiveURL       string          `json:"archive_url,omitempty"`
	AssigneesURL     string          `json:"assignees_url,omitempty"`
	BlobsURL         string          `json:"blobs_url,omitempty"`
	BranchesURL      string          `json:"branches_url,omitempty"`
	CollaboratorsURL string          `json:"collaborators_url,omitempty"`
	CommentsURL      string          `json:"comments_url,omitempty"`
	CommitsURL       string          `json:"commits_url,omitempty"`
	CompareURL       string          `json:"compare_url,omitempty"`
	ContentsURL      string          `json:"contents_url,omitempty"`
	ContributorsURL  string          `json:"contributors_url,omitempty"`
	DeploymentsURL   string          `json:"deployments_url,omitempty"`
	DownloadsURL     string          `json:"downloads_url,omitempty"`
	EventsURL        string          `json:"events_url,omitempty"`
	ForksURL         string          `json:"forks_url,omitempty"`
	GitCommitsURL    string          `json:"git_commits_url,omitempty"`
	GitRefsURL       string          `json:"git_refs_url,omitempty"`
	GitTagsURL       string          `json:"git_tags_url,omitempty"`
	HooksURL         string          `json:"hooks_url,omitempty"`
	IssueCommentURL  string          `json:"issue_comment_url,omitempty"`
	IssueEventsURL   string          `json:"issue_events_url,omitempty"`
	IssuesURL        string          `json:"issues_url,omitempty"`
	KeysURL          string          `json:"keys_url,omitempty"`
	LabelsURL        string          `json:"labels_url,omitempty"`
	LanguagesURL     string          `json:"languages_url,omitempty"`
	MergesURL        string          `json:"merges_url,omitempty"`
	MilestonesURL    string          `json:"milestones_url,omitempty"`
	NotificationsURL string          `json:"notifications_url,omitempty"`
	PullsURL         string          `json:"pulls_url,omitempty"`
	ReleasesURL      string          `json:"releases_url,omitempty"`
	StargazersURL    string          `json:"stargazers_url,omitempty"`
	StatusesURL      string          `json:"statuses_url,omitempty"`
	SubscribersURL   string          `json:"subscribers_url,omitempty"`
	SubscriptionURL  string          `json:"subscription_url,omitempty"`
	TagsURL          string          `json:"tags_url,omitempty"`
	TreesURL         string          `json:"trees_url,omitempty"`
	TeamsURL         string          `json:"teams_url,omitempty"`
}

// Organization represents a GitHub organization account.
type Organization struct {
	Login             string    `json:"login,omitempty"`
	ID                int       `json:"id,omitempty"`
	AvatarURL         string    `json:"avatar_url,omitempty"`
	HTMLURL           string    `json:"html_url,omitempty"`
	Name              string    `json:"name,omitempty"`
	Company           string    `json:"company,omitempty"`
	Blog              string    `json:"blog,omitempty"`
	Location          string    `json:"location,omitempty"`
	Email             string    `json:"email,omitempty"`
	Description       string    `json:"description,omitempty"`
	PublicRepos       int       `json:"public_repos,omitempty"`
	PublicGists       int       `json:"public_gists,omitempty"`
	Followers         int       `json:"followers,omitempty"`
	Following         int       `json:"following,omitempty"`
	CreatedAt         time.Time `json:"created_at,omitempty"`
	UpdatedAt         time.Time `json:"updated_at,omitempty"`
	TotalPrivateRepos int       `json:"total_private_repos,omitempty"`
	OwnedPrivateRepos int       `json:"owned_private_repos,omitempty"`
	PrivateGists      int       `json:"private_gists,omitempty"`
	DiskUsage         int       `json:"disk_usage,omitempty"`
	Collaborators     int       `json:"collaborators,omitempty"`
	BillingEmail      string    `json:"billing_email,omitempty"`
	Type              string    `json:"type,omitempty"`
	Plan              Plan      `json:"plan,omitempty"`
	URL               string    `json:"url,omitempty"`
	EventsURL         string    `json:"events_url,omitempty"`
	HooksURL          string    `json:"hooks_url,omitempty"`
	IssuesURL         string    `json:"issues_url,omitempty"`
	MembersURL        string    `json:"members_url,omitempty"`
	PublicMembersURL  string    `json:"public_members_url,omitempty"`
	ReposURL          string    `json:"repos_url,omitempty"`
}

// Plan represents the payment plan for an account.  See plans at https://github.com/plans.
type Plan struct {
	Name          string `json:"name,omitempty"`
	Space         int    `json:"space,omitempty"`
	Collaborators int    `json:"collaborators,omitempty"`
	PrivateRepos  int    `json:"private_repos,omitempty"`
}

// Branch represents a repository branch
type Branch struct {
	Name       string     `json:"name,omitempty"`
	Commit     Commit     `json:"commit,omitempty"`
	Protection Protection `json:"protection,omitempty"`
}

// Protection represents a repository branch's protection
type Protection struct {
	Enabled bool `json:"enabled,omitempty"`
}

// Commit represents a GitHub commit.
type Commit struct {
	Sha    string `json:"sha"`
	Commit struct {
		Author struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  Timestamp `json:"date"`
		} `json:"author"`
		Committer struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  Timestamp `json:"date"`
		} `json:"committer"`
		Message string `json:"message"`
		Tree    struct {
			Sha string `json:"sha"`
			URL string `json:"url"`
		} `json:"tree"`
		URL          string `json:"url"`
		CommentCount int    `json:"comment_count"`
	} `json:"commit"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	CommentsURL string `json:"comments_url"`
	Author      struct {
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
	} `json:"author"`
	Committer struct {
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
	} `json:"committer"`
	Parents []struct {
		Sha     string `json:"sha"`
		URL     string `json:"url"`
		HTMLURL string `json:"html_url"`
	} `json:"parents"`
	Stats struct {
		Total     int `json:"total"`
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
	} `json:"stats"`
	Files []struct {
		Sha         string `json:"sha"`
		Filename    string `json:"filename"`
		Status      string `json:"status"`
		Additions   int    `json:"additions"`
		Deletions   int    `json:"deletions"`
		Changes     int    `json:"changes"`
		BlobURL     string `json:"blob_url"`
		RawURL      string `json:"raw_url"`
		ContentsURL string `json:"contents_url"`
		Patch       string `json:"patch"`
	} `json:"files"`
}

// Tree represents a GitHub tree.
type Tree struct {
	SHA     *string     `json:"sha,omitempty"`
	Entries []TreeEntry `json:"tree,omitempty"`
}

// TreeEntry represents the contents of a tree structure.  TreeEntry can
// represent either a blob, a commit (in the case of a submodule), or another
// tree.
type TreeEntry struct {
	SHA     *string `json:"sha,omitempty"`
	Path    *string `json:"path,omitempty"`
	Mode    *string `json:"mode,omitempty"`
	Type    *string `json:"type,omitempty"`
	Size    *int    `json:"size,omitempty"`
	Content *string `json:"content,omitempty"`
}

//Events represent repository events
type Events []Event

// Event represent a repository event
type Event struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Actor struct {
		ID           int    `json:"id"`
		Login        string `json:"login"`
		DisplayLogin string `json:"display_login"`
		GravatarID   string `json:"gravatar_id"`
		URL          string `json:"url"`
		AvatarURL    string `json:"avatar_url"`
	} `json:"actor"`
	Repo struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"repo"`
	Payload struct {
		PushID       int    `json:"push_id"`
		Action       string `json:"action"`
		Size         int    `json:"size"`
		DistinctSize int    `json:"distinct_size"`
		Ref          string `json:"ref"`
		Head         string `json:"head"`
		Before       string `json:"before"`
		Commits      []struct {
			Sha    string `json:"sha"`
			Author struct {
				Email string `json:"email"`
				Name  string `json:"name"`
			} `json:"author"`
			Message  string `json:"message"`
			Distinct bool   `json:"distinct"`
			URL      string `json:"url"`
		} `json:"commits"`
		PullRequest PullRequest `json:"pull_request"`
	} `json:"payload"`
	Public    bool      `json:"public"`
	CreatedAt Timestamp `json:"created_at"`
	Org       struct {
		ID         int    `json:"id"`
		Login      string `json:"login"`
		GravatarID string `json:"gravatar_id"`
		URL        string `json:"url"`
		AvatarURL  string `json:"avatar_url"`
	} `json:"org"`
}

//CreateStatus represents create a Status API Payload
type CreateStatus struct {
	State       string `json:"state"`
	TargetURL   string `json:"target_url"`
	Description string `json:"description"`
	Context     string `json:"context"`
}

//Status represents Create a Status from API
type Status struct {
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	State       string    `json:"state"`
	TargetURL   string    `json:"target_url"`
	Description string    `json:"description"`
	ID          int       `json:"id"`
	URL         string    `json:"url"`
	Context     string    `json:"context"`
	Creator     struct {
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
	} `json:"creator"`
}

//RateLimit represents Rate Limit API
type RateLimit struct {
	Resources struct {
		Core struct {
			Limit     int `json:"limit"`
			Remaining int `json:"remaining"`
			Reset     int `json:"reset"`
		} `json:"core"`
		Search struct {
			Limit     int `json:"limit"`
			Remaining int `json:"remaining"`
			Reset     int `json:"reset"`
		} `json:"search"`
	} `json:"resources"`
	Rate struct {
		Limit     int `json:"limit"`
		Remaining int `json:"remaining"`
		Reset     int `json:"reset"`
	} `json:"rate"`
}

func (r *RateLimit) String() string {
	return fmt.Sprintf("Limit: %d - Remaining: %d - Reset: %d", r.Rate.Limit, r.Rate.Remaining, r.Rate.Reset)
}

// Cursor represents cursor from github api
type Cursor struct {
	Label string     `json:"label"`
	Ref   string     `json:"ref"`
	Sha   string     `json:"sha"`
	User  User       `json:"user"`
	Repo  Repository `json:"repo"`
}

// PullRequest represents pull request from github api
type PullRequest struct {
	URL                 string    `json:"url"`
	ID                  int       `json:"id"`
	HTMLURL             string    `json:"html_url"`
	DiffURL             string    `json:"diff_url"`
	PatchURL            string    `json:"patch_url"`
	IssueURL            string    `json:"issue_url"`
	Number              int       `json:"number"`
	State               string    `json:"state"`
	Locked              bool      `json:"locked"`
	Title               string    `json:"title"`
	User                User      `json:"user"`
	Body                string    `json:"body"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	ClosedAt            time.Time `json:"closed_at"`
	MergedAt            time.Time `json:"merged_at"`
	MergeCommitSha      string    `json:"merge_commit_sha"`
	CommitsURL          string    `json:"commits_url"`
	ReviewCommentsURL   string    `json:"review_comments_url"`
	ReviewCommentURL    string    `json:"review_comment_url"`
	CommentsURL         string    `json:"comments_url"`
	StatusesURL         string    `json:"statuses_url"`
	Head                Cursor    `json:"head"`
	Base                Cursor    `json:"base"`
	Merged              bool      `json:"merged"`
	Mergeable           bool      `json:"mergeable"`
	Rebaseable          bool      `json:"rebaseable"`
	MergeableState      string    `json:"mergeable_state"`
	Comments            int       `json:"comments"`
	ReviewComments      int       `json:"review_comments"`
	MaintainerCanModify bool      `json:"maintainer_can_modify"`
	Commits             int       `json:"commits"`
	Additions           int       `json:"additions"`
	Deletions           int       `json:"deletions"`
	ChangedFiles        int       `json:"changed_files"`
}

// ReleaseRequest Request sent to Github to create a release
type ReleaseRequest struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
}

// ReleaseResponse Response return by Github after release creation
type ReleaseResponse struct {
	ID        int64  `json:"id"`
	UploadURL string `json:"upload_url"`
}

// RepositoryInvitation allows users or external services to invite other users to collaborate on a repo. The invited users (or external services on behalf of invited users) can choose to accept or decline the invitations.
type RepositoryInvitation struct {
	ID         int `json:"id"`
	Repository struct {
		ID     int    `json:"id"`
		NodeID string `json:"node_id"`
		Owner  struct {
			Login             string `json:"login"`
			ID                int    `json:"id"`
			NodeID            string `json:"node_id"`
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
		} `json:"owner"`
		Name             string      `json:"name"`
		FullName         string      `json:"full_name"`
		Description      string      `json:"description"`
		Private          bool        `json:"private"`
		Fork             bool        `json:"fork"`
		URL              string      `json:"url"`
		HTMLURL          string      `json:"html_url"`
		ArchiveURL       string      `json:"archive_url"`
		AssigneesURL     string      `json:"assignees_url"`
		BlobsURL         string      `json:"blobs_url"`
		BranchesURL      string      `json:"branches_url"`
		CloneURL         string      `json:"clone_url"`
		CollaboratorsURL string      `json:"collaborators_url"`
		CommentsURL      string      `json:"comments_url"`
		CommitsURL       string      `json:"commits_url"`
		CompareURL       string      `json:"compare_url"`
		ContentsURL      string      `json:"contents_url"`
		ContributorsURL  string      `json:"contributors_url"`
		DeploymentsURL   string      `json:"deployments_url"`
		DownloadsURL     string      `json:"downloads_url"`
		EventsURL        string      `json:"events_url"`
		ForksURL         string      `json:"forks_url"`
		GitCommitsURL    string      `json:"git_commits_url"`
		GitRefsURL       string      `json:"git_refs_url"`
		GitTagsURL       string      `json:"git_tags_url"`
		GitURL           string      `json:"git_url"`
		HooksURL         string      `json:"hooks_url"`
		IssueCommentURL  string      `json:"issue_comment_url"`
		IssueEventsURL   string      `json:"issue_events_url"`
		IssuesURL        string      `json:"issues_url"`
		KeysURL          string      `json:"keys_url"`
		LabelsURL        string      `json:"labels_url"`
		LanguagesURL     string      `json:"languages_url"`
		MergesURL        string      `json:"merges_url"`
		MilestonesURL    string      `json:"milestones_url"`
		MirrorURL        string      `json:"mirror_url"`
		NotificationsURL string      `json:"notifications_url"`
		PullsURL         string      `json:"pulls_url"`
		ReleasesURL      string      `json:"releases_url"`
		SSHURL           string      `json:"ssh_url"`
		StargazersURL    string      `json:"stargazers_url"`
		StatusesURL      string      `json:"statuses_url"`
		SubscribersURL   string      `json:"subscribers_url"`
		SubscriptionURL  string      `json:"subscription_url"`
		SvnURL           string      `json:"svn_url"`
		TagsURL          string      `json:"tags_url"`
		TeamsURL         string      `json:"teams_url"`
		TreesURL         string      `json:"trees_url"`
		Homepage         string      `json:"homepage"`
		Language         interface{} `json:"language"`
		ForksCount       int         `json:"forks_count"`
		StargazersCount  int         `json:"stargazers_count"`
		WatchersCount    int         `json:"watchers_count"`
		Size             int         `json:"size"`
		DefaultBranch    string      `json:"default_branch"`
		OpenIssuesCount  int         `json:"open_issues_count"`
		Topics           []string    `json:"topics"`
		HasIssues        bool        `json:"has_issues"`
		HasWiki          bool        `json:"has_wiki"`
		HasPages         bool        `json:"has_pages"`
		HasDownloads     bool        `json:"has_downloads"`
		Archived         bool        `json:"archived"`
		PushedAt         time.Time   `json:"pushed_at"`
		CreatedAt        time.Time   `json:"created_at"`
		UpdatedAt        time.Time   `json:"updated_at"`
		Permissions      struct {
			Admin bool `json:"admin"`
			Push  bool `json:"push"`
			Pull  bool `json:"pull"`
		} `json:"permissions"`
		AllowRebaseMerge bool `json:"allow_rebase_merge"`
		AllowSquashMerge bool `json:"allow_squash_merge"`
		AllowMergeCommit bool `json:"allow_merge_commit"`
		SubscribersCount int  `json:"subscribers_count"`
		NetworkCount     int  `json:"network_count"`
	} `json:"repository"`
	Invitee struct {
		Login             string `json:"login"`
		ID                int    `json:"id"`
		NodeID            string `json:"node_id"`
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
	} `json:"invitee"`
	Inviter struct {
		Login             string `json:"login"`
		ID                int    `json:"id"`
		NodeID            string `json:"node_id"`
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
	} `json:"inviter"`
	Permissions string `json:"permissions"`
	CreatedAt   string `json:"created_at"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
}
