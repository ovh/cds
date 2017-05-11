package sdk

import (
	"time"
)

//Workflow represents a pipeline based workflow
type Workflow struct {
	ID           int64              `json:"id" db:"id"`
	Name         string             `json:"name" db:"name"`
	Description  string             `json:"description" db:"description"`
	LastModified time.Time          `json:"last_modified" db:"last_modified"`
	ProjectID    int64              `json:"project_id" db:"project_id"`
	ProjectKey   string             `json:"project_key" db:"-"`
	RootID       int64              `json:"root_id" db:"root_node_id"`
	Root         *WorkflowNode      `json:"root" db:"-"`
	Joins        []WorkflowNodeJoin `json:"joins" db:"-"`
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
	SourceNodeIDs  []int64                   `json:"source_node_id" db:"-"`
	SourceNodeRefs []string                  `json:"source_node_ref" db:"-"`
	Triggers       []WorkflowNodeJoinTrigger `json:"triggers" db:"-"`
}

//WorkflowNodeJoinTrigger is a trigger for joins
type WorkflowNodeJoinTrigger struct {
	ID                 int64                      `json:"id" db:"id"`
	WorkflowNodeJoinID int64                      `json:"join_id" db:"workflow_node_join_id"`
	WorkflowDestNodeID int64                      `json:"workflow_dest_node_id" db:"workflow_dest_node_id"`
	WorkflowDestNode   WorkflowNode               `json:"workflow_dest_node" db:"-"`
	Conditions         []WorkflowTriggerCondition `json:"conditions" db:"-"`
}

//WorkflowNode represents a node in w workflow tree
type WorkflowNode struct {
	ID         int64                 `json:"id" db:"id"`
	Ref        string                `json:"ref,omitempty" db:"-"`
	WorkflowID int64                 `json:"workflow_id" db:"workflow_id"`
	PipelineID int64                 `json:"pipeline_id" db:"pipeline_id"`
	Pipeline   Pipeline              `json:"pipeline" db:"-"`
	Context    *WorkflowNodeContext  `json:"context" db:"-"`
	Hooks      []WorkflowNodeHook    `json:"hooks" db:"-"`
	Triggers   []WorkflowNodeTrigger `json:"triggers" db:"-"`
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
		res = []int64{n.Context.ApplicationID}
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

//InvolvedEnvironments returns all environments used in the workflow
func (n *WorkflowNode) InvolvedEnvironments() []int64 {
	res := []int64{}
	if n.Context != nil {
		if n.Context.EnvironmentID == 0 && n.Context.Environment != nil {
			n.Context.EnvironmentID = n.Context.Environment.ID
		}
		res = []int64{n.Context.EnvironmentID}
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
	Conditions         []WorkflowTriggerCondition `json:"conditions" db:"-"`
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
	ApplicationID             int64        `json:"-" db:"application_id"`
	Application               *Application `json:"application" db:"-"`
	Environment               *Environment `json:"environment" db:"-"`
	EnvironmentID             int64        `json:"-" db:"environment_id"`
	DefaultPayload            []Parameter  `json:"default_payload" db:"-"`
	DefaultPipelineParameters []Parameter  `json:"default_pipeline_parameters" db:"-"`
}

//WorkflowNodeHook represents a hook which cann trigger the workflow from a given node
type WorkflowNodeHook struct {
	ID                  int64                      `json:"id" db:"id"`
	UUID                string                     `json:"uuid" db:"uuid"`
	WorkflowNodeID      int64                      `json:"-" db:"workflow_node_id"`
	WorkflowHookModelID int64                      `json:"-" db:"workflow_hook_model_id"`
	WorkflowHookModel   WorkflowHookModel          `json:"model" db:"-"`
	Conditions          []WorkflowTriggerCondition `json:"conditions" db:"-"`
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
