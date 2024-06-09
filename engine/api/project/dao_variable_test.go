package project_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DAOVariable(t *testing.T) {
	db, cache := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	u, _ := assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)

	v1 := &sdk.ProjectVariable{Name: "clear", Type: sdk.TextVariable, Value: "clear_value"}
	v2 := &sdk.ProjectVariable{Name: "secret", Type: sdk.SecretVariable, Value: "secret_value"}

	require.NoError(t, project.InsertVariable(db, proj.ID, v1, u))
	assert.Equal(t, "clear_value", v1.Value)

	require.NoError(t, project.InsertVariable(db, proj.ID, v2, u))
	assert.Equal(t, sdk.PasswordPlaceholder, v2.Value)

	vs, err := project.LoadAllVariables(context.TODO(), db, proj.ID)
	require.NoError(t, err)
	assert.Equal(t, "clear_value", vs[0].Value)
	assert.Equal(t, sdk.PasswordPlaceholder, vs[1].Value)

	vs, err = project.LoadAllVariablesWithDecryption(context.TODO(), db, proj.ID)
	require.NoError(t, err)
	assert.Equal(t, "clear_value", vs[0].Value)
	assert.Equal(t, "secret_value", vs[1].Value)

	require.NoError(t, project.UpdateVariable(db, proj.ID, &vs[1], &vs[1], u))

	v1, err = project.LoadVariable(db, proj.ID, "clear")
	require.NoError(t, err)
	assert.Equal(t, "clear_value", v1.Value)

	v2, err = project.LoadVariable(db, proj.ID, "secret")
	require.NoError(t, err)
	assert.Equal(t, sdk.PasswordPlaceholder, v2.Value)

	v2, err = project.LoadVariableWithDecryption(db, proj.ID, v2.ID, "secret")
	require.NoError(t, err)
	assert.Equal(t, "secret_value", v2.Value)

	require.NoError(t, project.DeleteVariable(db, proj.ID, v2, u))

	require.NoError(t, project.DeleteAllVariables(db, proj.ID))
}

func Test_DAOVariableAllProjects(t *testing.T) {
	db, cache := test.SetupPG(t)

	key1 := sdk.RandomString(10)
	proj1 := assets.InsertTestProject(t, db, cache, key1, key1)

	key2 := sdk.RandomString(10)
	proj2 := assets.InsertTestProject(t, db, cache, key2, key2)

	u, _ := assets.InsertLambdaUser(t, db, &proj1.ProjectGroups[0].Group)
	v1 := &sdk.ProjectVariable{Name: "clear", Type: sdk.TextVariable, Value: "clear_value1"}
	v2 := &sdk.ProjectVariable{Name: "secret", Type: sdk.SecretVariable, Value: "secret_value1"}
	v3 := &sdk.ProjectVariable{Name: "clear", Type: sdk.TextVariable, Value: "clear_value2"}
	v4 := &sdk.ProjectVariable{Name: "secret", Type: sdk.SecretVariable, Value: "secret_value2"}

	require.NoError(t, project.InsertVariable(db, proj1.ID, v1, u))
	require.NoError(t, project.InsertVariable(db, proj1.ID, v2, u))
	require.NoError(t, project.InsertVariable(db, proj2.ID, v3, u))
	require.NoError(t, project.InsertVariable(db, proj2.ID, v4, u))

	vars, err := project.LoadAllVariablesForProjectsWithDecryption(context.TODO(), db, []int64{proj1.ID, proj2.ID})
	require.NoError(t, err)

	require.Len(t, vars, 2)
	require.NotNil(t, vars[proj1.ID])
	require.NotNil(t, vars[proj2.ID])
	require.Len(t, vars[proj1.ID], 2)
	require.Len(t, vars[proj2.ID], 2)

	t.Logf("%+v", vars)
	for _, v := range vars[proj1.ID] {
		switch v.Type {
		case sdk.SecretVariable:
			require.Equal(t, "secret_value1", v.Value)
		default:
			require.Equal(t, "clear_value1", v.Value)
		}
	}
	for _, v := range vars[proj2.ID] {
		switch v.Type {
		case sdk.SecretVariable:
			require.Equal(t, "secret_value2", v.Value)
		default:
			require.Equal(t, "clear_value2", v.Value)
		}
	}

}
