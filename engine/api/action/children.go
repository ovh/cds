package action

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func insertActionChild(db gorp.SqlExecutor, child sdk.Action, actionID int64, execOrder int) error {
	// useful to not save a step_name if it's the same than the default name (for ascode)
	if strings.ToLower(child.Name) == strings.ToLower(child.StepName) && child.Type != sdk.AsCodeAction {
		child.StepName = ""
	}

	ae := actionEdge{
		ParentID:       actionID,
		ChildID:        child.ID,
		ExecOrder:      int64(execOrder), // TODO exec order can be int 64
		StepName:       child.StepName,
		Optional:       child.Optional,
		AlwaysExecuted: child.AlwaysExecuted,
		Enabled:        child.Enabled,
	}
	if err := insertEdge(db, &ae); err != nil {
		return err
	}

	// insert all parameters
	for i := range child.Parameters {
		// default value for parameter type list should be the first item ("aa;bb;cc" -> "aa")
		if child.Parameters[i].Type == sdk.ListParameter && strings.Contains(child.Parameters[i].Value, ";") {
			child.Parameters[i].Value = strings.Split(child.Parameters[i].Value, ";")[0]
		}

		if err := insertEdgeParameter(db, &actionEdgeParameter{
			ActionEdgeID: ae.ID,
			Name:         child.Parameters[i].Name,
			Type:         child.Parameters[i].Type,
			Value:        child.Parameters[i].Value,
			Description:  child.Parameters[i].Description,
			Advanced:     child.Parameters[i].Advanced,
		}); err != nil {
			return err
		}
	}

	return nil
}

// CheckChildrenForGroupIDs returns an error if given children not found.
func CheckChildrenForGroupIDs(ctx context.Context, db gorp.SqlExecutor, a *sdk.Action, groupIDs []int64) error {
	if len(a.Actions) == 0 {
		return nil
	}

	childrenIDs := a.ToUniqueChildrenIDs()

	children, err := LoadAllByIDsWithTypeBuiltinOrPluginOrDefaultInGroupIDs(ctx, db, childrenIDs, groupIDs, LoadOptions.WithChildren)
	if err != nil {
		return err
	}
	return handleChildrenError(a, children)
}

// CheckChildrenForGroupIDsWithLoop return an error if given children not found or tree loop detected.
func CheckChildrenForGroupIDsWithLoop(ctx context.Context, db gorp.SqlExecutor, a *sdk.Action, groupIDs []int64) error {
	return checkChildrenForGroupIDsWithLoopStep(ctx, db, a, a, groupIDs)
}

func checkChildrenForGroupIDsWithLoopStep(ctx context.Context, db gorp.SqlExecutor, root, current *sdk.Action, groupIDs []int64) error {
	if len(current.Actions) == 0 {
		return nil
	}

	childrenIDs := current.ToUniqueChildrenIDs()

	// children ids should not contains root action id
	for i := range childrenIDs {
		if childrenIDs[i] == root.ID {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "action loop usage detected for given steps")
		}
	}

	children, err := LoadAllByIDsWithTypeBuiltinOrPluginOrDefaultInGroupIDs(ctx, db, childrenIDs, groupIDs, LoadOptions.WithChildren)
	if err != nil {
		return err
	}
	if err := handleChildrenError(current, children); err != nil {
		return err
	}

	for i := range children {
		if err := checkChildrenForGroupIDsWithLoopStep(ctx, db, root, &children[i], groupIDs); err != nil {
			return err
		}
	}

	return nil
}

func handleChildrenError(current *sdk.Action, children []sdk.Action) error {
	childrenIDs := current.ToUniqueChildrenIDs()

	if len(children) == len(childrenIDs) {
		return nil
	}

	// construct list of children not found names or ids
	mChildren := make(map[int64]sdk.Action, len(children))
	for i := range children {
		mChildren[children[i].ID] = children[i]
	}

	notFoundChildrenIDs := make([]int64, 0, len(childrenIDs))
	for i := range childrenIDs {
		if _, ok := mChildren[childrenIDs[i]]; !ok {
			notFoundChildrenIDs = append(notFoundChildrenIDs, childrenIDs[i])
		}
	}

	notFoundChildrenRefs := make([]string, len(notFoundChildrenIDs))
	for i := range notFoundChildrenIDs {
		for j := range current.Actions {
			if current.Actions[j].ID == notFoundChildrenIDs[i] {
				if current.Actions[j].Group != nil && current.Actions[j].Name != "" {
					notFoundChildrenRefs[i] = fmt.Sprintf("path: %s/%s", current.Actions[j].Group.Name, current.Actions[j].Name)
				} else {
					notFoundChildrenRefs[i] = fmt.Sprintf("id: %d", current.Actions[j].ID)
				}
			}
		}
	}
	return sdk.NewErrorFrom(sdk.ErrWrongRequest, "some given step actions are not usable: (%s). Please check you have correct permissions on your project and your workflow", strings.Join(notFoundChildrenRefs, ", "))
}
