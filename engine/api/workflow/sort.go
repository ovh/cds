package workflow

import (
	"sort"
	"strings"

	"github.com/ovh/cds/sdk"
)

// Sort sorts all the workflow tree
func Sort(w *sdk.Workflow) {
	sortNode(w.Root)
}

func sortNode(n *sdk.WorkflowNode) {
	sortNodeHooks(&n.Hooks)
	sortNodeTriggers(&n.Triggers)
}

func sortNodeHooks(hooks *[]sdk.WorkflowNodeHook) {
	for i := range *hooks {
		sortConditions(&(*hooks)[i].Conditions)
	}

	sort.Slice(*hooks, func(i, j int) bool {
		return (*hooks)[i].UUID < (*hooks)[j].UUID
	})

}

func sortNodeTriggers(triggers *[]sdk.WorkflowNodeTrigger) {
	for i := range *triggers {
		sortConditions(&(*triggers)[i].Conditions)
		sortNodeHooks(&(*triggers)[i].WorkflowDestNode.Hooks)
	}

	sort.Slice(*triggers, func(i, j int) bool {
		t1 := &(*triggers)[i]
		t2 := &(*triggers)[j]

		if t1.WorkflowDestNode.Pipeline.Name != t2.WorkflowDestNode.Pipeline.Name {
			return strings.Compare(t1.WorkflowDestNode.Pipeline.Name, t2.WorkflowDestNode.Pipeline.Name) < 0
		}

		// Without context first
		if t1.WorkflowDestNode.Context == nil {
			return true
		}

		if t1.WorkflowDestNode.Context != nil && t2.WorkflowDestNode.Context != nil {

			// Without app first
			if t1.WorkflowDestNode.Context.Application == nil && t2.WorkflowDestNode.Context.Application != nil {
				return true
			}

			if t1.WorkflowDestNode.Context.Application == nil && t2.WorkflowDestNode.Context.Application == nil {
				// Check Env
				return sortEnvironment(t1.WorkflowDestNode.Context, t2.WorkflowDestNode.Context)
			}

			if t1.WorkflowDestNode.Context.Application != nil && t2.WorkflowDestNode.Context.Application != nil {
				// Check app Name
				if t1.WorkflowDestNode.Context.Application.Name == t2.WorkflowDestNode.Context.Application.Name {
					// Check Env
					return sortEnvironment(t1.WorkflowDestNode.Context, t2.WorkflowDestNode.Context)
				} else {
					return strings.Compare(t1.WorkflowDestNode.Context.Application.Name, t2.WorkflowDestNode.Context.Application.Name) < 0
				}
			}

		}

		return false
	})
}

func sortConditions(conditions *[]sdk.WorkflowTriggerCondition) {
	sort.Slice(*conditions, func(i, j int) bool {
		return strings.Compare((*conditions)[i].Variable, (*conditions)[j].Variable) < 0
	})
}

func sortEnvironment(c1, c2 *sdk.WorkflowNodeContext) bool {
	if c1.Environment == nil {
		return true
	}
	if c1.Environment != nil && c2.Environment != nil {
		if c1.Environment.Name == c2.Environment.Name {
			return true
		} else {
			return strings.Compare(c1.Application.Name, c2.Application.Name) < 0
		}
	}
	return false
}
