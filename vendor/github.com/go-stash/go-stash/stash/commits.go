package stash

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type Author struct {
	Name  string `json:"name"`
	Email string `json:"emailAddress"`
}

type CommitsResponse struct {
	Values        []Commit `json:"values"`
	Size          int      `json:"size"`
	NextPageStart int      `json:"nextPageStart"`
	IsLastPage    bool     `json:"isLastPage"`
}

type Commit struct {
	Hash      string  `json:"Id"`
	Author    *Author `json:"author"`
	Timestamp int64   `json:"authorTimestamp"`
	Message   string  `json:"message"`
}

type CommitResource struct {
	client *Client
}

type Status struct {
	Description string `json:"description"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	State       string `json:"state"`
	URL         string `json:"url"`
}

//Get commit data for commit hash
func (r *CommitResource) Get(project, slug, commitId string) (*Commit, error) {
	commit := Commit{}
	path := fmt.Sprintf("/projects/%s/repos/%s/commits/%s", project, slug,
		commitId)

	if err := r.client.do("GET", "core", path, nil, nil, &commit); err != nil {
		return nil, err
	}

	return &commit, nil
}

// SetStatus set a build status.
// doc: https://developer.atlassian.com/bitbucket/server/docs/latest/how-tos/updating-build-status-for-commits.html
func (r *CommitResource) SetStatus(sha1 string, status Status) error {
	values, err := json.Marshal(status)
	if err != nil {
		return err
	}
	return r.client.do("POST", "build-status", fmt.Sprintf("/commits/%s", sha1), nil, values, nil)
}

//GetBetween returns commit data from a given starting commit, between two commits
//The commits may be identified by branch or tag name or by hash.
func (r *CommitResource) GetBetween(project, slug, since, until string) ([]Commit, error) {
	response := CommitsResponse{}
	commits := []Commit{}
	path := fmt.Sprintf("/projects/%s/repos/%s/commits", project, slug)
	params := url.Values{}
	if since != "" {
		params.Add("since", since)
	}
	if until != "" {
		params.Add("until", until)
	}

	for {
		if response.NextPageStart != 0 {
			params.Set("start", fmt.Sprintf("%d", response.NextPageStart))
		}

		if err := r.client.do("GET", "core", path, params, nil, &response); err != nil {
			return nil, err
		}

		commits = append(commits, response.Values...)
		if response.IsLastPage {
			break
		}
	}

	return commits, nil
}
