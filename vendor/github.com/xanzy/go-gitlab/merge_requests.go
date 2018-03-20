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
	"time"
)

// MergeRequestsService handles communication with the merge requests related
// methods of the GitLab API.
//
// GitLab API docs: https://docs.gitlab.com/ce/api/merge_requests.html
type MergeRequestsService struct {
	client    *Client
	timeStats *timeStatsService
}

// MergeRequest represents a GitLab merge request.
//
// GitLab API docs: https://docs.gitlab.com/ce/api/merge_requests.html
type MergeRequest struct {
	ID           int        `json:"id"`
	IID          int        `json:"iid"`
	TargetBranch string     `json:"target_branch"`
	SourceBranch string     `json:"source_branch"`
	ProjectID    int        `json:"project_id"`
	Title        string     `json:"title"`
	State        string     `json:"state"`
	CreatedAt    *time.Time `json:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at"`
	Upvotes      int        `json:"upvotes"`
	Downvotes    int        `json:"downvotes"`
	Author       struct {
		ID        int        `json:"id"`
		Username  string     `json:"username"`
		Name      string     `json:"name"`
		State     string     `json:"state"`
		CreatedAt *time.Time `json:"created_at"`
	} `json:"author"`
	Assignee struct {
		ID        int        `json:"id"`
		Username  string     `json:"username"`
		Name      string     `json:"name"`
		State     string     `json:"state"`
		CreatedAt *time.Time `json:"created_at"`
	} `json:"assignee"`
	SourceProjectID           int        `json:"source_project_id"`
	TargetProjectID           int        `json:"target_project_id"`
	Labels                    []string   `json:"labels"`
	Description               string     `json:"description"`
	WorkInProgress            bool       `json:"work_in_progress"`
	Milestone                 *Milestone `json:"milestone"`
	MergeWhenPipelineSucceeds bool       `json:"merge_when_pipeline_succeeds"`
	MergeStatus               string     `json:"merge_status"`
	Subscribed                bool       `json:"subscribed"`
	SHA                       string     `json:"sha"`
	MergeCommitSHA            string     `json:"merge_commit_sha"`
	UserNotesCount            int        `json:"user_notes_count"`
	ChangesCount              string     `json:"changes_count"`
	SouldRemoveSourceBranch   bool       `json:"should_remove_source_branch"`
	ForceRemoveSourceBranch   bool       `json:"force_remove_source_branch"`
	WebURL                    string     `json:"web_url"`
	DiscussionLocked          bool       `json:"discussion_locked"`
	Changes                   []struct {
		OldPath     string `json:"old_path"`
		NewPath     string `json:"new_path"`
		AMode       string `json:"a_mode"`
		BMode       string `json:"b_mode"`
		Diff        string `json:"diff"`
		NewFile     bool   `json:"new_file"`
		RenamedFile bool   `json:"renamed_file"`
		DeletedFile bool   `json:"deleted_file"`
	} `json:"changes"`
	TimeStats *TimeStats `json:"time_stats"`
}

func (m MergeRequest) String() string {
	return Stringify(m)
}

// MergeRequestApprovals represents GitLab merge request approvals.
//
// GitLab API docs:
// https://docs.gitlab.com/ee/api/merge_requests.html#merge-request-approvals
type MergeRequestApprovals struct {
	ID                int        `json:"id"`
	ProjectID         int        `json:"project_id"`
	Title             string     `json:"title"`
	Description       string     `json:"description"`
	State             string     `json:"state"`
	CreatedAt         *time.Time `json:"created_at"`
	UpdatedAt         *time.Time `json:"updated_at"`
	MergeStatus       string     `json:"merge_status"`
	ApprovalsRequired int        `json:"approvals_required"`
	ApprovalsMissing  int        `json:"approvals_missing"`
	ApprovedBy        []struct {
		User struct {
			Name      string `json:"name"`
			Username  string `json:"username"`
			ID        int    `json:"id"`
			State     string `json:"state"`
			AvatarURL string `json:"avatar_url"`
			WebURL    string `json:"web_url"`
		} `json:"user"`
	} `json:"approved_by"`
}

func (m MergeRequestApprovals) String() string {
	return Stringify(m)
}

// MergeRequestDiffVersion represents Gitlab merge request version.
//
// Gitlab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#get-a-single-mr-diff-version
type MergeRequestDiffVersion struct {
	ID             int        `json:"id"`
	HeadCommitSHA  string     `json:"head_commit_sha,omitempty"`
	BaseCommitSHA  string     `json:"base_commit_sha,omitempty"`
	StartCommitSHA string     `json:"start_commit_sha,omitempty"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
	MergeRequestID int        `json:"merge_request_id,omitempty"`
	State          string     `json:"state,omitempty"`
	RealSize       string     `json:"real_size,omitempty"`
	Commits        []*Commit  `json:"commits,omitempty"`
	Diffs          []*Diff    `json:"diffs,omitempty"`
}

func (m MergeRequestDiffVersion) String() string {
	return Stringify(m)
}

// ListMergeRequestsOptions represents the available ListMergeRequests()
// options.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#list-merge-requests
type ListMergeRequestsOptions struct {
	ListOptions
	State           *string    `url:"state,omitempty" json:"state,omitempty"`
	OrderBy         *string    `url:"order_by,omitempty" json:"order_by,omitempty"`
	Sort            *string    `url:"sort,omitempty" json:"sort,omitempty"`
	Milestone       *string    `url:"milestone,omitempty" json:"milestone,omitempty"`
	View            *string    `url:"view,omitempty" json:"view,omitempty"`
	Labels          Labels     `url:"labels,omitempty" json:"labels,omitempty"`
	CreatedAfter    *time.Time `url:"created_after,omitempty" json:"created_after,omitempty"`
	CreatedBefore   *time.Time `url:"created_before,omitempty" json:"created_before,omitempty"`
	Scope           *string    `url:"scope,omitempty" json:"scope,omitempty"`
	AuthorID        *int       `url:"author_id,omitempty" json:"author_id,omitempty"`
	AssigneeID      *int       `url:"assignee_id,omitempty" json:"assignee_id,omitempty"`
	MyReactionEmoji *string    `url:"my_reaction_emoji,omitempty" json:"my_reaction_emoji,omitempty"`
}

// ListMergeRequests gets all merge requests. The state parameter can be used
// to get only merge requests with a given state (opened, closed, or merged)
// or all of them (all). The pagination parameters page and per_page can be
// used to restrict the list of merge requests.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#list-merge-requests
func (s *MergeRequestsService) ListMergeRequests(opt *ListMergeRequestsOptions, options ...OptionFunc) ([]*MergeRequest, *Response, error) {
	req, err := s.client.NewRequest("GET", "merge_requests", opt, options)
	if err != nil {
		return nil, nil, err
	}

	var m []*MergeRequest
	resp, err := s.client.Do(req, &m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, err
}

// ListProjectMergeRequestsOptions represents the available ListMergeRequests()
// options.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#list-project-merge-requests
type ListProjectMergeRequestsOptions struct {
	ListOptions
	IIDs            []int      `url:"iids[],omitempty" json:"iids,omitempty"`
	State           *string    `url:"state,omitempty" json:"state,omitempty"`
	OrderBy         *string    `url:"order_by,omitempty" json:"order_by,omitempty"`
	Sort            *string    `url:"sort,omitempty" json:"sort,omitempty"`
	Milestone       *string    `url:"milestone,omitempty" json:"milestone,omitempty"`
	View            *string    `url:"view,omitempty" json:"view,omitempty"`
	Labels          Labels     `url:"labels,omitempty" json:"labels,omitempty"`
	CreatedAfter    *time.Time `url:"created_after,omitempty" json:"created_after,omitempty"`
	CreatedBefore   *time.Time `url:"created_before,omitempty" json:"created_before,omitempty"`
	Scope           *string    `url:"scope,omitempty" json:"scope,omitempty"`
	AuthorID        *int       `url:"author_id,omitempty" json:"author_id,omitempty"`
	AssigneeID      *int       `url:"assignee_id,omitempty" json:"assignee_id,omitempty"`
	MyReactionEmoji *string    `url:"my_reaction_emoji,omitempty" json:"my_reaction_emoji,omitempty"`
}

// ListProjectMergeRequests gets all merge requests for this project. The state
// parameter can be used to get only merge requests with a given state (opened,
// closed, or merged) or all of them (all). The pagination parameters page and
// per_page can be used to restrict the list of merge requests.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#list-merge-requests
func (s *MergeRequestsService) ListProjectMergeRequests(pid interface{}, opt *ListProjectMergeRequestsOptions, options ...OptionFunc) ([]*MergeRequest, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_requests", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	var m []*MergeRequest
	resp, err := s.client.Do(req, &m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, err
}

// GetMergeRequest shows information about a single merge request.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#get-single-mr
func (s *MergeRequestsService) GetMergeRequest(pid interface{}, mergeRequest int, options ...OptionFunc) (*MergeRequest, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_requests/%d", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	m := new(MergeRequest)
	resp, err := s.client.Do(req, m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, err
}

// GetMergeRequestApprovals gets information about a merge requests approvals
//
// GitLab API docs:
// https://docs.gitlab.com/ee/api/merge_requests.html#merge-request-approvals
func (s *MergeRequestsService) GetMergeRequestApprovals(pid interface{}, mergeRequest int, options ...OptionFunc) (*MergeRequestApprovals, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_requests/%d/approvals", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	a := new(MergeRequestApprovals)
	resp, err := s.client.Do(req, a)
	if err != nil {
		return nil, resp, err
	}

	return a, resp, err
}

// GetMergeRequestCommits gets a list of merge request commits.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#get-single-mr-commits
func (s *MergeRequestsService) GetMergeRequestCommits(pid interface{}, mergeRequest int, options ...OptionFunc) ([]*Commit, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_requests/%d/commits", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	var c []*Commit
	resp, err := s.client.Do(req, &c)
	if err != nil {
		return nil, resp, err
	}

	return c, resp, err
}

// GetMergeRequestChanges shows information about the merge request including
// its files and changes.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#get-single-mr-changes
func (s *MergeRequestsService) GetMergeRequestChanges(pid interface{}, mergeRequest int, options ...OptionFunc) (*MergeRequest, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_requests/%d/changes", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	m := new(MergeRequest)
	resp, err := s.client.Do(req, m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, err
}

// GetIssuesClosedOnMerge gets all the issues that would be closed by merging the
// provided merge request.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#list-issues-that-will-close-on-merge
func (s *MergeRequestsService) GetIssuesClosedOnMerge(pid interface{}, mergeRequest int, options ...OptionFunc) ([]*Issue, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("/projects/%s/merge_requests/%v/closes_issues", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	var i []*Issue
	resp, err := s.client.Do(req, &i)
	if err != nil {
		return nil, resp, err
	}

	return i, resp, err
}

// CreateMergeRequestOptions represents the available CreateMergeRequest()
// options.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#create-mr
type CreateMergeRequestOptions struct {
	Title           *string `url:"title,omitempty" json:"title,omitempty"`
	Description     *string `url:"description,omitempty" json:"description,omitempty"`
	SourceBranch    *string `url:"source_branch,omitempty" json:"source_branch,omitempty"`
	TargetBranch    *string `url:"target_branch,omitempty" json:"target_branch,omitempty"`
	AssigneeID      *int    `url:"assignee_id,omitempty" json:"assignee_id,omitempty"`
	TargetProjectID *int    `url:"target_project_id,omitempty" json:"target_project_id,omitempty"`
}

// CreateMergeRequest creates a new merge request.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#create-mr
func (s *MergeRequestsService) CreateMergeRequest(pid interface{}, opt *CreateMergeRequestOptions, options ...OptionFunc) (*MergeRequest, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_requests", url.QueryEscape(project))

	req, err := s.client.NewRequest("POST", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	m := new(MergeRequest)
	resp, err := s.client.Do(req, m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, err
}

// UpdateMergeRequestOptions represents the available UpdateMergeRequest()
// options.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#update-mr
type UpdateMergeRequestOptions struct {
	Title        *string `url:"title,omitempty" json:"title,omitempty"`
	Description  *string `url:"description,omitempty" json:"description,omitempty"`
	TargetBranch *string `url:"target_branch,omitempty" json:"target_branch,omitempty"`
	AssigneeID   *int    `url:"assignee_id,omitempty" json:"assignee_id,omitempty"`
	Labels       Labels  `url:"labels,comma,omitempty" json:"labels,omitempty"`
	MilestoneID  *int    `url:"milestone_id,omitempty" json:"milestone_id,omitempty"`
	StateEvent   *string `url:"state_event,omitempty" json:"state_event,omitempty"`
}

// UpdateMergeRequest updates an existing project milestone.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#update-mr
func (s *MergeRequestsService) UpdateMergeRequest(pid interface{}, mergeRequest int, opt *UpdateMergeRequestOptions, options ...OptionFunc) (*MergeRequest, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_requests/%d", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("PUT", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	m := new(MergeRequest)
	resp, err := s.client.Do(req, m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, err
}

// DeleteMergeRequest deletes a merge request.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#delete-a-merge-request
func (s *MergeRequestsService) DeleteMergeRequest(pid interface{}, mergeRequest int, options ...OptionFunc) (*Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_requests/%d", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("DELETE", u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// AcceptMergeRequestOptions represents the available AcceptMergeRequest()
// options.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#accept-mr
type AcceptMergeRequestOptions struct {
	MergeCommitMessage        *string `url:"merge_commit_message,omitempty" json:"merge_commit_message,omitempty"`
	ShouldRemoveSourceBranch  *bool   `url:"should_remove_source_branch,omitempty" json:"should_remove_source_branch,omitempty"`
	MergeWhenPipelineSucceeds *bool   `url:"merge_when_pipeline_succeeds,omitempty" json:"merge_when_pipeline_succeeds,omitempty"`
	Sha                       *string `url:"sha,omitempty" json:"sha,omitempty"`
}

// AcceptMergeRequest merges changes submitted with MR using this API. If merge
// success you get 200 OK. If it has some conflicts and can not be merged - you
// get 405 and error message 'Branch cannot be merged'. If merge request is
// already merged or closed - you get 405 and error message 'Method Not Allowed'
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#accept-mr
func (s *MergeRequestsService) AcceptMergeRequest(pid interface{}, mergeRequest int, opt *AcceptMergeRequestOptions, options ...OptionFunc) (*MergeRequest, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_requests/%d/merge", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("PUT", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	m := new(MergeRequest)
	resp, err := s.client.Do(req, m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, err
}

// CancelMergeWhenPipelineSucceeds cancels a merge when pipeline succeeds. If
// you don't have permissions to accept this merge request - you'll get a 401.
// If the merge request is already merged or closed - you get 405 and error
// message 'Method Not Allowed'. In case the merge request is not set to be
// merged when the pipeline succeeds, you'll also get a 406 error.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#cancel-merge-when-pipeline-succeeds
func (s *MergeRequestsService) CancelMergeWhenPipelineSucceeds(pid interface{}, mergeRequest int, options ...OptionFunc) (*MergeRequest, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_requests/%d/cancel_merge_when_pipeline_succeeds", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("PUT", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	m := new(MergeRequest)
	resp, err := s.client.Do(req, m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, err
}

// GetMergeRequestDiffVersions get a list of merge request diff versions.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#get-mr-diff-versions
func (s *MergeRequestsService) GetMergeRequestDiffVersions(pid interface{}, mergeRequest int, options ...OptionFunc) ([]*MergeRequestDiffVersion, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_requests/%d/versions", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	var v []*MergeRequestDiffVersion
	resp, err := s.client.Do(req, &v)
	if err != nil {
		return nil, resp, err
	}

	return v, resp, err
}

// GetSingleMergeRequestDiffVersion get a single MR diff version
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#get-a-single-mr-diff-version
func (s *MergeRequestsService) GetSingleMergeRequestDiffVersion(pid interface{}, mergeRequest, version int, options ...OptionFunc) (*MergeRequestDiffVersion, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_requests/%d/versions/%d", url.QueryEscape(project), mergeRequest, version)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	var v = new(MergeRequestDiffVersion)
	resp, err := s.client.Do(req, v)
	if err != nil {
		return nil, resp, err
	}

	return v, resp, err
}

// Subscribe subscribes the authenticated user to the given merge request
// to receive notifications. If the user is already subscribed to the
// merge request, the status code 304 is returned.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#subscribe-to-a-merge-request
func (s *MergeRequestsService) SubscribeToMergeRequest(pid interface{}, mergeRequest int, options ...OptionFunc) (*MergeRequest, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_requests/%d/subscribe", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	m := new(MergeRequest)
	resp, err := s.client.Do(req, m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, err
}

// Unsubscribe unsubscribes the authenticated user from the given merge request
// to not receive notifications from that merge request. If the user is
// not subscribed to the merge request, status code 304 is returned.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#unsubscribe-from-a-merge-request
func (s *MergeRequestsService) UnsubscribeFromMergeRequest(pid interface{}, mergeRequest int, options ...OptionFunc) (*MergeRequest, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_requests/%d/unsubscribe", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	m := new(MergeRequest)
	resp, err := s.client.Do(req, m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, err
}

// CreateTodo manually creates a todo for the current user on a merge request.
// If there already exists a todo for the user on that merge request,
// status code 304 is returned.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#create-a-todo
func (s *MergeRequestsService) CreateTodo(pid interface{}, mergeRequest int, options ...OptionFunc) (*Todo, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_requests/%d/todo", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	t := new(Todo)
	resp, err := s.client.Do(req, t)
	if err != nil {
		return nil, resp, err
	}

	return t, resp, err
}

// SetTimeEstimate sets the time estimate for a single project merge request.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#set-a-time-estimate-for-a-merge-request
func (s *MergeRequestsService) SetTimeEstimate(pid interface{}, mergeRequest int, opt *SetTimeEstimateOptions, options ...OptionFunc) (*TimeStats, *Response, error) {
	return s.timeStats.setTimeEstimate(pid, "merge_requests", mergeRequest, opt, options...)
}

// ResetTimeEstimate resets the time estimate for a single project merge request.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#reset-the-time-estimate-for-a-merge-request
func (s *MergeRequestsService) ResetTimeEstimate(pid interface{}, mergeRequest int, options ...OptionFunc) (*TimeStats, *Response, error) {
	return s.timeStats.resetTimeEstimate(pid, "merge_requests", mergeRequest, options...)
}

// AddSpentTime adds spent time for a single project merge request.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#add-spent-time-for-a-merge-request
func (s *MergeRequestsService) AddSpentTime(pid interface{}, mergeRequest int, opt *AddSpentTimeOptions, options ...OptionFunc) (*TimeStats, *Response, error) {
	return s.timeStats.addSpentTime(pid, "merge_requests", mergeRequest, opt, options...)
}

// ResetSpentTime resets the spent time for a single project merge request.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#reset-spent-time-for-a-merge-request
func (s *MergeRequestsService) ResetSpentTime(pid interface{}, mergeRequest int, options ...OptionFunc) (*TimeStats, *Response, error) {
	return s.timeStats.resetSpentTime(pid, "merge_requests", mergeRequest, options...)
}

// GetTimeSpent gets the spent time for a single project merge request.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/merge_requests.html#get-time-tracking-stats
func (s *MergeRequestsService) GetTimeSpent(pid interface{}, mergeRequest int, options ...OptionFunc) (*TimeStats, *Response, error) {
	return s.timeStats.getTimeSpent(pid, "merge_requests", mergeRequest, options...)
}
