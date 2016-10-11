package stash

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

type Reviewer struct {
	User     *User  `json:"user"`
	Role     string `json:"role"`
	Approved bool   `json:"approved"`
}

// PullRequestReference is a reference to a repository
type PullRequestReference struct {
	Id              string `json:"id"`
	DisplayId       string `json:"displayId"`
	LatestChangeset string `json:"latestChangeset"`
	Repository      *Repo  `json:"repository"`
}

// PullRequest represents a pull request on stash
type PullRequest struct {
	Id           int                   `json:"id"`
	Version      int                   `json:"version"`
	Title        string                `json:"title"`
	Description  string                `json:"description"`
	State        string                `json:"state"`
	Open         bool                  `json:"open"`
	Closed       bool                  `json:"closed"`
	FromRef      *PullRequestReference `json:"fromRef"`
	ToRef        *PullRequestReference `json:"toRef"`
	Locked       bool                  `json:"locked"`
	Author       *Reviewer             `json:"author,omitempty"`
	Reviewers    []*Reviewer           `json:"reviewers"`
	Participants []*Reviewer           `json:"participants,omitempty"`
	Link         *Link                 `json:"link"`
	Links        *Links                `json:"links"`
}

// PullRequestList is the response from listing pull requests
type PullRequestList struct {
	Values        []*PullRequest `json:"values"`
	Size          int            `json:"size"`
	IsLastPage    bool           `json:"isLastPage"`
	NextPageStart int            `json:"nextPageStart"`
}

// PullRequestResource
type PullRequestResource struct {
	client *Client
}

// Create creates a new pull request for the given project/slug repository, from branch fromRef to branch toRef and optional reviewers
func (r *PullRequestResource) Create(project, slug, title, fromRef, toRef string, reviewers []string) (*PullRequest, error) {
	pr := &PullRequest{
		Title: title,
		Open:  true,
		FromRef: &PullRequestReference{
			Id: fromRef,
			Repository: &Repo{
				Slug: slug,
				Project: &Project{
					Key: project,
				},
			},
		},
		ToRef: &PullRequestReference{
			Id: toRef,
			Repository: &Repo{
				Slug: slug,
				Project: &Project{
					Key: project,
				},
			},
		},
		Locked: false,
	}

	for _, reviewer := range reviewers {
		pr.Reviewers = append(pr.Reviewers, &Reviewer{User: &User{Username: reviewer}})
	}

	prData, err := json.Marshal(pr)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/projects/%s/repos/%s/pull-requests", project, slug)

	if err := r.client.do("POST", "core", path, nil, prData, &pr); err != nil {
		return nil, err
	}

	return pr, nil
}

// List lists all the pull requests of the projects
func (r *PullRequestResource) List(project, slug, direction, at, state, order string, woAttrs, woProps bool) ([]*PullRequest, error) {
	path := fmt.Sprintf("/projects/%s/repos/%s/pull-requests", project, slug)
	params := url.Values{}

	// defaults to incoming
	if direction != "" {
		params.Set("direction", direction)
	}

	if at != "" {
		params.Set("at", at)
	}

	// defaults to open
	if state != "" {
		params.Set("state", state)
	}

	if woAttrs {
		params.Set("withAttributes", strconv.FormatBool(false))
	}

	if woProps {
		params.Set("withProperties", strconv.FormatBool(false))
	}

	pullRequests := []*PullRequest{}
	nextPage := 0
	for {
		if nextPage != 0 {
			params.Set("start", fmt.Sprintf("%d", nextPage))
		}

		var resp *PullRequestList
		if err := r.client.do("GET", "core", path, params, nil, &resp); err != nil {
			return nil, err
		}

		pullRequests = append(pullRequests, resp.Values...)

		if resp.IsLastPage {
			break
		} else {
			nextPage = resp.NextPageStart
		}
	}

	return pullRequests, nil
}

// Get gets an existing pull request from its id
func (r *PullRequestResource) Get(project, slug string, pullRequestID int) (*PullRequest, error) {
	path := fmt.Sprintf("/projects/%s/repos/%s/pull-requests/%d", project, slug, pullRequestID)

	var pr *PullRequest
	if err := r.client.do("GET", "core", path, nil, nil, &pr); err != nil {
		return nil, err
	}

	return pr, nil
}
