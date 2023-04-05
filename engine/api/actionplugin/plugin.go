package actionplugin

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/sdk"
)

// InsertWithGRPCPlugin creates action in database
func InsertWithGRPCPlugin(db gorp.SqlExecutor, pl *sdk.GRPCPlugin, params []sdk.Parameter) (*sdk.Action, error) {
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
		Parameters: params,
		Enabled:    true,
	}

	if err := action.Insert(db, &a); err != nil {
		return nil, err
	}

	return &a, nil
}

// UpdateGRPCPlugin creates action in database
func UpdateGRPCPlugin(ctx context.Context, db gorp.SqlExecutor, pl *sdk.GRPCPlugin, params []sdk.Parameter) (*sdk.Action, error) {
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
		Parameters: params,
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

// DeleteGRPCPlugin delete action in database
func DeleteGRPCPlugin(ctx context.Context, db gorp.SqlExecutor, pl *sdk.GRPCPlugin) error {
	act, err := action.LoadByTypesAndName(ctx, db, []string{sdk.PluginAction}, pl.Name, action.LoadOptions.Default)
	if err != nil {
		return err
	}

	if err := action.Delete(db, act); err != nil {
		return err
	}

	return nil
}
