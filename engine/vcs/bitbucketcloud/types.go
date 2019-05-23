package bitbucketcloud

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/ovh/cds/sdk"
)

var (
	_                    sdk.VCSAuthorizedClient = &bitbucketcloudClient{}
	_                    sdk.VCSServer           = &bitbucketcloudConsumer{}
	rawEmailCommitRegexp                         = regexp.MustCompile(`<(.*)>`)
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

type Webhook struct {
	ID      int      `json:"id"`
	URL     string   `json:"url"`
	TestURL string   `json:"test_url"`
	PingURL string   `json:"ping_url"`
	Name    string   `json:"name"`
	Events  []string `json:"events"`
	Active  bool     `json:"active"`
	Config  struct {
		URL         string `json:"url"`
		ContentType string `json:"content_type"`
	} `json:"config"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedAt time.Time `json:"created_at"`
}

// WebHookConfig represent the configuration of a webhook
type WebHookConfig struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
}

// User represents a public bitbucketcloud user.
type User struct {
	Username    string `json:"username"`
	Website     string `json:"website"`
	DisplayName string `json:"display_name"`
	UUID        string `json:"uuid"`
	Links       struct {
		Hooks        Link `json:"hooks"`
		Self         Link `json:"self"`
		Repositories Link `json:"repositories"`
		HTML         Link `json:"html"`
		Followers    Link `json:"followers"`
		Avatar       Link `json:"avatar"`
		Following    Link `json:"following"`
		Snippets     Link `json:"snippets"`
	} `json:"links"`
	Nickname      string    `json:"nickname"`
	CreatedOn     time.Time `json:"created_on"`
	IsStaff       bool      `json:"is_staff"`
	Location      string    `json:"location"`
	AccountStatus string    `json:"account_status"`
	Type          string    `json:"type"`
	AccountID     string    `json:"account_id"`
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

// Protection represents a repository branch's protection
type Protection struct {
	Enabled bool `json:"enabled,omitempty"`
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

// DiffCommits represent response from github api for a diff between commits
type DiffCommits struct {
	URL          string `json:"url"`
	HTMLURL      string `json:"html_url"`
	PermalinkURL string `json:"permalink_url"`
	DiffURL      string `json:"diff_url"`
	PatchURL     string `json:"patch_url"`
	BaseCommit   struct {
		URL         string `json:"url"`
		Sha         string `json:"sha"`
		NodeID      string `json:"node_id"`
		HTMLURL     string `json:"html_url"`
		CommentsURL string `json:"comments_url"`
		Commit      Commit `json:"commit"`
		Author      User   `json:"author"`
		Committer   User   `json:"committer"`
		Parents     []struct {
			URL string `json:"url"`
			Sha string `json:"sha"`
		} `json:"parents"`
	} `json:"base_commit"`
	MergeBaseCommit struct {
		URL         string `json:"url"`
		Sha         string `json:"sha"`
		NodeID      string `json:"node_id"`
		HTMLURL     string `json:"html_url"`
		CommentsURL string `json:"comments_url"`
		Commit      Commit `json:"commit"`
		Author      User   `json:"author"`
		Committer   User   `json:"committer"`
		Parents     []struct {
			URL string `json:"url"`
			Sha string `json:"sha"`
		} `json:"parents"`
	} `json:"merge_base_commit"`
	Status       string   `json:"status"`
	AheadBy      int      `json:"ahead_by"`
	BehindBy     int      `json:"behind_by"`
	TotalCommits int      `json:"total_commits"`
	Commits      []Commit `json:"commits"`
	Files        []struct {
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

type Ref struct {
	Ref    string `json:"ref"`
	NodeID string `json:"node_id"`
	URL    string `json:"url"`
	Object struct {
		Type string `json:"type"`
		Sha  string `json:"sha"`
		URL  string `json:"url"`
	} `json:"object"`
}

type AccessToken struct {
	AccessToken  string `json:"access_token"`
	Scopes       string `json:"scopes"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

type Link struct {
	Href string `json:"href"`
	Name string `json:"name"`
}

type Repositories struct {
	Pagelen  int          `json:"pagelen"`
	Page     int          `json:"page"`
	Size     int64        `json:"size"`
	Values   []Repository `json:"values"`
	Next     string       `json:"next"`
	Previous string       `json:"previous,omitempty"`
}

type Repository struct {
	Scm     string `json:"scm"`
	Website string `json:"website"`
	HasWiki bool   `json:"has_wiki"`
	Name    string `json:"name"`
	Links   struct {
		Watchers     Link   `json:"watchers"`
		Branches     Link   `json:"branches"`
		Tags         Link   `json:"tags"`
		Commits      Link   `json:"commits"`
		Clone        []Link `json:"clone"`
		Self         Link   `json:"self"`
		Source       Link   `json:"source"`
		HTML         Link   `json:"html"`
		Avatar       Link   `json:"avatar"`
		Hooks        Link   `json:"hooks"`
		Forks        Link   `json:"forks"`
		Downloads    Link   `json:"downloads"`
		Issues       Link   `json:"issues"`
		Pullrequests Link   `json:"pullrequests"`
	} `json:"links"`
	ForkPolicy string    `json:"fork_policy"`
	UUID       string    `json:"uuid"`
	Language   string    `json:"language"`
	CreatedOn  time.Time `json:"created_on"`
	Mainbranch struct {
		Type string `json:"type"`
		Name string `json:"name"`
	} `json:"mainbranch"`
	FullName    string    `json:"full_name"`
	HasIssues   bool      `json:"has_issues"`
	Owner       User      `json:"owner"`
	UpdatedOn   time.Time `json:"updated_on"`
	Size        int       `json:"size"`
	Type        string    `json:"type"`
	Slug        string    `json:"slug"`
	IsPrivate   bool      `json:"is_private"`
	Description string    `json:"description"`
}

type Status struct {
	UUID        string    `json:"uuid"`
	Key         string    `json:"key"`
	RefName     string    `json:"refname"` //optional
	URL         string    `json:"url"`
	State       string    `json:"state"` // SUCCESSFUL / FAILED / INPROGRESS / STOPPED
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedOn   time.Time `json:"created_on"`
	UpdatedOn   time.Time `json:"updated_on"`
	Links       struct {
		Self   Link `json:"self"`
		Commit Link `json:"commit"`
	} `json:"links"`
}

type Branches struct {
	Pagelen  int      `json:"pagelen"`
	Page     int      `json:"page"`
	Size     int64    `json:"size"`
	Values   []Branch `json:"values"`
	Next     string   `json:"next"`
	Previous string   `json:"previous,omitempty"`
}

type Branch struct {
	Heads []struct {
		Hash  string `json:"hash"`
		Type  string `json:"type"`
		Links Link   `json:"links"`
	} `json:"heads"`
	Name  string `json:"name"`
	Links struct {
		Commits struct {
			Href string `json:"href"`
		} `json:"commits"`
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
	DefaultMergeStrategy string   `json:"default_merge_strategy"`
	MergeStrategies      []string `json:"merge_strategies"`
	Type                 string   `json:"type"`
	Target               struct {
		Hash       string `json:"hash"`
		Repository struct {
			Links struct {
				Self   Link `json:"self"`
				HTML   Link `json:"html"`
				Avatar Link `json:"avatar"`
			} `json:"links"`
			Type     string `json:"type"`
			Name     string `json:"name"`
			FullName string `json:"full_name"`
			UUID     string `json:"uuid"`
		} `json:"repository"`
		Links struct {
			Self     Link `json:"self"`
			Comments Link `json:"comments"`
			Patch    Link `json:"patch"`
			HTML     Link `json:"html"`
			Diff     Link `json:"diff"`
			Approve  Link `json:"approve"`
			Statuses Link `json:"statuses"`
		} `json:"links"`
		Author struct {
			Raw  string `json:"raw"`
			Type string `json:"type"`
		} `json:"author"`
		Parents []struct {
			Hash  string `json:"hash"`
			Type  string `json:"type"`
			Links struct {
				Self Link `json:"self"`
				HTML Link `json:"html"`
			} `json:"links"`
		} `json:"parents"`
		Date    time.Time `json:"date"`
		Message string    `json:"message"`
		Type    string    `json:"type"`
	} `json:"target"`
}

type Commits struct {
	Pagelen  int      `json:"pagelen"`
	Page     int      `json:"page"`
	Size     int64    `json:"size"`
	Values   []Commit `json:"values"`
	Next     string   `json:"next"`
	Previous string   `json:"previous,omitempty"`
}

type Commit struct {
	Rendered struct {
		Message struct {
			Raw    string `json:"raw"`
			Markup string `json:"markup"`
			HTML   string `json:"html"`
			Type   string `json:"type"`
		} `json:"message"`
	} `json:"rendered"`
	Hash       string `json:"hash"`
	Repository struct {
		Links struct {
			Self   Link `json:"self"`
			HTML   Link `json:"html"`
			Avatar Link `json:"avatar"`
		} `json:"links"`
		Type     string `json:"type"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		UUID     string `json:"uuid"`
	} `json:"repository"`
	Links struct {
		Self     Link `json:"self"`
		Comments Link `json:"comments"`
		Patch    Link `json:"patch"`
		HTML     Link `json:"html"`
		Diff     Link `json:"diff"`
		Approve  Link `json:"approve"`
		Statuses Link `json:"statuses"`
	} `json:"links"`
	Author struct {
		Raw  string `json:"raw"`
		Type string `json:"type"`
		User User   `json:"user"`
	} `json:"author,omitempty"`
	Summary struct {
		Raw    string `json:"raw"`
		Markup string `json:"markup"`
		HTML   string `json:"html"`
		Type   string `json:"type"`
	} `json:"summary"`
	Parents []struct {
		Hash  string `json:"hash"`
		Type  string `json:"type"`
		Links Link   `json:"links"`
	} `json:"parents"`
	Date    time.Time `json:"date"`
	Message string    `json:"message"`
	Type    string    `json:"type"`
}
