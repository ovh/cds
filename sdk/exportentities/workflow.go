package exportentities

import "github.com/ovh/cds/sdk"

type Workflow struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	// This will be filled for complex workflows
	Workflow map[string]WorkflowEntry `json:"workflow,omitempty" yaml:"workflow,omitempty"`
	Hooks    map[string][]HookEntry   `json:"hooks,omitempty" yaml:"hooks,omitempty"`
	// This will be filled for simple workflows
	DependsOn       []string                   `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Conditions      sdk.WorkflowNodeConditions `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	PipelineName    string                     `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
	ApplicationName string                     `json:"application,omitempty" yaml:"application,omitempty"`
	EnvironmentName string                     `json:"environment,omitempty" yaml:"environment,omitempty"`
	PipelineHooks   []HookEntry                `json:"pipeline_hooks,omitempty" yaml:"pipeline_hooks,omitempty"`
}

type WorkflowEntry struct {
	DependsOn       []string                   `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Conditions      sdk.WorkflowNodeConditions `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	PipelineName    string                     `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
	ApplicationName string                     `json:"application,omitempty" yaml:"application,omitempty"`
	EnvironmentName string                     `json:"environment,omitempty" yaml:"environment,omitempty"`
}

type HookEntry struct {
	Model  string            `json:"type,omitempty" yaml:"type,omitempty"`
	Config map[string]string `json:"config,omitempty" yaml:"config,omitempty"`
}

type WorkflowVersion string

const WorkflowVersion1 = "v1.0"

func NewWorkflow(w sdk.Workflow, version WorkflowVersion) (Workflow, error) {
	e := Workflow{}
	e.Version = string(version)
	e.Workflow = map[string]WorkflowEntry{}
	e.Hooks = map[string][]HookEntry{}
	nodeIDs := w.Nodes()

	var craftWorkflowEntry = func(n *sdk.WorkflowNode) (WorkflowEntry, error) {
		entry := WorkflowEntry{}

		ancestorIDs := n.Ancestors(&w, false)
		ancestors := []string{}
		for _, aID := range ancestorIDs {
			a := w.GetNode(aID)
			if a == nil {
				return entry, sdk.ErrWorkflowNodeNotFound
			}
			ancestors = append(ancestors, a.Name)
		}

		entry.DependsOn = ancestors
		entry.PipelineName = n.Pipeline.Name
		entry.Conditions = n.Context.Conditions

		if n.Context.Application != nil {
			entry.ApplicationName = n.Context.Application.Name
		}
		if n.Context.Environment != nil {
			entry.EnvironmentName = n.Context.Environment.Name
		}
		return entry, nil
	}

	hooks := w.GetHooks()

	if len(nodeIDs) == 0 {
		n := w.GetNode(w.Root.ID)
		if n == nil {
			return e, sdk.ErrWorkflowNodeNotFound
		}
		entry, err := craftWorkflowEntry(n)
		if err != nil {
			return e, err
		}
		e.ApplicationName = entry.ApplicationName
		e.PipelineName = entry.PipelineName
		e.EnvironmentName = entry.EnvironmentName
		e.DependsOn = entry.DependsOn
		e.Conditions = entry.Conditions
		for _, h := range hooks {
			//TODO check != nil
			e.PipelineHooks = append(e.PipelineHooks, HookEntry{
				Model:  h.WorkflowHookModel.Name,
				Config: h.Config.Values(),
			})
		}
	} else {
		nodeIDs = append(nodeIDs, w.Root.ID)
		for _, id := range nodeIDs {
			n := w.GetNode(id)
			if n == nil {
				return e, sdk.ErrWorkflowNodeNotFound
			}
			entry, err := craftWorkflowEntry(n)
			if err != nil {
				return e, err
			}
			e.Workflow[n.Name] = entry
		}
		for _, h := range hooks {
			//TODO check != nil
			e.Hooks[w.GetNode(h.WorkflowNodeID).Name] = append(e.Hooks[w.GetNode(h.WorkflowNodeID).Name], HookEntry{
				Model:  h.WorkflowHookModel.Name,
				Config: h.Config.Values(),
			})
		}
	}

	return e, nil
}
