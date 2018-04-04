package sdk

// SharedInfraGroupName is the name of the builtin group used to share infrastructure between projects
const SharedInfraGroupName = "shared.infra"

// Group represent a group of user.
type Group struct {
	ID     int64   `json:"id" yaml:"-"`
	Name   string  `json:"name" yaml:"name" cli:"name,key"`
	Admins []User  `json:"admins,omitempty" yaml:"admins,omitempty"`
	Users  []User  `json:"users,omitempty" yaml:"users,omitempty"`
	Tokens []Token `json:"tokens,omitempty" yaml:"tokens,omitempty"`
}

// GroupPermission represent a group and his role in the project
type GroupPermission struct {
	Group      Group `json:"group"`
	Permission int   `json:"permission"`
}

// EnvironmentGroup represent a link with a pipeline
type EnvironmentGroup struct {
	Environment Environment `json:"environment"`
	Permission  int         `json:"permission"`
}

// ApplicationGroup represent a link with a pipeline
type ApplicationGroup struct {
	Application Application `json:"application"`
	Permission  int         `json:"permission"`
}

// PipelineGroup represent a link with a pipeline
type PipelineGroup struct {
	Pipeline   Pipeline `json:"pipeline"`
	Permission int      `json:"permission"`
}

// ProjectGroup represent a link with a project
type ProjectGroup struct {
	Project    Project `json:"project"`
	Permission int     `json:"permission"`
}

// WorkflowGroup represents the permission to a workflow
type WorkflowGroup struct {
	Workflow   Workflow `json:"workflow"`
	Permission int      `json:"permission"`
}
