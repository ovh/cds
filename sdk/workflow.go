package sdk

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/runabove/venom"
)

//Workflow represents a pipeline based workflow
type Workflow struct {
	ID           int64              `json:"id" db:"id"`
	Name         string             `json:"name" db:"name"`
	Description  string             `json:"description,omitempty" db:"description"`
	LastModified time.Time          `json:"last_modified" db:"last_modified"`
	ProjectID    int64              `json:"project_id,omitempty" db:"project_id"`
	ProjectKey   string             `json:"project_key" db:"-"`
	RootID       int64              `json:"root_id,omitempty" db:"root_node_id"`
	Root         *WorkflowNode      `json:"root" db:"-"`
	Joins        []WorkflowNodeJoin `json:"joins,omitempty" db:"-"`
}

//JoinsID returns joins ID
func (w *Workflow) JoinsID() []int64 {
	res := []int64{}
	for _, j := range w.Joins {
		res = append(res, j.ID)
	}
	return res
}

//Nodes returns nodes IDs excluding the root ID
func (w *Workflow) Nodes() []int64 {
	if w.Root == nil {
		return nil
	}

	res := []int64{}
	for _, t := range w.Root.Triggers {
		res = append(res, t.WorkflowDestNode.Nodes()...)
	}

	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			res = append(res, t.WorkflowDestNode.Nodes()...)
		}
	}
	return res
}

//GetNode returns the node given its id
func (w *Workflow) GetNode(id int64) *WorkflowNode {
	n := w.Root.GetNode(id)
	if n != nil {
		return n
	}
	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			n = t.WorkflowDestNode.GetNode(id)
			if n != nil {
				return n
			}
		}
	}
	return nil
}

//TriggersID returns triggers IDs
func (w *Workflow) TriggersID() []int64 {
	res := w.Root.TriggersID()
	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			res = append(res, t.ID)
			res = append(res, t.WorkflowDestNode.TriggersID()...)
		}
	}
	return res
}

//References returns a slice with all node references
func (w *Workflow) References() []string {
	if w.Root == nil {
		return nil
	}

	res := w.Root.References()
	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			res = append(res, t.WorkflowDestNode.References()...)
		}
	}
	return res
}

//InvolvedApplications returns all applications used in the workflow
func (w *Workflow) InvolvedApplications() []int64 {
	if w.Root == nil {
		return nil
	}

	res := w.Root.InvolvedApplications()
	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			res = append(res, t.WorkflowDestNode.InvolvedApplications()...)
		}
	}
	return res
}

//InvolvedPipelines returns all pipelines used in the workflow
func (w *Workflow) InvolvedPipelines() []int64 {
	if w.Root == nil {
		return nil
	}

	res := w.Root.InvolvedPipelines()
	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			res = append(res, t.WorkflowDestNode.InvolvedPipelines()...)
		}
	}
	return res
}

//GetPipelines returns all pipelines used in the workflow
func (w *Workflow) GetPipelines() []Pipeline {
	if w.Root == nil {
		return nil
	}

	res := w.Root.GetPipelines()
	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			res = append(res, t.WorkflowDestNode.GetPipelines()...)
		}
	}
	return res
}

//InvolvedEnvironments returns all environments used in the workflow
func (w *Workflow) InvolvedEnvironments() []int64 {
	if w.Root == nil {
		return nil
	}

	res := w.Root.InvolvedEnvironments()
	for _, j := range w.Joins {
		for _, t := range j.Triggers {
			res = append(res, t.WorkflowDestNode.InvolvedEnvironments()...)
		}
	}
	return res
}

//WorkflowNodeJoin aims to joins multiple node into multiple triggers
type WorkflowNodeJoin struct {
	ID             int64                     `json:"id" db:"id"`
	WorkflowID     int64                     `json:"workflow_id" db:"workflow_id"`
	SourceNodeIDs  []int64                   `json:"source_node_id,omitempty" db:"-"`
	SourceNodeRefs []string                  `json:"source_node_ref,omitempty" db:"-"`
	Triggers       []WorkflowNodeJoinTrigger `json:"triggers,omitempty" db:"-"`
}

//WorkflowNodeJoinTrigger is a trigger for joins
type WorkflowNodeJoinTrigger struct {
	ID                 int64                      `json:"id" db:"id"`
	WorkflowNodeJoinID int64                      `json:"join_id" db:"workflow_node_join_id"`
	WorkflowDestNodeID int64                      `json:"workflow_dest_node_id" db:"workflow_dest_node_id"`
	WorkflowDestNode   WorkflowNode               `json:"workflow_dest_node" db:"-"`
	Conditions         []WorkflowTriggerCondition `json:"conditions,omitempty" db:"-"`
}

//WorkflowNode represents a node in w workflow tree
type WorkflowNode struct {
	ID               int64                 `json:"id" db:"id"`
	Ref              string                `json:"ref,omitempty" db:"-"`
	WorkflowID       int64                 `json:"workflow_id" db:"workflow_id"`
	PipelineID       int64                 `json:"pipeline_id" db:"pipeline_id"`
	Pipeline         Pipeline              `json:"pipeline" db:"-"`
	Context          *WorkflowNodeContext  `json:"context" db:"-"`
	TriggerSrcID     int64                 `json:"-" db:"-"`
	TriggerJoinSrcID int64                 `json:"-" db:"-"`
	Hooks            []WorkflowNodeHook    `json:"hooks,omitempty" db:"-"`
	Triggers         []WorkflowNodeTrigger `json:"triggers,omitempty" db:"-"`
}

//GetNode returns the node given its id
func (n *WorkflowNode) GetNode(id int64) *WorkflowNode {
	if n == nil {
		return nil
	}
	if n.ID == id {
		return n
	}
	for _, t := range n.Triggers {
		n = t.WorkflowDestNode.GetNode(id)
		if n != nil {
			return n
		}
	}
	return nil
}

//Nodes returns a slice with all node IDs
func (n *WorkflowNode) Nodes() []int64 {
	res := []int64{n.ID}
	for _, t := range n.Triggers {
		res = append(res, t.WorkflowDestNode.Nodes()...)
	}
	return res
}

//TriggersID returns a slides of triggers IDs
func (n *WorkflowNode) TriggersID() []int64 {
	res := []int64{}
	for _, t := range n.Triggers {
		res = append(res, t.ID)
		res = append(res, t.WorkflowDestNode.TriggersID()...)
	}
	return res
}

//References returns a slice with all node references
func (n *WorkflowNode) References() []string {
	res := []string{}
	if n.Ref != "" {
		res = []string{n.Ref}
	}
	for _, t := range n.Triggers {
		res = append(res, t.WorkflowDestNode.References()...)
	}
	return res
}

//InvolvedApplications returns all applications used in the workflow
func (n *WorkflowNode) InvolvedApplications() []int64 {
	res := []int64{}
	if n.Context != nil {
		if n.Context.ApplicationID == 0 && n.Context.Application != nil {
			n.Context.ApplicationID = n.Context.Application.ID
		}
		if n.Context.ApplicationID != 0 {
			res = []int64{n.Context.ApplicationID}
		}
	}
	for _, t := range n.Triggers {
		res = append(res, t.WorkflowDestNode.InvolvedApplications()...)
	}
	return res
}

//InvolvedPipelines returns all pipelines used in the workflow
func (n *WorkflowNode) InvolvedPipelines() []int64 {
	res := []int64{}
	if n.Context != nil {
		if n.PipelineID == 0 {
			n.PipelineID = n.Pipeline.ID
		}
		res = []int64{n.PipelineID}
	}
	for _, t := range n.Triggers {
		res = append(res, t.WorkflowDestNode.InvolvedPipelines()...)
	}
	return res
}

//GetPipelines returns all pipelines used in the workflow
func (n *WorkflowNode) GetPipelines() []Pipeline {
	res := []Pipeline{n.Pipeline}
	for _, t := range n.Triggers {
		res = append(res, t.WorkflowDestNode.GetPipelines()...)
	}
	return res
}

//InvolvedEnvironments returns all environments used in the workflow
func (n *WorkflowNode) InvolvedEnvironments() []int64 {
	res := []int64{}
	if n.Context != nil {
		if n.Context.EnvironmentID == 0 && n.Context.Environment != nil {
			n.Context.EnvironmentID = n.Context.Environment.ID
		}
		if n.Context.EnvironmentID != 0 {
			res = []int64{n.Context.EnvironmentID}
		}
	}
	for _, t := range n.Triggers {
		res = append(res, t.WorkflowDestNode.InvolvedEnvironments()...)
	}
	return res
}

//WorkflowNodeTrigger is a ling betweeb two pipelines in a workflow
type WorkflowNodeTrigger struct {
	ID                 int64                      `json:"id" db:"id"`
	WorkflowNodeID     int64                      `json:"workflow_node_id" db:"workflow_node_id"`
	WorkflowDestNodeID int64                      `json:"workflow_dest_node_id" db:"workflow_dest_node_id"`
	WorkflowDestNode   WorkflowNode               `json:"workflow_dest_node" db:"-"`
	Conditions         []WorkflowTriggerCondition `json:"conditions,omitempty" db:"-"`
}

//WorkflowTriggerCondition represents a condition to trigger ot not a pipeline in a workflow. Operator can be =, !=, regex
type WorkflowTriggerCondition struct {
	Variable string `json:"variable"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

//WorkflowNodeContext represents a context attached on a node
type WorkflowNodeContext struct {
	ID                        int64        `json:"id" db:"id"`
	WorkflowNodeID            int64        `json:"workflow_node_id" db:"workflow_node_id"`
	ApplicationID             int64        `json:"application_id" db:"application_id"`
	Application               *Application `json:"application,omitempty" db:"-"`
	Environment               *Environment `json:"environment,omitempty" db:"-"`
	EnvironmentID             int64        `json:"environment_id" db:"environment_id"`
	DefaultPayload            []Parameter  `json:"default_payload,omitempty" db:"-"`
	DefaultPipelineParameters []Parameter  `json:"default_pipeline_parameters,omitempty" db:"-"`
}

//WorkflowNodeHook represents a hook which cann trigger the workflow from a given node
type WorkflowNodeHook struct {
	ID                  int64                      `json:"id" db:"id"`
	UUID                string                     `json:"uuid" db:"uuid"`
	WorkflowNodeID      int64                      `json:"-" db:"workflow_node_id"`
	WorkflowHookModelID int64                      `json:"-" db:"workflow_hook_model_id"`
	WorkflowHookModel   WorkflowHookModel          `json:"model" db:"-"`
	Conditions          []WorkflowTriggerCondition `json:"conditions,omitempty" db:"-"`
	Config              WorkflowNodeHookConfig     `json:"config" db:"-"`
}

//WorkflowNodeHookConfig represents the configguration for a WorkflowNodeHook
type WorkflowNodeHookConfig map[string]string

//WorkflowHookModel represents a hook which can be used in workflows.
type WorkflowHookModel struct {
	ID            int64                  `json:"id" db:"id"`
	Name          string                 `json:"name" db:"name"`
	Type          string                 `json:"type"  db:"type"`
	Image         string                 `json:"image" db:"image"`
	Command       string                 `json:"command" db:"command"`
	DefaultConfig WorkflowNodeHookConfig `json:"default_config" db:"-"`
}

//WorkflowRun is an execution instance of a run
type WorkflowRun struct {
	ID               int64             `json:"id" db:"id"`
	Number           int64             `json:"num" db:"num"`
	ProjectID        int64             `json:"project_id,omitempty" db:"project_id"`
	WorkflowID       int64             `json:"workflow_id" db:"workflow_id"`
	Workflow         Workflow          `json:"workflow" db:"-"`
	Start            time.Time         `json:"start" db:"start"`
	LastModified     time.Time         `json:"last_modified" db:"last_modified"`
	WorkflowNodeRuns []WorkflowNodeRun `json:"nodes" db:"-"`
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
	Payload            []Parameter               `json:"payload" db:"-"`
	PipelineParameters []Parameter               `json:"pipeline_parameters" db:"-"`
	BuildParameters    []Parameter               `json:"build_parameters" db:"-"`
	Artifacts          []WorkflowNodeRunArtifact `json:"artifacts,omitempty" db:"-"`
	Tests              *venom.Tests              `json:"tests,omitempty" db:"-"`
	Commits            []VCSCommit               `json:"commits,omitempty" db:"-"`
}

//WorkflowNodeRunArtifact represents tests list
type WorkflowNodeRunArtifact struct {
	WorkflowNodeRunID int64  `json:"workflow_node_run_id" db:"workflow_node_run_id"`
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	Tag               string `json:"tag"`
	DownloadHash      string `json:"download_hash"`
	Size              int64  `json:"size,omitempty"`
	Perm              uint32 `json:"perm,omitempty"`
	MD5sum            string `json:"md5sum,omitempty"`
	ObjectPath        string `json:"object_path,omitempty"`
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

//WorkflowList return the list of the workflows for a project
func WorkflowList(projectkey string) ([]Workflow, error) {
	path := fmt.Sprintf("/project/%s/workflows", projectkey)
	body, _, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var ws = []Workflow{}
	if err := json.Unmarshal(body, &ws); err != nil {
		return nil, err
	}

	return ws, nil
}

//WorkflowGet returns a workflow given its name
func WorkflowGet(projectkey, name string) (*Workflow, error) {
	path := fmt.Sprintf("/project/%s/workflows/%s", projectkey, name)
	body, _, err := Request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var w = Workflow{}
	if err := json.Unmarshal(body, &w); err != nil {
		return nil, err
	}

	return &w, nil
}
