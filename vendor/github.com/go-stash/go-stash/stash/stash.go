package stash

import (
	"errors"
	"fmt"
)

var (
	ErrNilClient = errors.New("client is nil")
)

// New creates an instance of the Stash Client
func New(apiUrl, consumerKey, accessToken, tokenSecret, privateKey string) *Client {
	c := &Client{}
	c.ApiUrl = apiUrl
	c.ConsumerKey = consumerKey
	c.ConsumerSecret = "dont't care"
	c.ConsumerPrivateKeyPem = privateKey
	c.AccessToken = accessToken
	c.TokenSecret = tokenSecret

	c.Repos = &RepoResource{c}
	c.Branches = &BranchResource{c}
	c.Commits = &CommitResource{c}
	c.Contents = &ContentResource{c}
	c.Hooks = &HookResource{c}
	c.RepoKeys = &RepoKeyResource{c}
	c.Keys = &KeyResource{c}
	c.Users = &UserResource{c}
	c.PullRequests = &PullRequestResource{c}
	return c
}

type Client struct {
	ApiUrl                string
	ConsumerKey           string
	ConsumerSecret        string
	ConsumerPrivateKeyPem string
	AccessToken           string
	TokenSecret           string

	Repos        *RepoResource
	Branches     *BranchResource
	Commits      *CommitResource
	Contents     *ContentResource
	Hooks        *HookResource
	RepoKeys     *RepoKeyResource
	Keys         *KeyResource
	Users        *UserResource
	PullRequests *PullRequestResource
}

// Guest Client that can be used to access
// public APIs that do not require authentication.
var Guest = New("", "", "", "", "")

func (c *Client) GetFullApiUrl(api string) string {
	var url string
	switch api {
	case "keys":
		url = fmt.Sprintf("%s/rest/keys/1.0", c.ApiUrl)
	case "ssh":
		url = fmt.Sprintf("%s/rest/ssh/1.0", c.ApiUrl)
	case "core":
		url = fmt.Sprintf("%s/rest/api/1.0", c.ApiUrl)
	case "build-status":
		url = fmt.Sprintf("%s/rest/build-status/1.0", c.ApiUrl)
	}

	return url
}
