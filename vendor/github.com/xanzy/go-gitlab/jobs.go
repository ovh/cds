//
// Copyright 2017, Arkbriar
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
	"io"
	"net/url"
	"time"
)

// ListJobsOptions are options for two list apis
type ListJobsOptions struct {
	ListOptions
	Scope []BuildState `url:"scope,omitempty" json:"scope,omitempty"`
}

// JobsService handles communication with the ci builds related methods
// of the GitLab API.
//
// GitLab API docs: https://docs.gitlab.com/ce/api/jobs.html
type JobsService struct {
	client *Client
}

// Job represents a ci build.
//
// GitLab API docs: https://docs.gitlab.com/ce/api/jobs.html
type Job struct {
	Commit        *Commit    `json:"commit"`
	CreatedAt     *time.Time `json:"created_at"`
	ArtifactsFile struct {
		Filename string `json:"filename"`
		Size     int    `json:"size"`
	} `json:"artifacts_file"`
	FinishedAt *time.Time `json:"finished_at"`
	ID         int        `json:"id"`
	Name       string     `json:"name"`
	Ref        string     `json:"ref"`
	Runner     struct {
		ID          int    `json:"id"`
		Description string `json:"description"`
		Active      bool   `json:"active"`
		IsShared    bool   `json:"is_shared"`
		Name        string `json:"name"`
	} `json:"runner"`
	Stage     string     `json:"stage"`
	StartedAt *time.Time `json:"started_at"`
	Status    string     `json:"status"`
	Tag       bool       `json:"tag"`
	User      *User      `json:"user"`
}

// ListProjectJobs gets a list of jobs in a project.
//
// The scope of jobs to show, one or array of: created, pending, running,
// failed, success, canceled, skipped; showing all jobs if none provided
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/jobs.html#list-project-jobs
func (s *JobsService) ListProjectJobs(pid interface{}, opts *ListJobsOptions, options ...OptionFunc) ([]Job, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/jobs", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, opts, options)
	if err != nil {
		return nil, nil, err
	}

	var jobs []Job
	resp, err := s.client.Do(req, &jobs)
	if err != nil {
		return nil, resp, err
	}

	return jobs, resp, err
}

// ListPipelineJobs gets a list of jobs for specific pipeline in a
// project. If the pipeline ID is not found, it will respond with 404.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/jobs.html#list-pipeline-jobs
func (s *JobsService) ListPipelineJobs(pid interface{}, pipelineID int, opts *ListJobsOptions, options ...OptionFunc) ([]Job, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/pipelines/%d/jobs", project, pipelineID)

	req, err := s.client.NewRequest("GET", u, opts, options)
	if err != nil {
		return nil, nil, err
	}

	var jobs []Job
	resp, err := s.client.Do(req, &jobs)
	if err != nil {
		return nil, resp, err
	}

	return jobs, resp, err
}

// GetJob gets a single job of a project.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/jobs.html#get-a-single-job
func (s *JobsService) GetJob(pid interface{}, jobID int, options ...OptionFunc) (*Job, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/jobs/%d", project, jobID)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	job := new(Job)
	resp, err := s.client.Do(req, job)
	if err != nil {
		return nil, resp, err
	}

	return job, resp, err
}

// GetJobArtifacts get jobs artifacts of a project
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/jobs.html#get-job-artifacts
func (s *JobsService) GetJobArtifacts(pid interface{}, jobID int, options ...OptionFunc) (io.Reader, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/jobs/%d/artifacts", project, jobID)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	artifactsBuf := new(bytes.Buffer)
	resp, err := s.client.Do(req, artifactsBuf)
	if err != nil {
		return nil, resp, err
	}

	return artifactsBuf, resp, err
}

// DownloadArtifactsFile download the artifacts file from the given
// reference name and job provided the job finished successfully.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/jobs.html#download-the-artifacts-file
func (s *JobsService) DownloadArtifactsFile(pid interface{}, refName string, job string, options ...OptionFunc) (io.Reader, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/jobs/artifacts/%s/download?job=%s", project, refName, job)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	artifactsBuf := new(bytes.Buffer)
	resp, err := s.client.Do(req, artifactsBuf)
	if err != nil {
		return nil, resp, err
	}

	return artifactsBuf, resp, err
}

// GetTraceFile gets a trace of a specific job of a project
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/jobs.html#get-a-trace-file
func (s *JobsService) GetTraceFile(pid interface{}, jobID int, options ...OptionFunc) (io.Reader, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/jobs/%d/trace", project, jobID)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	traceBuf := new(bytes.Buffer)
	resp, err := s.client.Do(req, traceBuf)
	if err != nil {
		return nil, resp, err
	}

	return traceBuf, resp, err
}

// CancelJob cancels a single job of a project.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/jobs.html#cancel-a-job
func (s *JobsService) CancelJob(pid interface{}, jobID int, options ...OptionFunc) (*Job, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/jobs/%d/cancel", project, jobID)

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	job := new(Job)
	resp, err := s.client.Do(req, job)
	if err != nil {
		return nil, resp, err
	}

	return job, resp, err
}

// RetryJob retries a single job of a project
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/jobs.html#retry-a-job
func (s *JobsService) RetryJob(pid interface{}, jobID int, options ...OptionFunc) (*Job, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/jobs/%d/retry", project, jobID)

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	job := new(Job)
	resp, err := s.client.Do(req, job)
	if err != nil {
		return nil, resp, err
	}

	return job, resp, err
}

// EraseJob erases a single job of a project, removes a job
// artifacts and a job trace.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/jobs.html#erase-a-job
func (s *JobsService) EraseJob(pid interface{}, jobID int, options ...OptionFunc) (*Job, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/jobs/%d/erase", project, jobID)

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	job := new(Job)
	resp, err := s.client.Do(req, job)
	if err != nil {
		return nil, resp, err
	}

	return job, resp, err
}

// KeepArtifacts prevents artifacts from being deleted when
// expiration is set.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/jobs.html#keep-artifacts
func (s *JobsService) KeepArtifacts(pid interface{}, jobID int, options ...OptionFunc) (*Job, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/jobs/%d/artifacts/keep", project, jobID)

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	job := new(Job)
	resp, err := s.client.Do(req, job)
	if err != nil {
		return nil, resp, err
	}

	return job, resp, err
}

// PlayJob triggers a nanual action to start a job.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/jobs.html#play-a-job
func (s *JobsService) PlayJob(pid interface{}, jobID int, options ...OptionFunc) (*Job, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/jobs/%d/play", project, jobID)

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	job := new(Job)
	resp, err := s.client.Do(req, job)
	if err != nil {
		return nil, resp, err
	}

	return job, resp, err
}
