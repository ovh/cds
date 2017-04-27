package sdk

//Workflow represents a pipeline based workflow
type Workflow struct {
	ID         int64         `json:"id" db:"id"`
	Name       string        `json:"name" db:"name"`
	ProjectID  int64         `json:"-" db:"project_id"`
	ProjectKey int64         `json:"project_key" db:"-"`
	Root       *WorkflowNode `json:"root" db:"-"`
}

//WorkflowNode represents a node in w workflow tree
type WorkflowNode struct {
	ID         int64                `json:"id" db:"id"`
	WorkflowID int64                `json:"workflow_id" db:"workflow_id"`
	PipelineID int64                `json:"-" db:"pipeline_id"`
	Pipeline   Pipeline             `json:"pipeline" db:"-"`
	Context    *WorkflowNodeContext `json:"context" db:"-"`
	Hooks      []WorkflowNodeHook   `json:"hooks" db:"-"`
}

//WorkflowNodeContext represents a context attached on a node
type WorkflowNodeContext struct {
	ID             int64        `json:"id" db:"id"`
	WorkflowNodeID int64        `json:"workflow_node_id" db:"workflow_node_id"`
	ApplicationID  int64        `json:"-" db:"pipeline_id"`
	Application    *Application `json:"application" db:"-"`
	Environment    *Environment `json:"environment" db:"-"`
	EnvironmentID  int64        `json:"-" db:"environment_id"`
}

//WorkflowNodeHook represents a hook which cann trigger the workflow from a given node
type WorkflowNodeHook struct {
	ID                  int64                  `json:"id" db:"id"`
	UUID                string                 `json:"string" db:"string"`
	WorkflowNodeID      int64                  `json:"-" db:"workflow_node_id"`
	WorkflowHookModelID int64                  `json:"-" db:"workflow_hook_model_id"`
	Prerequisites       []Prerequisite         `json:"prerequisites" db:"-"`
	Config              WorkflowNodeHookConfig `json:"config" db:"-"`
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
