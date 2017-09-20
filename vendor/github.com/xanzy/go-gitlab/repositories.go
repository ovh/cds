//
// Copyright 2017, Sander van Harmelen
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package gitlab

import (
	"bytes"
	"fmt"
	"net/url"
)

// RepositoriesService handles communication with the repositories related
// methods of the GitLab API.
//
// GitLab API docs: https://docs.gitlab.com/ce/api/repositories.html
type RepositoriesService struct {
	client *Client
}

// TreeNode represents a GitLab repository file or directory.
//
// GitLab API docs: https://docs.gitlab.com/ce/api/repositories.html
type TreeNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
	Mode string `json:"mode"`
}

func (t TreeNode) String() string {
	return Stringify(t)
}

// ListTreeOptions represents the available ListTree() options.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/repositories.html#list-repository-tree
type ListTreeOptions struct {
	Path      *string `url:"path,omitempty" json:"path,omitempty"`
	Ref       *string `url:"ref,omitempty" json:"ref,omitempty"`
	Recursive *bool   `url:"recursive,omitempty" json:"recursive,omitempty"`
}

// ListTree gets a list of repository files and directories in a project.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/repositories.html#list-repository-tree
func (s *RepositoriesService) ListTree(pid interface{}, opt *ListTreeOptions, options ...OptionFunc) ([]*TreeNode, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/tree", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	var t []*TreeNode
	resp, err := s.client.Do(req, &t)
	if err != nil {
		return nil, resp, err
	}

	return t, resp, err
}

// RawFileContent gets the raw file contents for a file by commit SHA and path
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/repositories.html#raw-file-content
func (s *RepositoriesService) RawFileContent(pid interface{}, sha string, options ...OptionFunc) ([]byte, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/blobs/%s", url.QueryEscape(project), sha)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	var b bytes.Buffer
	resp, err := s.client.Do(req, &b)
	if err != nil {
		return nil, resp, err
	}

	return b.Bytes(), resp, err
}

// RawBlobContent gets the raw file contents for a blob by blob SHA.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/repositories.html#raw-blob-content
func (s *RepositoriesService) RawBlobContent(pid interface{}, sha string, options ...OptionFunc) ([]byte, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/blobs/%s/raw", url.QueryEscape(project), sha)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	var b bytes.Buffer
	resp, err := s.client.Do(req, &b)
	if err != nil {
		return nil, resp, err
	}

	return b.Bytes(), resp, err
}

// ArchiveOptions represents the available Archive() options.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/repositories.html#get-file-archive
type ArchiveOptions struct {
	SHA *string `url:"sha,omitempty" json:"sha,omitempty"`
}

// Archive gets an archive of the repository.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/repositories.html#get-file-archive
func (s *RepositoriesService) Archive(pid interface{}, opt *ArchiveOptions, options ...OptionFunc) ([]byte, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/archive", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	var b bytes.Buffer
	resp, err := s.client.Do(req, &b)
	if err != nil {
		return nil, resp, err
	}

	return b.Bytes(), resp, err
}

// Compare represents the result of a comparison of branches, tags or commits.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/repositories.html#compare-branches-tags-or-commits
type Compare struct {
	Commit         *Commit   `json:"commit"`
	Commits        []*Commit `json:"commits"`
	Diffs          []*Diff   `json:"diffs"`
	CompareTimeout bool      `json:"compare_timeout"`
	CompareSameRef bool      `json:"compare_same_ref"`
}

func (c Compare) String() string {
	return Stringify(c)
}

// CompareOptions represents the available Compare() options.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/repositories.html#compare-branches-tags-or-commits
type CompareOptions struct {
	From *string `url:"from,omitempty" json:"from,omitempty"`
	To   *string `url:"to,omitempty" json:"to,omitempty"`
}

// Compare compares branches, tags or commits.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/repositories.html#compare-branches-tags-or-commits
func (s *RepositoriesService) Compare(pid interface{}, opt *CompareOptions, options ...OptionFunc) (*Compare, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/compare", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	c := new(Compare)
	resp, err := s.client.Do(req, c)
	if err != nil {
		return nil, resp, err
	}

	return c, resp, err
}

// Contributor represents a GitLap contributor.
//
// GitLab API docs: https://docs.gitlab.com/ce/api/repositories.html#contributer
type Contributor struct {
	Name      string `json:"name,omitempty"`
	Email     string `json:"email,omitempty"`
	Commits   int    `json:"commits,omitempty"`
	Additions int    `json:"additions,omitempty"`
	Deletions int    `json:"deletions,omitempty"`
}

func (c Contributor) String() string {
	return Stringify(c)
}

// Contributors gets the repository contributors list.
//
// GitLab API docs: https://docs.gitlab.com/ce/api/repositories.html#contributer
func (s *RepositoriesService) Contributors(pid interface{}, options ...OptionFunc) ([]*Contributor, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/contributors", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	var c []*Contributor
	resp, err := s.client.Do(req, &c)
	if err != nil {
		return nil, resp, err
	}

	return c, resp, err
}
