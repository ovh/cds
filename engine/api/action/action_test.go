package action_test

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestCRUD(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	grp2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	t.Cleanup(func() {
		assets.DeleteTestGroup(t, db, grp1)
		assets.DeleteTestGroup(t, db, grp2)
	})

	scriptAction := assets.GetBuiltinOrPluginActionByName(t, db, "Script")

	acts := []sdk.Action{
		{
			GroupID:     &grp1.ID,
			Type:        sdk.DefaultAction,
			Name:        sdk.RandomString(10),
			Description: "My action 1 description",
			Parameters: []sdk.Parameter{
				{
					Name: "my-bool",
					Type: sdk.BooleanParameter,
				},
				{
					Name: "my-string",
					Type: sdk.StringParameter,
				},
			},
			Requirements: []sdk.Requirement{
				{
					Name:  "my-service",
					Type:  sdk.ServiceRequirement,
					Value: "my-service",
				},
			},
			Actions: []sdk.Action{
				{
					ID: scriptAction.ID,
					Parameters: []sdk.Parameter{
						{
							Name:  "script",
							Type:  sdk.TextParameter,
							Value: "echo \"test\"",
						},
					},
				},
			},
		},
		{
			GroupID: &grp2.ID,
			Type:    sdk.DefaultAction,
			Name:    sdk.RandomString(10),
		},
	}

	// Insert
	for i := range acts {
		require.NoError(t, action.Insert(db, &acts[i]), "No err should be returned when inserting an action")
	}

	// Update
	acts[0].Parameters = append(acts[0].Parameters, sdk.Parameter{
		Name: "my-number",
		Type: sdk.NumberParameter,
	})
	require.NoError(t, action.Update(db, &acts[0]), "No err should be returned when updating an action")
	assert.Len(t, acts[0].Parameters, 3)

	// LoadByID
	result, err := action.LoadByID(context.TODO(), db, 0)
	require.NoError(t, err)
	assert.Nil(t, result)
	result, err = action.LoadByID(context.TODO(), db, acts[0].ID, action.LoadOptions.Default)
	require.NoError(t, err)
	assert.Equal(t, acts[0].Name, result.Name)
	assert.Len(t, result.Parameters, 3)
	assert.Len(t, result.Requirements, 1)
	assert.Len(t, result.Actions, 1)
	require.Len(t, result.Actions[0].Parameters, 1)
	assert.Equal(t, "echo \"test\"", result.Actions[0].Parameters[0].Value)

	// LoadAllByTypes
	results, err := action.LoadAllByTypes(context.TODO(), db, []string{sdk.PluginAction, sdk.BuiltinAction})
	require.NoError(t, err)
	lengthExistingBuiltinAndPlugin := len(results)

	// LoadAllTypeBuiltInOrPluginOrDefaultForGroupIDs
	results, err = action.LoadAllTypeBuiltInOrPluginOrDefaultForGroupIDs(context.TODO(), db, []int64{grp1.ID})
	require.NoError(t, err)
	assert.Len(t, results, lengthExistingBuiltinAndPlugin+1)

	// LoadAllTypeBuiltInOrPLoadAllTypeDefaultByGroupIDsluginOrDefaultForGroupIDs
	results, err = action.LoadAllTypeDefaultByGroupIDs(context.TODO(), db, []int64{grp2.ID}, action.LoadOptions.Default)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, acts[1].ID, results[0].ID)
	results, err = action.LoadAllTypeDefaultByGroupIDs(context.TODO(), db, []int64{grp1.ID, grp2.ID})
	require.NoError(t, err)
	require.Len(t, results, 2)
	sort.Slice(results, func(i, j int) bool { return results[i].ID < results[j].ID })
	assert.Equal(t, acts[0].ID, results[0].ID)
	assert.Equal(t, acts[1].ID, results[1].ID)

	// LoadAllByIDsWithTypeBuiltinOrPluginOrDefaultInGroupIDs
	results, err = action.LoadAllByIDsWithTypeBuiltinOrPluginOrDefaultInGroupIDs(context.TODO(), db, []int64{scriptAction.ID, acts[0].ID, acts[1].ID}, []int64{grp1.ID, grp2.ID})
	require.NoError(t, err)
	require.Len(t, results, 3)
	sort.Slice(results, func(i, j int) bool { return results[i].ID < results[j].ID })
	assert.Equal(t, scriptAction.ID, results[0].ID)
	assert.Equal(t, acts[0].ID, results[1].ID)
	assert.Equal(t, acts[1].ID, results[2].ID)

	// LoadByTypesAndName
	result, err = action.LoadByTypesAndName(context.TODO(), db, []string{sdk.DefaultAction}, "Action 0")
	require.NoError(t, err)
	assert.Nil(t, result)
	result, err = action.LoadByTypesAndName(context.TODO(), db, []string{sdk.DefaultAction}, acts[0].Name)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, acts[0].ID, result.ID)

	// LoadTypeDefaultByNameAndGroupID
	result, err = action.LoadTypeDefaultByNameAndGroupID(context.TODO(), db, acts[1].Name, grp1.ID)
	require.NoError(t, err)
	assert.Nil(t, result)
	result, err = action.LoadTypeDefaultByNameAndGroupID(context.TODO(), db, acts[1].Name, grp2.ID)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, acts[1].ID, result.ID)

	// LoadByIDs
	results, err = action.LoadAllByIDs(context.TODO(), db, []int64{acts[0].ID, acts[1].ID}, action.LoadOptions.Default)
	require.NoError(t, err)
	require.Len(t, results, 2)
	sort.Slice(results, func(i, j int) bool { return results[i].ID < results[j].ID })
	assert.Equal(t, acts[0].Name, results[0].Name)
	assert.Len(t, results[0].Parameters, 3)
	assert.Len(t, results[0].Requirements, 1)
	assert.Len(t, results[0].Actions, 1)
	require.Len(t, results[0].Actions[0].Parameters, 1)
	assert.Equal(t, "echo \"test\"", results[0].Actions[0].Parameters[0].Value)
	results, err = action.LoadAllByIDs(context.TODO(), db, []int64{acts[1].ID})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, acts[1].Name, results[0].Name)

	// Delete
	for i := range acts {
		require.NoError(t, action.Delete(db, &acts[i]), "No err should be returned when removing an action")
	}
}

func Test_RetrieveForGroupAndName(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	defer func() {
		assets.DeleteTestGroup(t, db, grp1)
	}()

	scriptAction := assets.GetBuiltinOrPluginActionByName(t, db, "Script")

	// Insert new action
	act := sdk.Action{
		GroupID: &grp1.ID,
		Type:    sdk.DefaultAction,
		Name:    sdk.RandomString(10),
	}
	require.NoError(t, action.Insert(db, &act), "No err should be returned when inserting an action")
	defer func() {
		assert.NoError(t, action.Delete(db, &act))
	}()

	// Retrieve builtin action
	result, err := action.RetrieveForGroupAndName(context.TODO(), db, nil, "Script")
	require.NoError(t, err)
	assert.Equal(t, scriptAction.ID, result.ID)

	// Retrieve default action
	result, err = action.RetrieveForGroupAndName(context.TODO(), db, grp1, act.Name)
	require.NoError(t, err)
	assert.Equal(t, act.ID, result.ID)
}

func Test_CheckChildrenForGroupIDsWithLoop(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	defer func() {
		assets.DeleteTestGroup(t, db, grp1)
	}()

	// Insert action with builtin child
	one := sdk.Action{
		GroupID: &grp1.ID,
		Type:    sdk.DefaultAction,
		Name:    sdk.RandomString(10),
	}
	require.NoError(t, action.Insert(db, &one), "No err should be returned when inserting an action")
	defer func() {
		assert.NoError(t, action.Delete(db, &one))
	}()

	// Insert action with default child
	two := sdk.Action{
		GroupID: &grp1.ID,
		Type:    sdk.DefaultAction,
		Name:    sdk.RandomString(10),
		Actions: []sdk.Action{
			{
				ID: one.ID,
			},
		},
	}
	require.NoError(t, action.Insert(db, &two), "No err should be returned when inserting an action")
	defer func() {
		assert.NoError(t, action.Delete(db, &two))
	}()

	// test valid use case
	assert.NoError(t, action.CheckChildrenForGroupIDsWithLoop(context.TODO(), db, &two, []int64{grp1.ID}))

	// test invalid recusive
	one.Actions = append(one.Actions, sdk.Action{
		ID: two.ID,
	})
	assert.NoError(t, action.CheckChildrenForGroupIDs(context.TODO(), db, &one, []int64{grp1.ID}))
	assert.Error(t, action.CheckChildrenForGroupIDsWithLoop(context.TODO(), db, &one, []int64{grp1.ID}))
}
