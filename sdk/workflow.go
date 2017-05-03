package sdk

import (
	"time"
)

//Workflow represents a pipeline based workflow
type Workflow struct {
	ID           int64         `json:"id" db:"id"`
	Name         string        `json:"name" db:"name"`
	Description  string        `json:"description" db:"description"`
	LastModified time.Time     `json:"last_modified" db:"last_modified"`
	ProjectID    int64         `json:"project_id" db:"project_id"`
	ProjectKey   string        `json:"project_key" db:"-"`
	RootID       int64         `json:"root_id" db:"root_node_id"`
	Root         *WorkflowNode `json:"root" db:"-"`
}

//WorkflowNode represents a node in w workflow tree
type WorkflowNode struct {
	ID         int64                 `json:"id" db:"id"`
	WorkflowID int64                 `json:"workflow_id" db:"workflow_id"`
	PipelineID int64                 `json:"pipeline_id" db:"pipeline_id"`
	Pipeline   Pipeline              `json:"pipeline" db:"-"`
	Context    *WorkflowNodeContext  `json:"context" db:"-"`
	Hooks      []WorkflowNodeHook    `json:"hooks" db:"-"`
	Triggers   []WorkflowNodeTrigger `json:"triggers" db:"-"`
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
	ID             int64        `json:"id" db:"id"`
	WorkflowNodeID int64        `json:"workflow_node_id" db:"workflow_node_id"`
	ApplicationID  int64        `json:"-" db:"application_id"`
	Application    *Application `json:"application" db:"-"`
	Environment    *Environment `json:"environment" db:"-"`
	EnvironmentID  int64        `json:"-" db:"environment_id"`
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
