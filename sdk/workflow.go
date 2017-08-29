package sdk

import (
	"encoding/json"
	"fmt"
	"time"
)

//Workflow represents a pipeline based workflow
type Workflow struct {
	ID           int64              `json:"id" db:"id" cli:"-"`
	Name         string             `json:"name" db:"name" cli:"name,key"`
	Description  string             `json:"description,omitempty" db:"description" cli:"description"`
	LastModified time.Time          `json:"last_modified" db:"last_modified"`
	ProjectID    int64              `json:"project_id,omitempty" db:"project_id" cli:"-"`
	ProjectKey   string             `json:"project_key" db:"-" cli:"-"`
	RootID       int64              `json:"root_id,omitempty" db:"root_node_id" cli:"-"`
	Root         *WorkflowNode      `json:"root" db:"-" cli:"-"`
	Joins        []WorkflowNodeJoin `json:"joins,omitempty" db:"-" cli:"-"`
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

//GetJoin returns the join given its id
func (w *Workflow) GetJoin(id int64) *WorkflowNodeJoin {
	for _, j := range w.Joins {
		if j.ID == id {
			return &j
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
	Name             string                `json:"name" db:"name"`
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

// EqualsTo returns true if a node has the same pipeline and context than another
func (n *WorkflowNode) EqualsTo(n1 *WorkflowNode) bool {
	if n.PipelineID != n1.PipelineID {
		return false
	}
	if n.Context == nil && n1.Context != nil {
		return false
	}
	if n.Context != nil && n1.Context == nil {
		return false
	}
	if n.Context.ApplicationID != n1.Context.ApplicationID {
		return false
	}
	if n.Context.EnvironmentID != n1.Context.EnvironmentID {
		return false
	}
	return true
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
	DefaultPayload            interface{}  `json:"default_payload,omitempty" db:"-"`
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

var WorkflowHookModelBuiltin = "builtin"

//WorkflowNodeHookConfig represents the configguration for a WorkflowNodeHook
type WorkflowNodeHookConfig map[string]string

//WorkflowHookModel represents a hook which can be used in workflows.
type WorkflowHookModel struct {
	ID            int64                  `json:"id" db:"id" cli:"-"`
	Name          string                 `json:"name" db:"name" cli:"name"`
	Type          string                 `json:"type"  db:"type"`
	Author        string                 `json:"author" db:"author"`
	Description   string                 `json:"description" db:"description"`
	Identifier    string                 `json:"identifier" db:"identifier"`
	Icon          string                 `json:"-" db:"icon"`
	Image         string                 `json:"image" db:"image"`
	Command       string                 `json:"command" db:"command"`
	DefaultConfig WorkflowNodeHookConfig `json:"default_config" db:"-"`
	Disabled      bool                   `json:"disabled" db:"disabled"`
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

// WorkflowDelete Call API to delete a workflow
func WorkflowDelete(projectkey, name string) error {
	path := fmt.Sprintf("/project/%s/workflows/%s", projectkey, name)
	_, _, err := Request("DELETE", path, nil)
	return err
}
