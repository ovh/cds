package exportentities

import "github.com/ovh/cds/sdk"

type Workflow map[string]WorkflowEntry

type WorkflowEntry struct {
	DependsOn       []string                     `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Conditions      []sdk.WorkflowNodeConditions `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	Pipeline        *Pipeline                    `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
	PipelineName    string                       `json:"pipeline_name,omitempty" yaml:"pipeline_name,omitempty"`
	Application     *Application                 `json:"application,omitempty" yaml:"application,omitempty"`
	ApplicationName string                       `json:"application_name,omitempty" yaml:"application_name,omitempty"`
	Environment     *Environment                 `json:"environment,omitempty" yaml:"environment,omitempty"`
	EnvironmentName string                       `json:"environment_name,omitempty" yaml:"environment_name,omitempty"`
	Hooks           map[string]HookEntry         `json:"hooks,omitempty" yaml:"hooks,omitempty"`
}

type HookEntry map[string]string

func NewWorkflow(w sdk.Workflow, deep bool) (Workflow, error) {
	e := Workflow{}
	nodeIDs := w.Nodes()
	nodeIDs = append(nodeIDs, w.Root.ID)
	for _, id := range nodeIDs {
		n := w.GetNode(id)
		if n == nil {
			return e, sdk.ErrWorkflowNodeNotFound
		}
		ancestorIDs := n.Ancestors(&w, false)
		ancestors := []string{}
		for _, aID := range ancestorIDs {
			a := w.GetNode(aID)
			if a == nil {
				return e, sdk.ErrWorkflowNodeNotFound
			}
			ancestors = append(ancestors, a.Name)
		}

		entry := WorkflowEntry{}
		entry.DependsOn = ancestors
		if deep {
			entry.Pipeline = NewPipeline(&n.Pipeline)
		} else {
			entry.PipelineName = n.Pipeline.Name
		}

		if n.Context.Application != nil {
			if deep {
				entry.Application = NewApplication(n.Context.Application)
			} else {
				entry.ApplicationName = n.Context.Application.Name
			}
		}
		if n.Context.Environment != nil {
			if deep {
				entry.Environment = NewEnvironment(n.Context.Environment)
			} else {
				entry.EnvironmentName = n.Context.Environment.Name
			}
		}
		e[n.Name] = entry
	}
	return e, nil
}
