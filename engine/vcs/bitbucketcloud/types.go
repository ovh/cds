package bitbucketcloud

import (
	"regexp"
	"time"

	"github.com/ovh/cds/sdk"
)

var (
	_                    sdk.VCSAuthorizedClient = &bitbucketcloudClient{}
	_                    sdk.VCSServer           = &bitbucketcloudConsumer{}
	rawEmailCommitRegexp                         = regexp.MustCompile(`<(.*)>`)
)

// WebhookCreate represent struct to create a webhook
type WebhookCreate struct {
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Active      bool     `json:"active"`
	Events      []string `json:"events"`
}

type Webhook struct {
	ReadOnly    bool   `json:"read_only"`
	Description string `json:"description"`
	Links       struct {
		Self Link `json:"self"`
	} `json:"links"`
	URL                  string    `json:"url"`
	CreatedAt            time.Time `json:"created_at"`
	SkipCertVerification bool      `json:"skip_cert_verification"`
	Source               string    `json:"source"`
	HistoryEnabled       bool      `json:"history_enabled"`
	Active               bool      `json:"active"`
	Subject              struct {
		Links struct {
			Self   Link `json:"self"`
			HTML   Link `json:"html"`
			Avatar Link `json:"avatar"`
		} `json:"links"`
		Type     string `json:"type"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		UUID     string `json:"uuid"`
	} `json:"subject"`
	Type   string   `json:"type"`
	Events []string `json:"events"`
	UUID   string   `json:"uuid"`
}

type Webhooks struct {
	Pagelen  int       `json:"pagelen"`
	Page     int       `json:"page"`
	Size     int64     `json:"size"`
	Values   []Webhook `json:"values"`
	Next     string    `json:"next"`
	Previous string    `json:"previous,omitempty"`
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

// PullRequest represents pull request from github api
type PullRequest struct {
	Description string `json:"description"`
	Links       struct {
		Decline  Link `json:"decline"`
		Commits  Link `json:"commits"`
		Self     Link `json:"self"`
		Comments Link `json:"comments"`
		Merge    Link `json:"merge"`
		HTML     Link `json:"html"`
		Activity Link `json:"activity"`
		Diff     Link `json:"diff"`
		Approve  Link `json:"approve"`
		Statuses Link `json:"statuses"`
	} `json:"links"`
	Title             string `json:"title"`
	CloseSourceBranch bool   `json:"close_source_branch"`
	Type              string `json:"type"`
	ID                int    `json:"id"`
	Destination       struct {
		Commit struct {
			Hash  string `json:"hash"`
			Type  string `json:"type"`
			Links struct {
				Self Link `json:"self"`
				HTML Link `json:"html"`
			} `json:"links"`
		} `json:"commit"`
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
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"destination"`
	CreatedOn time.Time `json:"created_on"`
	Summary   struct {
		Raw    string `json:"raw"`
		Markup string `json:"markup"`
		HTML   string `json:"html"`
		Type   string `json:"type"`
	} `json:"summary"`
	Source struct {
		Commit struct {
			Hash  string `json:"hash"`
			Type  string `json:"type"`
			Links struct {
				Self Link `json:"self"`
				HTML Link `json:"html"`
			} `json:"links"`
		} `json:"commit"`
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
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"source"`
	CommentCount int       `json:"comment_count"`
	State        string    `json:"state"`
	TaskCount    int       `json:"task_count"`
	Reason       string    `json:"reason"`
	UpdatedOn    time.Time `json:"updated_on"`
	Author       User      `json:"author"`
	MergeCommit  struct {
		Hash string `json:"hash"`
	} `json:"merge_commit"`
}

type PullRequests struct {
	Pagelen  int           `json:"pagelen"`
	Page     int           `json:"page"`
	Size     int64         `json:"size"`
	Values   []PullRequest `json:"values"`
	Next     string        `json:"next"`
	Previous string        `json:"previous,omitempty"`
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

// Workspace represent a workspace inside bitbucket cloud. https://developer.atlassian.com/cloud/bitbucket/rest/api-group-workspaces/#api-user-permissions-workspaces-get
type Workspace struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	Type string `json:"type"`
}

type Workspaces struct {
	Pagelen  int         `json:"pagelen"`
	Page     int         `json:"page"`
	Size     int64       `json:"size"`
	Values   []Workspace `json:"values"`
	Next     string      `json:"next"`
	Previous string      `json:"previous,omitempty"`
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

type Statuses struct {
	Pagelen  int      `json:"pagelen"`
	Page     int      `json:"page"`
	Size     int64    `json:"size"`
	Values   []Status `json:"values"`
	Next     string   `json:"next"`
	Previous string   `json:"previous,omitempty"`
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
		Commits Link `json:"commits"`
		Self    Link `json:"self"`
		HTML    Link `json:"html"`
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

type Tags struct {
	Pagelen  int    `json:"pagelen"`
	Page     int    `json:"page"`
	Size     int64  `json:"size"`
	Values   []Tag  `json:"values"`
	Next     string `json:"next"`
	Previous string `json:"previous,omitempty"`
}

type Tag struct {
	Name  string `json:"name"`
	Links struct {
		Commits Link `json:"commits"`
		Self    Link `json:"self"`
		HTML    Link `json:"html"`
	} `json:"links"`
	Date    time.Time `json:"date"`
	Message string    `json:"message"`
	Type    string    `json:"type"`
	Target  struct {
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
			User struct {
				Username    string `json:"username"`
				DisplayName string `json:"display_name"`
				UUID        string `json:"uuid"`
				Links       struct {
					Self struct {
						Href string `json:"href"`
					} `json:"self"`
					HTML struct {
						Href string `json:"href"`
					} `json:"html"`
					Avatar struct {
						Href string `json:"href"`
					} `json:"avatar"`
				} `json:"links"`
				Nickname  string `json:"nickname"`
				Type      string `json:"type"`
				AccountID string `json:"account_id"`
			} `json:"user"`
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
