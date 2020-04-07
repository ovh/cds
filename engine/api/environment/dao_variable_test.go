package environment_test

import (
	"testing"

	"github.com/ovh/cds/engine/api/environment"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DAOVariable(t *testing.T) {
	db, cache, end := test.SetupPG(t)
	defer end()

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	u, _ := assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)

	env := sdk.Environment{
		Name:      "test",
		ProjectID: proj.ID,
	}

	require.NoError(t, environment.InsertEnvironment(db, &env))
	v1 := &sdk.Variable{Name: "clear", Type: sdk.TextVariable, Value: "clear_value"}
	v2 := &sdk.Variable{Name: "secret", Type: sdk.SecretVariable, Value: "secret_value"}

	require.NoError(t, environment.InsertVariable(db, env.ID, v1, u))
	assert.Equal(t, "clear_value", v1.Value)

	require.NoError(t, environment.InsertVariable(db, env.ID, v2, u))
	assert.Equal(t, sdk.PasswordPlaceholder, v2.Value)

	vs, err := environment.LoadAllVariables(db, env.ID)
	require.NoError(t, err)
	assert.Equal(t, "clear_value", vs[0].Value)
	assert.Equal(t, sdk.PasswordPlaceholder, vs[1].Value)

	vs, err = environment.LoadAllVariablesWithDecrytion(db, env.ID)
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
