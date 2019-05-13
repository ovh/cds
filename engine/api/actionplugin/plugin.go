package actionplugin

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/sdk"
)

//InsertWithGRPCPlugin creates action in database
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

//UpdateGRPCPlugin creates action in database
func UpdateGRPCPlugin(db gorp.SqlExecutor, pl *sdk.GRPCPlugin, params []sdk.Parameter, userID int64) (*sdk.Action, error) {
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

	oldA, err := action.LoadByTypesAndName(db, []string{sdk.PluginAction}, a.Name, action.LoadOptions.Default)
	if err != nil {
		return nil, err
	}
	a.ID = oldA.ID

	if err := action.Update(db, &a); err != nil {
		return nil, err
	}

	return &a, nil
}
