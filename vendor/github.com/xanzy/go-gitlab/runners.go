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
	"fmt"
	"net/url"
)

// RunnersService handles communication with the runner related methods of the
// GitLab API.
//
// GitLab API docs: https://docs.gitlab.com/ce/api/runners.html
type RunnersService struct {
	client *Client
}

// Runner represents a GitLab CI Runner.
//
// GitLab API docs: https://docs.gitlab.com/ce/api/runners.html
type Runner struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
	IsShared    bool   `json:"is_shared"`
	Name        string `json:"name"`
	Online      bool   `json:"online"`
	Status      string `json:"status"`
}

// ListRunnersOptions represents the available ListRunners() options.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/runners.html#list-owned-runners
type ListRunnersOptions struct {
	ListOptions
	Scope *string `url:"scope,omitempty" json:"scope,omitempty"`
}

// ListRunners gets a list of runners accessible by the authenticated user.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/runners.html#list-owned-runners
func (s *RunnersService) ListRunners(opt *ListRunnersOptions, options ...OptionFunc) ([]*Runner, *Response, error) {
	req, err := s.client.NewRequest("GET", "runners", opt, options)
	if err != nil {
		return nil, nil, err
	}

	var rs []*Runner
	resp, err := s.client.Do(req, &rs)
	if err != nil {
		return nil, resp, err
	}

	return rs, resp, err
}

// ListAllRunners gets a list of all runners in the GitLab instance. Access is
// restricted to users with admin privileges.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/runners.html#list-all-runners
func (s *RunnersService) ListAllRunners(opt *ListRunnersOptions, options ...OptionFunc) ([]*Runner, *Response, error) {
	req, err := s.client.NewRequest("GET", "runners/all", opt, options)
	if err != nil {
		return nil, nil, err
	}

	var rs []*Runner
	resp, err := s.client.Do(req, &rs)
	if err != nil {
		return nil, resp, err
	}

	return rs, resp, err
}

// ListProjectRunnersOptions represents the available ListProjectRunners()
// options.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/runners.html#list-project-s-runners
type ListProjectRunnersOptions ListRunnersOptions

// ListProjectRunners gets a list of runners accessible by the authenticated user.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/runners.html#list-project-s-runners
func (s *RunnersService) ListProjectRunners(pid interface{}, opt *ListProjectRunnersOptions, options ...OptionFunc) ([]*Runner, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/runners", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	var rs []*Runner
	resp, err := s.client.Do(req, &rs)
	if err != nil {
		return nil, resp, err
	}

	return rs, resp, err
}

// EnableProjectRunnerOptions represents the available EnableProjectRunner()
// options.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/runners.html#enable-a-runner-in-project
type EnableProjectRunnerOptions struct {
	RunnerID int `json:"runner_id"`
}

// EnableProjectRunner enables an available specific runner in the project.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/runners.html#enable-a-runner-in-project
func (s *RunnersService) EnableProjectRunner(pid interface{}, opt *EnableProjectRunnerOptions, options ...OptionFunc) (*Runner, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/runners", url.QueryEscape(project))

	req, err := s.client.NewRequest("POST", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	var r *Runner
	resp, err := s.client.Do(req, &r)
	if err != nil {
		return nil, resp, err
	}

	return r, resp, err
}
