package sdk

import (
	"bytes"
	"fmt"
	"html/template"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/ovh/venom"
	"github.com/sguiheux/go-coverage"
)

var (
	WorkflowRunHeader = "X-Workflow-Run"
	WorkflowHeader    = "X-Workflow"
	ProjectKeyHeader  = "X-Project-Key"
)

type WorkflowRunHeaders map[string]string

func (h *WorkflowRunHeaders) Set(k, v string) {
	if h == nil {
		h = new(WorkflowRunHeaders)
	}
	(*h)[k] = v
}

func (h *WorkflowRunHeaders) Get(k string) (string, bool) {
	if h == nil {
		return "", false
	}
	v, has := (*h)[k]
	return v, has
}

//WorkflowRun is an execution instance of a run
type WorkflowRun struct {
	ID               int64                            `json:"id" db:"id"`
	Number           int64                            `json:"num" db:"num" cli:"num,key"`
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
	Header           WorkflowRunHeaders               `json:"header,omitempty" db:"-"`
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

func (r *WorkflowRun) PendingOutgoingHook() map[string]*WorkflowNodeRun {
	nrs := make(map[string]*WorkflowNodeRun)
	for i := range r.WorkflowNodeRuns {
		runs := r.WorkflowNodeRuns[i]
		if len(runs) > 0 && runs[0].OutgoingHook == nil {
			continue
		}
		for j := range runs {
			nr := &runs[j]
			if nr.Status != StatusWaiting.String() && nr.Status != StatusBuilding.String() {
				continue
			}
			nrs[nr.UUID] = nr
		}
	}
	return nrs
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

// TODO remove old struct
func (r *WorkflowRun) RootRun() *WorkflowNodeRun {
	rootNodeRuns, has := r.WorkflowNodeRuns[r.Workflow.WorkflowData.Node.ID]
	if !has || len(rootNodeRuns) < 1 {
		return nil
	}
	rootRun := rootNodeRuns[0]
	return &rootRun
}

func (r *WorkflowRun) HasParentWorkflow() bool {
	rr := r.RootRun()
	if rr == nil {
		return false
	}

	if rr.HookEvent == nil {
		return false
	}

	return rr.HookEvent.ParentWorkflow.Key != "" &&
		rr.HookEvent.ParentWorkflow.Name != "" &&
		rr.HookEvent.ParentWorkflow.Run != 0 &&
		rr.HookEvent.ParentWorkflow.HookRunID != ""
}

func (r *WorkflowRun) GetOutgoingHookRun(uuid string) *WorkflowNodeRun {
	for i := range r.WorkflowNodeRuns {
		nodeRuns := r.WorkflowNodeRuns[i]
		if len(nodeRuns) == 0 || nodeRuns[0].OutgoingHook == nil {
			continue
		}
		for j := range nodeRuns {
			nr := &nodeRuns[j]
			if nr.UUID == uuid {
				return nr
			}
		}
	}
	return nil
}

//WorkflowRunInfo is an info on workflow run
type WorkflowRunInfo struct {
	APITime time.Time `json:"api_time,omitempty" db:"-"`
	Message SpawnMsg  `json:"message,omitempty" db:"-"`
	// UserMessage contains msg translated for end user
	UserMessage string `json:"user_message,omitempty" db:"-"`
	SubNumber   int64  `json:"sub_number,omitempty" db:"-"`
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
	WorkflowRunID          int64                                `json:"workflow_run_id"`
	WorkflowID             int64                                `json:"workflow_id"`
	ApplicationID          int64                                `json:"application_id"`
	ID                     int64                                `json:"id"`
	WorkflowNodeID         int64                                `json:"workflow_node_id"`
	WorkflowNodeName       string                               `json:"workflow_node_name"`
	Number                 int64                                `json:"num"`
	SubNumber              int64                                `json:"subnumber"`
	Status                 string                               `json:"status"`
	Stages                 []Stage                              `json:"stages,omitempty"`
	Start                  time.Time                            `json:"start"`
	LastModified           time.Time                            `json:"last_modified"`
	Done                   time.Time                            `json:"done"`
	HookEvent              *WorkflowNodeRunHookEvent            `json:"hook_event,omitempty"`
	Manual                 *WorkflowNodeRunManual               `json:"manual,omitempty"`
	SourceNodeRuns         []int64                              `json:"source_node_runs,omitempty"`
	Payload                interface{}                          `json:"payload,omitempty"`
	PipelineParameters     []Parameter                          `json:"pipeline_parameters,omitempty"`
	BuildParameters        []Parameter                          `json:"build_parameters,omitempty"`
	Artifacts              []WorkflowNodeRunArtifact            `json:"artifacts,omitempty"`
	StaticFiles            []StaticFiles                        `json:"static_files,omitempty"`
	Coverage               WorkflowNodeRunCoverage              `json:"coverage,omitempty"`
	VulnerabilitiesReport  WorkflowNodeRunVulnerabilityReport   `json:"vulnerabilities_report,omitempty"`
	Tests                  *venom.Tests                         `json:"tests,omitempty"`
	Commits                []VCSCommit                          `json:"commits,omitempty"`
	TriggersRun            map[int64]WorkflowNodeTriggerRun     `json:"triggers_run,omitempty"`
	VCSRepository          string                               `json:"vcs_repository"`
	VCSTag                 string                               `json:"vcs_tag"`
	VCSBranch              string                               `json:"vcs_branch"`
	VCSHash                string                               `json:"vcs_hash"`
	VCSServer              string                               `json:"vcs_server"`
	CanBeRun               bool                                 `json:"can_be_run"`
	Header                 WorkflowRunHeaders                   `json:"header,omitempty"`
	UUID                   string                               `json:"uuid,omitempty"`
	OutgoingHook           *NodeOutGoingHook                    `json:"outgoinghook,omitempty"`
	HookExecutionTimeStamp int64                                `json:"hook_execution_timestamp,omitempty"`
	HookExecutionID        string                               `json:"execution_id,omitempty"`
	Callback               *WorkflowNodeOutgoingHookRunCallback `json:"callback,omitempty"`
}

// WorkflowNodeOutgoingHookRunCallback is the callback coming from hooks uservice avec an outgoing hook execution
type WorkflowNodeOutgoingHookRunCallback struct {
	NodeHookID        int64     `json:"workflow_node_outgoing_hook_id"`
	Start             time.Time `json:"start"`
	Done              time.Time `json:"done"`
	Status            string    `json:"status"`
	Log               string    `json:"log"`
	WorkflowRunNumber *int64    `json:"workflow_run_number"`
}

// WorkflowNodeRunVulnerabilityReport represents vulnerabilities report for the current node run
type WorkflowNodeRunVulnerabilityReport struct {
	ID                int64                        `json:"id" db:"id"`
	ApplicationID     int64                        `json:"application_id" db:"application_id"`
	WorkflowID        int64                        `json:"workflow_id" db:"workflow_id"`
	WorkflowRunID     int64                        `json:"workflow_run_id" db:"workflow_run_id"`
	WorkflowNodeRunID int64                        `json:"workflow_node_run_id" db:"workflow_node_run_id"`
	Num               int64                        `json:"num" db:"workflow_number"`
	Branch            string                       `json:"branch" db:"branch"`
	Report            WorkflowNodeRunVulnerability `json:"report" db:"-"`
}

// WorkflowNodeRunVulnerability content of the workflow node run vulnerability report
type WorkflowNodeRunVulnerability struct {
	Vulnerabilities      []Vulnerability  `json:"vulnerabilities"`
	Summary              map[string]int64 `json:"summary"`
	DefaultBranchSummary map[string]int64 `json:"default_branch_summary"`
	PreviousRunSummary   map[string]int64 `json:"previous_run_summary"`
}

// WorkflowNodeRunCoverage represents the code coverage report
type WorkflowNodeRunCoverage struct {
	WorkflowID        int64                         `json:"workflow_id" db:"workflow_id"`
	WorkflowNodeRunID int64                         `json:"workflow_node_run_id" db:"workflow_node_run_id"`
	WorkflowRunID     int64                         `json:"workflow_run_id" db:"workflow_run_id"`
	ApplicationID     int64                         `json:"application_id" db:"application_id"`
	Num               int64                         `json:"run_number" db:"run_number"`
	Repository        string                        `json:"repository" db:"repository"`
	Branch            string                        `json:"branch" db:"branch"`
	Report            coverage.Report               `json:"report" db:"-"`
	Trend             WorkflowNodeRunCoverageTrends `json:"trend" db:"-"`
}

// WorkflowNodeRunCoverageTrends represents code coverage trend with current branch and default branch
type WorkflowNodeRunCoverageTrends struct {
	CurrentBranch coverage.Report `json:"current_branch_report"`
	DefaultBranch coverage.Report `json:"default_branch_report"`
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
	Name              string    `json:"name" db:"name" cli:"name,key"`
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
// /!\ DONT FORGET TO REGENERATE EASYJSON FILES /!\
//easyjson:json
type WorkflowNodeJobRun struct {
	ProjectID                 int64              `json:"project_id"`
	ID                        int64              `json:"id"`
	WorkflowNodeRunID         int64              `json:"workflow_node_run_id,omitempty"`
	Job                       ExecutedJob        `json:"job"`
	Parameters                []Parameter        `json:"parameters,omitempty"`
	Status                    string             `json:"status"`
	Retry                     int                `json:"retry"`
	SpawnAttempts             []int64            `json:"spawn_attempts,omitempty"`
	Queued                    time.Time          `json:"queued,omitempty"`
	QueuedSeconds             int64              `json:"queued_seconds,omitempty"`
	Start                     time.Time          `json:"start,omitempty"`
	Done                      time.Time          `json:"done,omitempty"`
	Model                     string             `json:"model,omitempty"`
	ModelType                 string             `json:"model_type,omitempty"`
	BookedBy                  Service            `json:"bookedby"`
	SpawnInfos                []SpawnInfo        `json:"spawninfos"`
	ExecGroups                []Group            `json:"exec_groups"`
	IntegrationPluginBinaries []GRPCPluginBinary `json:"integration_plugin_binaries,omitempty"`
	Header                    WorkflowRunHeaders `json:"header,omitempty"`
	ContainsService           bool               `json:"contains_service,omitempty"`
}

// /!\ DONT FORGET TO REGENERATE EASYJSON FILES /!\

// WorkflowNodeJobRunSummary is a light representation of WorkflowNodeJobRun for CDS event
type WorkflowNodeJobRunSummary struct {
	ID                int64              `json:"id"`
	WorkflowNodeRunID int64              `json:"workflow_node_run_id,omitempty"`
	Status            string             `json:"status"`
	Queued            int64              `json:"queued,omitempty"`
	Start             int64              `json:"start,omitempty"`
	Done              int64              `json:"done,omitempty"`
	Job               ExecutedJobSummary `json:"job_summary,omitempty"`
	SpawnInfos        []SpawnInfo        `json:"spawninfos"`
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
	ParentWorkflow       struct {
		Key       string `json:"key" db:"-"`
		Name      string `json:"name" db:"-"`
		Run       int64  `json:"run" db:"-"`
		HookRunID string `hook_run_id:"uuid" db:"-"`
	} `json:"parent_workflow" db:"-"`
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

const workflowNodeRunReport = `{{- if .Stages }}
CDS Report {{.WorkflowNodeName}}#{{.Number}}.{{.SubNumber}} {{ if eq .Status "Success" -}} ✔ {{ else }}{{ if eq .Status "Fail" -}} ✘ {{ else }}{{ if eq .Status "Stopped" -}} ■ {{ else }}- {{ end }} {{ end }} {{ end }}
{{- range $s := .Stages}}
{{- if $s.RunJobs }}
* {{$s.Name}}
{{- range $j := $s.RunJobs}}
  * {{$j.Job.Action.Name}} {{ if eq $j.Status "Success" -}} ✔ {{ else }}{{ if eq $j.Status "Fail" -}} ✘ {{ else }}{{ if eq $j.Status "Stopped" -}} ■ {{ else }}- {{ end }} {{ end }} {{ end }}
{{- end}}
{{- end}}
{{- end}}
{{- end}}

{{- if .Tests }}
{{- if gt .Tests.TotalKO 0}}
Unit Tests Report

{{- range $ts := .Tests.TestSuites}}
* {{ $ts.Name }}
{{range $tc := $ts.TestCases}}
  {{- if or ($tc.Errors) ($tc.Failures) }}  * {{ $tc.Name }} ✘ {{- end}}
{{end}}
{{- end}}
{{- end}}
{{- end}}
`

func (nr WorkflowNodeRun) Report() (string, error) {
	t := template.New("")
	t, err := t.Parse(workflowNodeRunReport)
	if err != nil {
		return "", err
	}
	out := new(bytes.Buffer)
	errE := t.Execute(out, nr)
	return out.String(), errE
}

type WorkflowQueue []WorkflowNodeJobRun

func (q WorkflowQueue) Sort() {
	//Count the number of WorkflowNodeJobRun per project_id
	n := make(map[int64]int, len(q))
	for _, j := range q {
		nb := n[j.ProjectID]
		nb++
		n[j.ProjectID] = nb
	}

	sort.Slice(q, func(i, j int) bool {
		p1 := n[q[i].ProjectID]
		p2 := n[q[j].ProjectID]
		return p1 < p2
	})

}
