package action

import (
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func insertActionChild(db gorp.SqlExecutor, child sdk.Action, actionID int64, execOrder int) error {
	// useful to not save a step_name if it's the same than the default name (for ascode)
	if strings.ToLower(child.Name) == strings.ToLower(child.StepName) {
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
func CheckChildrenForGroupIDs(db gorp.SqlExecutor, a *sdk.Action, groupIDs []int64) error {
	if len(a.Actions) == 0 {
		return nil
	}

	childrenIDs := a.ToUniqueChildrenIDs()

	// children should be builtin, plugin or default with group matching
	query := gorpmapping.NewQuery(`
  	SELECT *
		FROM action
		WHERE
			id = ANY(string_to_array($1, ',')::int[])
			AND (
				type = $2
				OR type = $3
				OR (type = $4 AND group_id = ANY(string_to_array($5, ',')::int[]))
			)
	`).Args(
		gorpmapping.IDsToQueryString(childrenIDs),
		sdk.BuiltinAction,
		sdk.PluginAction,
		sdk.DefaultAction,
		gorpmapping.IDsToQueryString(groupIDs),
	)
	children, err := getAll(db, query)
	if err != nil {
		return err
	}
	if len(children) != len(childrenIDs) {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "some given step actions are not usable")
	}

	return nil
}

// CheckChildrenForGroupIDsWithLoop return an error if given children not found or tree loop detected.
func CheckChildrenForGroupIDsWithLoop(db gorp.SqlExecutor, a *sdk.Action, groupIDs []int64) error {
	return checkChildrenForGroupIDsWithLoopStep(db, a, a, groupIDs)
}

func checkChildrenForGroupIDsWithLoopStep(db gorp.SqlExecutor, root, current *sdk.Action, groupIDs []int64) error {
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

	// children should be builtin, plugin or default with group matching
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM action
		WHERE
			id = ANY(string_to_array($1, ',')::int[])
			AND (
				type = $2
				OR type = $3
				OR (type = $4 AND group_id = ANY(string_to_array($5, ',')::int[]))
			)
	`).Args(
		gorpmapping.IDsToQueryString(childrenIDs),
		sdk.BuiltinAction,
		sdk.PluginAction,
		sdk.DefaultAction,
		gorpmapping.IDsToQueryString(groupIDs),
	)
	children, err := getAll(db, query)
	if err != nil {
		return err
	}
	if len(children) != len(childrenIDs) {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "some given step actions are not usable")
	}

	for i := range children {
		if err := checkChildrenForGroupIDsWithLoopStep(db, root, &children[i], groupIDs); err != nil {
			return err
		}
	}

	return nil
}
