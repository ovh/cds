package bitbucketserver

import (
	"github.com/ovh/cds/sdk"
)

var (
	_ sdk.VCSAuthorizedClient = &bitbucketClient{}
	_ sdk.VCSServer           = &bitbucketConsumer{}
)

// WebHook Represent a webhook in bitbucket model
type WebHook struct {
	ID            int               `json:"id,omitempty"`
	Active        bool              `json:"active"`
	Configuration map[string]string `json:"configuration"`
	Events        []string          `json:"events"`
	Name          string            `json:"name"`
	URL           string            `json:"url"`
}

// GetWebHooksResponse represent the response send by bitbucket when listing webhooks
type GetWebHooksResponse struct {
	Values []WebHook `json:"values"`
}

type Branch struct {
	ID         string `json:"id"`
	DisplayID  string `json:"displayId"`
	LatestHash string `json:"latestChangeset"`
	IsDefault  bool   `json:"isDefault"`
}

type BranchResponse struct {
	Values     []Branch `json:"values"`
	Size       int      `json:"size"`
	IsLastPage bool     `json:"isLastPage"`
}

type Tag struct {
	ID              string `json:"id"`
	DisplayID       string `json:"displayId"`
	Type            string `json:"type"`
	LatestCommit    string `json:"latestCommit"`
	LatestChangeset string `json:"latestChangeset"`
	Hash            string `json:"hash"`
}

type TagResponse struct {
	Values     []Tag `json:"values"`
	Size       int   `json:"size"`
	IsLastPage bool  `json:"isLastPage"`
}

type Author struct {
	Name        string `json:"name"`
	Email       string `json:"emailAddress"`
	DisplayName string `json:"displayName"`
	Slug        string `json:"slug"`
}

type CommitsResponse struct {
	Values        []Commit `json:"values"`
	Size          int      `json:"size"`
	NextPageStart int      `json:"nextPageStart"`
	IsLastPage    bool     `json:"isLastPage"`
}

type Commit struct {
	Hash      string `json:"id"`
	Author    Author `json:"author"`
	Timestamp int64  `json:"authorTimestamp"`
	Message   string `json:"message"`
}

type Status struct {
	Description string `json:"description"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	State       string `json:"state"`
	URL         string `json:"url"`
	Timestamp   int64  `json:"dateAdded"`
}

type Lines struct {
	Text string `json:"text"`
}

type Content struct {
	Lines []Lines `json:"lines"`
}

type HookDetail struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Description   string `json:"description"`
	Version       string `json:"version"`
	ConfigFormKey string `json:"configFormKey"`
}

type Hook struct {
	Enabled bool        `json:"enabled"`
	Details *HookDetail `json:"details"`
}

type Key struct {
	ID    int64  `json:"id"`
	Text  string `json:"text"`
	Label string `json:"label"`
}

type Keys struct {
	Values []Key `json:"values"`
}

type Response struct {
	Values        []Repo `json:"values"`
	Size          int    `json:"size"`
	NextPageStart int    `json:"nextPageStart"`
	IsLastPage    bool   `json:"isLastPage"`
}

type ResponseStatus struct {
	Values        []Status `json:"values"`
	Size          int      `json:"size"`
	NextPageStart int      `json:"nextPageStart"`
	IsLastPage    bool     `json:"isLastPage"`
}

type Repo struct {
	Name    string                      `json:"name"`
	Slug    string                      `json:"slug"`
	Public  bool                        `json:"public"`
	ScmID   string                      `json:"scmId"`
	Project *sdk.BitbucketServerProject `json:"project"`
	Link    *Link                       `json:"link"`
	Links   *Links                      `json:"links"`
}

type Links struct {
	Clone []Clone `json:"clone"`
	Self  []Clone `json:"self"`
}

type Clone struct {
	URL  string `json:"href"`
	Name string `json:"name"`
}

type Link struct {
	URL string `json:"url"`
	Rel string `json:"rel"`
}

type UsersResponse struct {
	Values        []sdk.BitbucketServerActor `json:"values"`
	Size          int                        `json:"size"`
	NextPageStart int                        `json:"nextPageStart"`
	IsLastPage    bool                       `json:"isLastPage"`
}

type PullRequestResponse struct {
	Values        []sdk.BitbucketServerPullRequest `json:"values"`
	Size          int                              `json:"size"`
	NextPageStart int                              `json:"nextPageStart"`
	IsLastPage    bool                             `json:"isLastPage"`
}

type UsersPermissionResponse struct {
	Values        []UserPermission `json:"values"`
	Size          int              `json:"size"`
	NextPageStart int              `json:"nextPageStart"`
	IsLastPage    bool             `json:"isLastPage"`
}

type UserPermission struct {
	User       sdk.BitbucketServerActor `json:"user"`
	Permission string                   `json:"permission"`
}
