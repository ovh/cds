package environment_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/environment"

	"github.com/ovh/cds/sdk"

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

	env := sdk.Environment{
		Name:      "test",
		ProjectID: proj.ID,
	}

	require.NoError(t, environment.InsertEnvironment(db, &env))
	v1 := &sdk.EnvironmentVariable{Name: "clear", Type: sdk.TextVariable, Value: "clear_value"}
	v2 := &sdk.EnvironmentVariable{Name: "secret", Type: sdk.SecretVariable, Value: "secret_value"}

	require.NoError(t, environment.InsertVariable(db, env.ID, v1, u))
	assert.Equal(t, "clear_value", v1.Value)

	require.NoError(t, environment.InsertVariable(db, env.ID, v2, u))
	assert.Equal(t, sdk.PasswordPlaceholder, v2.Value)

	vs, err := environment.LoadAllVariables(db, env.ID)
	require.NoError(t, err)
	assert.Equal(t, "clear_value", vs[0].Value)
	assert.Equal(t, sdk.PasswordPlaceholder, vs[1].Value)

	vs, err = environment.LoadAllVariablesWithDecryption(db, env.ID)
	require.NoError(t, err)
	assert.Equal(t, "clear_value", vs[0].Value)
	assert.Equal(t, "secret_value", vs[1].Value)

	require.NoError(t, environment.UpdateVariable(db, env.ID, &vs[1], &vs[1], u))

	v1, err = environment.LoadVariable(db, env.ID, "clear")
	require.NoError(t, err)
	assert.Equal(t, "clear_value", v1.Value)

	v2, err = environment.LoadVariable(db, env.ID, "secret")
	require.NoError(t, err)
	assert.Equal(t, sdk.PasswordPlaceholder, v2.Value)

	v2, err = environment.LoadVariableWithDecryption(db, env.ID, v2.ID, "secret")
	require.NoError(t, err)
	assert.Equal(t, "secret_value", v2.Value)

	require.NoError(t, environment.DeleteVariable(db, env.ID, v2, u))

	require.NoError(t, environment.DeleteAllVariables(db, env.ID))
}

func Test_DAOAllVarsFromEnvs(t *testing.T) {
	db, cache := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	u, _ := assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)

	env1 := sdk.Environment{Name: "test1", ProjectID: proj.ID}
	env2 := sdk.Environment{Name: "test2", ProjectID: proj.ID}

	require.NoError(t, environment.InsertEnvironment(db, &env1))
	require.NoError(t, environment.InsertEnvironment(db, &env2))
	v1 := &sdk.EnvironmentVariable{Name: "clear", Type: sdk.TextVariable, Value: "clear_value1"}
	v2 := &sdk.EnvironmentVariable{Name: "secret", Type: sdk.SecretVariable, Value: "secret_value1"}
	v3 := &sdk.EnvironmentVariable{Name: "clear", Type: sdk.TextVariable, Value: "clear_value2"}
	v4 := &sdk.EnvironmentVariable{Name: "secret", Type: sdk.SecretVariable, Value: "secret_value2"}

	require.NoError(t, environment.InsertVariable(db, env1.ID, v1, u))
	require.NoError(t, environment.InsertVariable(db, env1.ID, v2, u))
	require.NoError(t, environment.InsertVariable(db, env2.ID, v3, u))
	require.NoError(t, environment.InsertVariable(db, env2.ID, v4, u))

	vars, err := environment.LoadAllVariablesForEnvsWithDecryption(context.TODO(), db, []int64{env1.ID, env2.ID})
	require.NoError(t, err)

	require.Len(t, vars, 2)
	require.Len(t, vars, 2)
	require.NotNil(t, vars[env1.ID])
	require.NotNil(t, vars[env2.ID])

	require.Len(t, vars[env1.ID], 2)
	require.Len(t, vars[env2.ID], 2)

	for _, v := range vars[env1.ID] {
		switch v.Type {
		case sdk.SecretVariable:
			require.Equal(t, "secret_value1", v.Value)
		default:
			require.Equal(t, "clear_value1", v.Value)
		}
	}
	for _, v := range vars[env2.ID] {
		switch v.Type {
		case sdk.SecretVariable:
			require.Equal(t, "secret_value2", v.Value)
		default:
			require.Equal(t, "clear_value2", v.Value)
		}
	}

}
