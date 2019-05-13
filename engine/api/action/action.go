package action

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
)

// Insert given action and its components in database.
func Insert(db gorp.SqlExecutor, a *sdk.Action) error {
	if a.Type == sdk.DefaultAction {
		a.Enabled = true
	}

	if err := insert(db, a); err != nil {
		return err
	}

	for i := range a.Actions {
		if err := insertActionChild(db, a.Actions[i], a.ID, i+1); err != nil {
			return err
		}
	}

	for i := range a.Parameters {
		if err := insertParameter(db, &actionParameter{
			ActionID:    a.ID,
			Name:        a.Parameters[i].Name,
			Type:        a.Parameters[i].Type,
			Value:       a.Parameters[i].Value,
			Description: a.Parameters[i].Description,
			Advanced:    a.Parameters[i].Advanced,
		}); err != nil {
			return sdk.WrapError(err, "cannot insert action parameter %s", a.Parameters[i].Name)
		}
	}

	for i := range a.Requirements {
		r := a.Requirements[i]
		r.ActionID = a.ID
		if err := InsertRequirement(db, &r); err != nil {
			return err
		}
	}

	return nil
}

// Update given action and its components in database.
func Update(db gorp.SqlExecutor, a *sdk.Action) error {
	if a.Type == sdk.DefaultAction {
		a.Enabled = true
	}

	if err := update(db, a); err != nil {
		return err
	}

	if err := deleteEdgesByParentID(db, a.ID); err != nil {
		return err
	}
	for i := range a.Actions {
		if err := insertActionChild(db, a.Actions[i], a.ID, i+1); err != nil {
			return err
		}
	}

	if err := deleteParametersByActionID(db, a.ID); err != nil {
		return err
	}
	for i := range a.Parameters {
		if err := insertParameter(db, &actionParameter{
			ActionID:    a.ID,
			Name:        a.Parameters[i].Name,
			Type:        a.Parameters[i].Type,
			Value:       a.Parameters[i].Value,
			Description: a.Parameters[i].Description,
			Advanced:    a.Parameters[i].Advanced,
		}); err != nil {
			return sdk.WrapError(err, "cannot insert action parameter %s", a.Parameters[i].Name)
		}
	}

	if err := DeleteRequirementsByActionID(db, a.ID); err != nil {
		return err
	}
	for i := range a.Requirements {
		r := a.Requirements[i]
		r.ActionID = a.ID
		if err := InsertRequirement(db, &r); err != nil {
			return err
		}
	}

	return nil
}

// RetrieveForGroupAndName try to find an action for given group and name.
func RetrieveForGroupAndName(db gorp.SqlExecutor, g *sdk.Group, name string) (*sdk.Action, error) {
	if g != nil {
		grp, err := group.LoadGroupByName(db, g.Name)
		if err != nil {
			return nil, err
		}

		a, err := LoadTypeDefaultByNameAndGroupID(db, name, grp.ID,
			LoadOptions.WithRequirements,
			LoadOptions.WithParameters,
			LoadOptions.WithGroup,
		)
		if err != nil {
			return nil, err
		}
		if a == nil {
			return nil, sdk.NewErrorFrom(sdk.ErrNoAction, "invalid given action %s for group %s", name, grp.Name)
		}

		return a, nil
	}

	a, err := LoadTypeDefaultByNameAndGroupID(db, name, group.SharedInfraGroup.ID,
		LoadOptions.WithRequirements,
		LoadOptions.WithParameters,
		LoadOptions.WithGroup,
	)
	if err != nil {
		return nil, err
	}
	if a != nil {
		return a, nil
	}

	a, err = LoadByTypesAndName(db, []string{sdk.BuiltinAction, sdk.PluginAction}, name,
		LoadOptions.WithRequirements,
		LoadOptions.WithParameters,
		LoadOptions.WithGroup,
	)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, sdk.NewErrorFrom(sdk.ErrNoAction, "invalid given action %s", name)
	}

	return a, nil
}
