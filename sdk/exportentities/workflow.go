package exportentities

import "github.com/ovh/cds/sdk"

type Workflow struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	// This will be filled for complex workflows
	Workflow map[string]WorkflowEntry `json:"workflow,omitempty" yaml:"workflow,omitempty"`
	Hooks    map[string][]HookEntry   `json:"hooks,omitempty" yaml:"hooks,omitempty"`
	// This will be filled for simple workflows
	DependsOn       []string                    `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Conditions      *sdk.WorkflowNodeConditions `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	When            []string                    `json:"when,omitempty" yaml:"when,omitempty"` //This is use only for manual and success condition
	PipelineName    string                      `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
	ApplicationName string                      `json:"application,omitempty" yaml:"application,omitempty"`
	EnvironmentName string                      `json:"environment,omitempty" yaml:"environment,omitempty"`
	PipelineHooks   []HookEntry                 `json:"pipeline_hooks,omitempty" yaml:"pipeline_hooks,omitempty"`
	Permissions     map[string]int              `json:"permissions,omitempty" yaml:"permissions,omitempty"`
}

type WorkflowEntry struct {
	DependsOn       []string                    `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Conditions      *sdk.WorkflowNodeConditions `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	When            []string                    `json:"when,omitempty" yaml:"when,omitempty"` //This is use only for manual and success condition
	PipelineName    string                      `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
	ApplicationName string                      `json:"application,omitempty" yaml:"application,omitempty"`
	EnvironmentName string                      `json:"environment,omitempty" yaml:"environment,omitempty"`
}

type HookEntry struct {
	Model  string            `json:"type,omitempty" yaml:"type,omitempty"`
	Config map[string]string `json:"config,omitempty" yaml:"config,omitempty"`
}

type WorkflowVersion string

const WorkflowVersion1 = "v1.0"

//NewWorkflow creates a new exportable workflow
func NewWorkflow(w sdk.Workflow, withPermission bool) (Workflow, error) {
	exportedWorkflow := Workflow{}
	exportedWorkflow.Version = WorkflowVersion1
	exportedWorkflow.Workflow = map[string]WorkflowEntry{}
	exportedWorkflow.Hooks = map[string][]HookEntry{}
	nodeIDs := w.Nodes()

	if withPermission {
		exportedWorkflow.Permissions = make(map[string]int, len(w.Groups))
		for _, p := range w.Groups {
			exportedWorkflow.Permissions[p.Group.Name] = p.Permission
		}
	}

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
		conditions := []sdk.WorkflowNodeCondition{}
		for _, c := range n.Context.Conditions.PlainConditions {
			if c.Operator == sdk.WorkflowConditionsOperatorEquals &&
				c.Value == "Success" &&
				c.Variable == "cds.status" {
				entry.When = append(entry.When, "success")
			} else if c.Operator == sdk.WorkflowConditionsOperatorEquals &&
				c.Value == "true" &&
				c.Variable == "cds.manual" {
				entry.When = append(entry.When, "manual")
			} else {
				conditions = append(conditions, c)
			}
		}

		if len(conditions) > 0 || n.Context.Conditions.LuaScript != "" {
			entry.Conditions = &sdk.WorkflowNodeConditions{
				PlainConditions: conditions,
				LuaScript:       n.Context.Conditions.LuaScript,
			}
		}

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
			return exportedWorkflow, sdk.ErrWorkflowNodeNotFound
		}
		entry, err := craftWorkflowEntry(n)
		if err != nil {
			return exportedWorkflow, err
		}
		exportedWorkflow.ApplicationName = entry.ApplicationName
		exportedWorkflow.PipelineName = entry.PipelineName
		exportedWorkflow.EnvironmentName = entry.EnvironmentName
		exportedWorkflow.DependsOn = entry.DependsOn
		if entry.Conditions != nil && (len(entry.Conditions.PlainConditions) > 0 || entry.Conditions.LuaScript != "") {
			exportedWorkflow.When = entry.When
			exportedWorkflow.Conditions = entry.Conditions
		}
		for _, h := range hooks {
			if exportedWorkflow.Hooks == nil {
				exportedWorkflow.Hooks = make(map[string][]HookEntry)
			}
			exportedWorkflow.PipelineHooks = append(exportedWorkflow.PipelineHooks, HookEntry{
				Model:  h.WorkflowHookModel.Name,
				Config: h.Config.Values(),
			})
		}
	} else {
		nodeIDs = append(nodeIDs, w.Root.ID)
		for _, id := range nodeIDs {
			n := w.GetNode(id)
			if n == nil {
				return exportedWorkflow, sdk.ErrWorkflowNodeNotFound
			}
			entry, err := craftWorkflowEntry(n)
			if err != nil {
				return exportedWorkflow, err
			}
			exportedWorkflow.Workflow[n.Name] = entry

		}

		for _, h := range hooks {
			if exportedWorkflow.Hooks == nil {
				exportedWorkflow.Hooks = make(map[string][]HookEntry)
			}
			exportedWorkflow.Hooks[w.GetNode(h.WorkflowNodeID).Name] = append(exportedWorkflow.Hooks[w.GetNode(h.WorkflowNodeID).Name], HookEntry{
				Model:  h.WorkflowHookModel.Name,
				Config: h.Config.Values(),
			})
		}
	}

	return exportedWorkflow, nil
}
