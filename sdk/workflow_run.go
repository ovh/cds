package sdk

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ovh/venom"
)

//WorkflowRun is an execution instance of a run
type WorkflowRun struct {
	ID               int64                       `json:"id" db:"id"`
	Number           int64                       `json:"num" db:"num"`
	ProjectID        int64                       `json:"project_id,omitempty" db:"project_id"`
	WorkflowID       int64                       `json:"workflow_id" db:"workflow_id"`
	Workflow         Workflow                    `json:"workflow" db:"-"`
	Start            time.Time                   `json:"start" db:"start"`
	LastModified     time.Time                   `json:"last_modified" db:"last_modified"`
	WorkflowNodeRuns map[int64][]WorkflowNodeRun `json:"nodes" db:"-"`
	Infos            []WorkflowRunInfo           `json:"infos" db:"-"`
	Tags             []WorkflowRunTag            `json:"tags" db:"-"`
}

// Translate translates messages in WorkflowNodeRun
func (r *WorkflowRun) Translate(lang string) {
	for ki, info := range r.Infos {
		m := NewMessage(Messages[info.Message.ID], info.Message.Args...)
		r.Infos[ki].UserMessage = m.String(lang)
	}
}

// Tag push a new Tag in WorkflowRunTag
func (r *WorkflowRun) Tag(tag, value string) {
	var found bool
	for i := range r.Tags {
		if r.Tags[i].Tag == tag {
			found = true
			if r.Tags[i].Value != value {
				r.Tags[i].Value = strings.Join([]string{r.Tags[i].Value, value}, ",")
			}
		}
	}
	if !found {
		r.Tags = append(r.Tags, WorkflowRunTag{Tag: tag, Value: value})
	}
}

//WorkflowRunInfo is an info on workflow run
type WorkflowRunInfo struct {
	APITime time.Time `json:"api_time,omitempty" db:"-"`
	Message SpawnMsg  `json:"message,omitempty" db:"-"`
	// UserMessage contains msg translated for end user
	UserMessage string `json:"user_message,omitempty" db:"-"`
}

//WorkflowRunTag is a tag on workflow run
type WorkflowRunTag struct {
	WorkflowRunID int64  `json:"-" db:"workflow_run_id"`
	Tag           string `json:"tag" db:"tag"`
	Value         string `json:"value" db:"value"`
}

//WorkflowNodeRun is as execution instance of a node
type WorkflowNodeRun struct {
	WorkflowRunID      int64                     `json:"workflow_run_id" db:"workflow_run_id"`
	ID                 int64                     `json:"id" db:"id"`
	WorkflowNodeID     int64                     `json:"workflow_node_id" db:"workflow_node_id"`
	Number             int64                     `json:"num" db:"num"`
	SubNumber          int64                     `json:"subnumber" db:"sub_num"`
	Status             string                    `json:"status" db:"status"`
	Stages             []Stage                   `json:"stages" db:"-"`
	Start              time.Time                 `json:"start" db:"start"`
	LastModified       time.Time                 `json:"last_modified" db:"last_modified"`
	Done               time.Time                 `json:"done" db:"done"`
	HookEvent          *WorkflowNodeRunHookEvent `json:"hook_event" db:"-"`
	Manual             *WorkflowNodeRunManual    `json:"manual" db:"-"`
	SourceNodeRuns     []int64                   `json:"source_node_runs" db:"-"`
	Payload            interface{}               `json:"payload" db:"-"`
	PipelineParameters []Parameter               `json:"pipeline_parameters" db:"-"`
	BuildParameters    []Parameter               `json:"build_parameters" db:"-"`
	Artifacts          []WorkflowNodeRunArtifact `json:"artifacts,omitempty" db:"-"`
	Tests              *venom.Tests              `json:"tests,omitempty" db:"-"`
	Commits            []VCSCommit               `json:"commits,omitempty" db:"-"`
}

// Translate translates messages in WorkflowNodeRun
func (nr *WorkflowNodeRun) Translate(lang string) {
	for ks := range nr.Stages {
		for kj := range nr.Stages[ks].RunJobs {
			nr.Stages[ks].RunJobs[kj].Translate(lang)
		}
	}
}

//WorkflowNodeRunArtifact represents tests list
type WorkflowNodeRunArtifact struct {
	WorkflowID        int64     `json:"workflow_id" db:"workflow_run_id"`
	WorkflowNodeRunID int64     `json:"workflow_node_run_id" db:"workflow_node_run_id"`
	ID                int64     `json:"id" db:"id"`
	Name              string    `json:"name" db:"name"`
	Tag               string    `json:"tag" db:"tag"`
	DownloadHash      string    `json:"download_hash" db:"download_hash"`
	Size              int64     `json:"size,omitempty" db:"size"`
	Perm              uint32    `json:"perm,omitempty" db:"perm"`
	MD5sum            string    `json:"md5sum,omitempty" db:"md5sum"`
	ObjectPath        string    `json:"object_path,omitempty" db:"object_path"`
	Created           time.Time `json:"created,omitempty" db:"created"`
}

//WorkflowNodeJobRun represents an job to be run
type WorkflowNodeJobRun struct {
	ID                int64       `json:"id" db:"id"`
	WorkflowNodeRunID int64       `json:"workflow_node_run_id,omitempty" db:"workflow_node_run_id"`
	Job               ExecutedJob `json:"job" db:"-"`
	Parameters        []Parameter `json:"parameters,omitempty" db:"-"`
	Status            string      `json:"status"  db:"status"`
	Queued            time.Time   `json:"queued,omitempty" db:"queued"`
	QueuedSeconds     int64       `json:"queued_seconds,omitempty" db:"-"`
	Start             time.Time   `json:"start,omitempty" db:"start"`
	Done              time.Time   `json:"done,omitempty" db:"done"`
	Model             string      `json:"model,omitempty" db:"model"`
	BookedBy          Hatchery    `json:"bookedby" db:"-"`
	SpawnInfos        []SpawnInfo `json:"spawninfos" db:"-"`
}

// Translate translates messages in WorkflowNodeJobRun
func (njr *WorkflowNodeJobRun) Translate(lang string) {
	for ki, info := range njr.SpawnInfos {
		m := NewMessage(Messages[info.Message.ID], info.Message.Args...)
		njr.SpawnInfos[ki].UserMessage = m.String(lang)
	}

}

//WorkflowNodeRunHookEvent is an instanc of event received on a hook
type WorkflowNodeRunHookEvent struct {
	Payload            interface{} `json:"payload" db:"-"`
	PipelineParameters []Parameter `json:"pipeline_parameter" db:"-"`
	WorkflowNodeHookID int64       `json:"workflow_node_hook_id" db:"-"`
}

//WorkflowNodeRunManual is an instanc of event received on a hook
type WorkflowNodeRunManual struct {
	Payload            interface{} `json:"payload" db:"-"`
	PipelineParameters []Parameter `json:"pipeline_parameter" db:"-"`
	User               User        `json:"user" db:"-"`
}

//GetName returns the name the artifact
func (a *WorkflowNodeRunArtifact) GetName() string {
	return a.Name
}

//GetPath returns the path of the artifact
func (a *WorkflowNodeRunArtifact) GetPath() string {
	container := fmt.Sprintf("%d-%d-%s", a.WorkflowID, a.WorkflowNodeRunID, a.Tag)
	container = url.QueryEscape(container)
	container = strings.Replace(container, "/", "-", -1)
	return container
}
