package stash

import (
	"fmt"
	"net/url"
)

type Branch struct {
	ID         string `json:"id"`
	DisplayID  string `json:"displayId"`
	LatestHash string `json:"latestChangeset"`
	IsDefault  bool   `json:"isDefault"`
}

type BranchResource struct {
	client *Client
}

type BranchResponse struct {
	Values        []Branch `json:"values"`
	Size          int    `json:"size"`
	IsLastPage    bool   `json:"isLastPage"`
}

// List list of branches for repo
func (r *BranchResource) List(project, slug string) ([]Branch, error) {
	branches := []Branch{}

	path := fmt.Sprintf("/projects/%s/repos/%s/branches", project, slug)
	params := url.Values{}

	nextPage := 0
	for {
		if nextPage != 0 {
			params.Set("start", fmt.Sprintf("%d", nextPage))
		}

		var response BranchResponse
		if err := r.client.do("GET", "core", path, params, nil, &response); err != nil {
			return nil, err
		}

		branches = append(branches, response.Values...)

		if response.IsLastPage {
			break
		} else {
			nextPage += response.Size
		}
	}
	return branches, nil
}

// Find a branches for repo
func (r *BranchResource) Find(project, slug, filter string) (Branch, error) {
	branches := BranchResponse{}
	path := fmt.Sprintf("/projects/%s/repos/%s/branches?filterText=%s", project, slug, url.QueryEscape(filter))

	if err := r.client.do("GET", "core", path, nil, nil, &branches); err != nil {
		return Branch{}, err
	}

	if len(branches.Values) > 0 {
		return branches.Values[0], nil
	}
	return Branch{}, nil
}
