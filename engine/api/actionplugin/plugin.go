package actionplugin

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/sdk"
)

// InsertWithGRPCPlugin creates action in database
func InsertWithGRPCPlugin(db gorp.SqlExecutor, pl *sdk.GRPCPlugin, inputs map[string]sdk.PluginInput) (*sdk.Action, error) {
	a := sdk.Action{
		Name:        pl.Name,
		Type:        sdk.PluginAction,
		Description: pl.Description,
		Requirements: sdk.RequirementList{
			sdk.Requirement{
				Name:  pl.Name,
				Type:  sdk.PluginRequirement,
				Value: pl.Name,
			},
		},
		Parameters: inputsToParameters(inputs),
		Enabled:    true,
	}

	if err := action.Insert(db, &a); err != nil {
		return nil, err
	}

	return &a, nil
}

// UpdateGRPCPlugin creates action in database
func UpdateGRPCPlugin(ctx context.Context, db gorp.SqlExecutor, pl *sdk.GRPCPlugin, inputs map[string]sdk.PluginInput) (*sdk.Action, error) {
	a := sdk.Action{
		Name:        pl.Name,
		Type:        sdk.PluginAction,
		Description: pl.Description,
		Requirements: sdk.RequirementList{
			sdk.Requirement{
				Name:  pl.Name,
				Type:  sdk.PluginRequirement,
				Value: pl.Name,
			},
		},
		Parameters: inputsToParameters(inputs),
		Enabled:    true,
	}

	oldA, err := action.LoadByTypesAndName(ctx, db, []string{sdk.PluginAction}, a.Name, action.LoadOptions.Default)
	if err != nil {
		return nil, err
	}
	a.ID = oldA.ID

	if err := action.Update(db, &a); err != nil {
		return nil, err
	}

	return &a, nil
}

func inputsToParameters(inputs map[string]sdk.PluginInput) []sdk.Parameter {
	params := make([]sdk.Parameter, 0, len(inputs))
	for k, v := range inputs {
		p := sdk.Parameter{
			Name:        k,
			Type:        v.Type,
			Value:       v.Default,
			Description: v.Description,
			Advanced:    v.Advanced,
		}
		params = append(params, p)
	}
	return params
}
