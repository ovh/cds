package project_test

import (
	"testing"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/api/project"
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

	v1 := &sdk.Variable{Name: "clear", Type: sdk.TextVariable, Value: "clear_value"}
	v2 := &sdk.Variable{Name: "secret", Type: sdk.SecretVariable, Value: "secret_value"}

	require.NoError(t, project.InsertVariable(db, proj.ID, v1, u))
	assert.Equal(t, "clear_value", v1.Value)

	require.NoError(t, project.InsertVariable(db, proj.ID, v2, u))
	assert.Equal(t, sdk.PasswordPlaceholder, v2.Value)

	vs, err := project.LoadAllVariables(db, proj.ID)
	require.NoError(t, err)
	assert.Equal(t, "clear_value", vs[0].Value)
	assert.Equal(t, sdk.PasswordPlaceholder, vs[1].Value)

	vs, err = project.LoadAllVariablesWithDecrytion(db, proj.ID)
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
