package workflow

import (
	"sort"
	"strings"

	"github.com/ovh/cds/sdk"
)

// Sort sorts all the workflow tree
func Sort(w *sdk.Workflow) {
	if w == nil {
		return
	}
	SortNode(w.Root)
}

// SortNode sort the content of a node
func SortNode(n *sdk.WorkflowNode) {
	if n == nil {
		return
	}
	sortNodeHooks(&n.Hooks)
	sortNodeTriggers(&n.Triggers)
}

func sortNodeHooks(hooks *[]sdk.WorkflowNodeHook) {
	if hooks == nil {
		return
	}
	sort.Slice(*hooks, func(i, j int) bool {
		return (*hooks)[i].UUID < (*hooks)[j].UUID
	})
}

func sortNodeTriggers(triggers *[]sdk.WorkflowNodeTrigger) {
	if triggers == nil {
		return
	}
	for i := range *triggers {
		sortNodeHooks(&(*triggers)[i].WorkflowDestNode.Hooks)
	}

	sort.Slice(*triggers, func(i, j int) bool {
		t1 := &(*triggers)[i]
		t2 := &(*triggers)[j]

		return strings.Compare(t1.WorkflowDestNode.Name, t2.WorkflowDestNode.Name) < 0
	})
}
