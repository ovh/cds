package sdk

import (
	"bytes"
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"time"

	"github.com/ovh/venom"
)

//WorkflowRun is an execution instance of a run
type WorkflowRun struct {
	ID               int64                            `json:"id" db:"id"`
	Number           int64                            `json:"num" db:"num" cli:"num"`
	ProjectID        int64                            `json:"project_id,omitempty" db:"project_id"`
	WorkflowID       int64                            `json:"workflow_id" db:"workflow_id"`
	Status           string                           `json:"status" db:"status" cli:"status"`
	Workflow         Workflow                         `json:"workflow" db:"-"`
	Start            time.Time                        `json:"start" db:"start" cli:"start"`
	LastModified     time.Time                        `json:"last_modified" db:"last_modified"`
	WorkflowNodeRuns map[int64][]WorkflowNodeRun      `json:"nodes,omitempty" db:"-"`
	Infos            []WorkflowRunInfo                `json:"infos,omitempty" db:"-"`
	Tags             []WorkflowRunTag                 `json:"tags,omitempty" db:"-" cli:"tags"`
	LastSubNumber    int64                            `json:"last_subnumber" db:"last_sub_num"`
	LastExecution    time.Time                        `json:"last_execution" db:"last_execution" cli:"last_execution"`
	ToDelete         bool                             `json:"to_delete" db:"to_delete" cli:"-"`
	JoinTriggersRun  map[int64]WorkflowNodeTriggerRun `json:"join_triggers_run,omitempty" db:"-"`
}

// WorkflowNodeRunRelease represents the request struct use by release builtin action for workflow
type WorkflowNodeRunRelease struct {
	TagName        string   `json:"tag_name"`
	ReleaseTitle   string   `json:"release_title"`
	ReleaseContent string   `json:"release_content"`
	Artifacts      []string `json:"artifacts,omitempty"`
}

// WorkflowRunPostHandlerOption contains the body content for launch a workflow
type WorkflowRunPostHandlerOption struct {
	Hook        *WorkflowNodeRunHookEvent `json:"hook,omitempty"`
	Manual      *WorkflowNodeRunManual    `json:"manual,omitempty"`
	Number      *int64                    `json:"number,omitempty"`
	FromNodeIDs []int64                   `json:"from_nodes,omitempty"`
}

//WorkflowRunNumber contains a workflow run number
type WorkflowRunNumber struct {
	Num int64 `json:"num" cli:"run-number"`
}

// Translate translates messages in WorkflowNodeRun
func (r *WorkflowRun) Translate(lang string) {
	for ki, info := range r.Infos {
		m := NewMessage(Messages[info.Message.ID], info.Message.Args...)
		r.Infos[ki].UserMessage = m.String(lang)
	}
}

// Tag push a new Tag in WorkflowRunTag and return if a tag was added or no
func (r *WorkflowRun) Tag(tag, value string) bool {
	if value == "" {
		return false
	}
	var found bool
	for i := range r.Tags {
		if r.Tags[i].Tag == tag {
			found = true
			tagValues := strings.Split(r.Tags[i].Value, ",")
			var exist bool
			for _, tagVal := range tagValues {
				if tagVal == value {
					exist = true
					break
				}
			}
			if !exist {
				r.Tags[i].Value = strings.Join([]string{r.Tags[i].Value, value}, ",")
			}
		}
	}
	if !found {
		r.Tags = append(r.Tags, WorkflowRunTag{Tag: tag, Value: value})
		return true
	}

	return false
}

// TagExists return true if tag already exits
func (r *WorkflowRun) TagExists(tag string) bool {
	for i := range r.Tags {
		if r.Tags[i].Tag == tag {
			return true
		}
	}
	return false
}

//WorkflowRunInfo is an info on workflow run
type WorkflowRunInfo struct {
	APITime time.Time `json:"api_time,omitempty" db:"-"`
	Message SpawnMsg  `json:"message,omitempty" db:"-"`
	// UserMessage contains msg translated for end user
	UserMessage string `json:"user_message,omitempty" db:"-"`
	IsError     bool   `json:"is_error" db:"-"`
}

//WorkflowRunTag is a tag on workflow run
type WorkflowRunTag struct {
	WorkflowRunID int64  `json:"-" db:"workflow_run_id"`
	Tag           string `json:"tag,omitempty" db:"tag" cli:"tag"`
	Value         string `json:"value,omitempty" db:"value" cli:"value"`
}

//WorkflowNodeRun is as execution instance of a node. This type is duplicated for database persistence in the engine/api/workflow package
type WorkflowNodeRun struct {
	WorkflowRunID      int64                            `json:"workflow_run_id"`
	ID                 int64                            `json:"id"`
	WorkflowNodeID     int64                            `json:"workflow_node_id"`
	WorkflowNodeName   string                           `json:"workflow_node_name"`
	Number             int64                            `json:"num"`
	SubNumber          int64                            `json:"subnumber"`
	Status             string                           `json:"status"`
	Stages             []Stage                          `json:"stages,omitempty"`
	Start              time.Time                        `json:"start"`
	LastModified       time.Time                        `json:"last_modified"`
	Done               time.Time                        `json:"done"`
	HookEvent          *WorkflowNodeRunHookEvent        `json:"hook_event,omitempty"`
	Manual             *WorkflowNodeRunManual           `json:"manual,omitempty"`
	SourceNodeRuns     []int64                          `json:"source_node_runs,omitempty"`
	Payload            interface{}                      `json:"payload,omitempty"`
	PipelineParameters []Parameter                      `json:"pipeline_parameters,omitempty"`
	BuildParameters    []Parameter                      `json:"build_parameters,omitempty"`
	Artifacts          []WorkflowNodeRunArtifact        `json:"artifacts,omitempty"`
	Tests              *venom.Tests                     `json:"tests,omitempty"`
	Commits            []VCSCommit                      `json:"commits,omitempty"`
	TriggersRun        map[int64]WorkflowNodeTriggerRun `json:"triggers_run,omitempty"`
	VCSRepository      string                           `json:"vcs_repository"`
	VCSBranch          string                           `json:"vcs_branch"`
	VCSHash            string                           `json:"vcs_hash"`
	CanBeRun           bool                             `json:"can_be_run"`
}

// WorkflowNodeTriggerRun Represent the state of a trigger
type WorkflowNodeTriggerRun struct {
	WorkflowDestNodeID int64  `json:"workflow_dest_node_id" db:"-"`
	Status             string `json:"status" db:"-"`
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
	Name              string    `json:"name" db:"name" cli:"name"`
	Tag               string    `json:"tag" db:"tag" cli:"tag"`
	Ref               string    `json:"ref" db:"ref" cli:"ref"`
	DownloadHash      string    `json:"download_hash" db:"download_hash"`
	Size              int64     `json:"size,omitempty" db:"size"`
	Perm              uint32    `json:"perm,omitempty" db:"perm"`
	MD5sum            string    `json:"md5sum,omitempty" db:"md5sum" cli:"-"`
	SHA512sum         string    `json:"sha512sum,omitempty" db:"sha512sum" cli:"sha512sum"`
	ObjectPath        string    `json:"object_path,omitempty" db:"object_path"`
	Created           time.Time `json:"created,omitempty" db:"created"`
	TempURL           string    `json:"temp_url,omitempty" db:"-"`
	TempURLSecretKey  string    `json:"-" db:"-"`
}

// Equal returns true if w WorkflowNodeRunArtifact equals c
func (w WorkflowNodeRunArtifact) Equal(c WorkflowNodeRunArtifact) bool {
	if w.SHA512sum != "" {
		return w.WorkflowID == c.WorkflowID &&
			w.WorkflowNodeRunID == c.WorkflowNodeRunID &&
			w.DownloadHash == c.DownloadHash &&
			w.Tag == c.Tag &&
			w.TempURL == c.TempURL &&
			w.SHA512sum == c.SHA512sum
	}
	return w.WorkflowID == c.WorkflowID &&
		w.WorkflowNodeRunID == c.WorkflowNodeRunID &&
		w.DownloadHash == c.DownloadHash &&
		w.Tag == c.Tag &&
		w.TempURL == c.TempURL &&
		w.MD5sum == c.MD5sum
}

//WorkflowNodeJobRun represents an job to be run
//easyjson:json
type WorkflowNodeJobRun struct {
	ID                     int64              `json:"id" db:"id"`
	WorkflowNodeRunID      int64              `json:"workflow_node_run_id,omitempty" db:"workflow_node_run_id"`
	Job                    ExecutedJob        `json:"job" db:"-"`
	Parameters             []Parameter        `json:"parameters,omitempty" db:"-"`
	Status                 string             `json:"status"  db:"status"`
	Retry                  int                `json:"retry"  db:"retry"`
	SpawnAttempts          []int64            `json:"spawn_attempts,omitempty"  db:"-"`
	Queued                 time.Time          `json:"queued,omitempty" db:"queued"`
	QueuedSeconds          int64              `json:"queued_seconds,omitempty" db:"-"`
	Start                  time.Time          `json:"start,omitempty" db:"start"`
	Done                   time.Time          `json:"done,omitempty" db:"done"`
	Model                  string             `json:"model,omitempty" db:"model"`
	BookedBy               Hatchery           `json:"bookedby" db:"-"`
	SpawnInfos             []SpawnInfo        `json:"spawninfos" db:"-"`
	ExecGroups             []Group            `json:"exec_groups" db:"-"`
	PlatformPluginBinaries []GRPCPluginBinary `json:"platform_plugin_binaries" db:"-"`
}

// WorkflowNodeJobRunSummary is a light representation of WorkflowNodeJobRun for CDS event
type WorkflowNodeJobRunSummary struct {
	ID                int64              `json:"id" db:"id"`
	WorkflowNodeRunID int64              `json:"workflow_node_run_id,omitempty" db:"workflow_node_run_id"`
	Status            string             `json:"status"  db:"status"`
	Queued            int64              `json:"queued,omitempty" db:"queued"`
	Start             int64              `json:"start,omitempty" db:"start"`
	Done              int64              `json:"done,omitempty" db:"done"`
	Job               ExecutedJobSummary `json:"job_summary,omitempty"`
	SpawnInfos        []SpawnInfo        `json:"spawninfos" db:"-"`
}

// ToSummary transforms a WorkflowNodeJobRun into a WorkflowNodeJobRunSummary
func (wnjr WorkflowNodeJobRun) ToSummary() WorkflowNodeJobRunSummary {
	sum := WorkflowNodeJobRunSummary{
		Done:              wnjr.Done.Unix(),
		WorkflowNodeRunID: wnjr.WorkflowNodeRunID,
		Status:            wnjr.Status,
		ID:                wnjr.ID,
		Queued:            wnjr.Queued.Unix(),
		Start:             wnjr.Start.Unix(),
		Job:               wnjr.Job.ToSummary(),
		SpawnInfos:        wnjr.SpawnInfos,
	}
	return sum
}

//WorkflowNodeJobRunInfo represents info on a job
type WorkflowNodeJobRunInfo struct {
	ID                   int64       `json:"id"`
	WorkflowNodeJobRunID int64       `json:"workflow_node_job_run_id,omitempty"`
	WorkflowNodeRunID    int64       `json:"workflow_node_run_id,omitempty"`
	SpawnInfos           []SpawnInfo `json:"info"`
	Created              time.Time   `json:"created"`
}

// Translate translates messages in WorkflowNodeJobRun
func (wnjr *WorkflowNodeJobRun) Translate(lang string) {
	for ki, info := range wnjr.SpawnInfos {
		m := NewMessage(Messages[info.Message.ID], info.Message.Args...)
		wnjr.SpawnInfos[ki].UserMessage = m.String(lang)
	}
}

//WorkflowNodeRunHookEvent is an instanc of event received on a hook
type WorkflowNodeRunHookEvent struct {
	Payload              map[string]string `json:"payload" db:"-"`
	WorkflowNodeHookUUID string            `json:"uuid" db:"-"`
}

//WorkflowNodeRunManual is an instanc of event received on a hook
type WorkflowNodeRunManual struct {
	Payload            interface{} `json:"payload" db:"-"`
	PipelineParameters []Parameter `json:"pipeline_parameter" db:"-"`
	User               User        `json:"user" db:"-"`
}

//GetName returns the name the artifact
func (w *WorkflowNodeRunArtifact) GetName() string {
	return w.Name
}

//GetPath returns the path of the artifact
func (w *WorkflowNodeRunArtifact) GetPath() string {
	ref := w.Ref
	if ref == "" {
		ref = w.Tag
	}
	container := fmt.Sprintf("%d-%d-%s", w.WorkflowID, w.WorkflowNodeRunID, ref)
	container = url.QueryEscape(container)
	container = strings.Replace(container, "/", "-", -1)
	return container
}

const workflowNodeRunReport = `CDS Report {{.WorkflowNodeName}}#{{.Number}}.{{.SubNumber}}
{{range $s := .Stages}}
* {{$s.Name}}
{{- range $j := $s.RunJobs}}
  * {{$j.Job.Action.Name}} {{ if eq $j.Status "Success" -}} ✔ {{ else }}{{ if eq $j.Status "Fail" -}} ✘ {{ else }}- {{ end }} {{ end }}
{{- end}}
{{end}}

{{- if gt .Tests.TotalKO 0}}
Unit Tests Report

{{- range $ts := .Tests.TestSuites}}
* {{ $ts.Name }}
{{range $tc := $ts.TestCases}}
  {{- if or ($tc.Errors) ($tc.Failures) }}  * {{ $tc.Name }} ✘ {{- end}}
{{end}}
{{- end}}
{{- end}}
`

func (w WorkflowNodeRun) Report() (string, error) {
	t := template.New("")
	t, err := t.Parse(workflowNodeRunReport)
	if err != nil {
		return "", err
	}
	out := new(bytes.Buffer)
	errE := t.Execute(out, w)
	return out.String(), errE
}
