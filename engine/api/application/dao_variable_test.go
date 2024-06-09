package application_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DAOVariable(t *testing.T) {
	db, cache := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	app := sdk.Application{
		Name: "my-app",
	}

	u, _ := assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)

	require.NoError(t, application.Insert(db, *proj, &app))

	v1 := &sdk.ApplicationVariable{Name: "clear", Type: sdk.TextVariable, Value: "clear_value"}
	v2 := &sdk.ApplicationVariable{Name: "secret", Type: sdk.SecretVariable, Value: "secret_value"}

	require.NoError(t, application.InsertVariable(db, app.ID, v1, u))
	assert.Equal(t, "clear_value", v1.Value)

	require.NoError(t, application.InsertVariable(db, app.ID, v2, u))
	assert.Equal(t, sdk.PasswordPlaceholder, v2.Value)

	vs, err := application.LoadAllVariables(context.TODO(), db, app.ID)
	require.NoError(t, err)
	assert.Equal(t, "clear_value", vs[0].Value)
	assert.Equal(t, sdk.PasswordPlaceholder, vs[1].Value)

	vs, err = application.LoadAllVariablesWithDecryption(context.TODO(), db, app.ID)
	require.NoError(t, err)
	assert.Equal(t, "clear_value", vs[0].Value)
	assert.Equal(t, "secret_value", vs[1].Value)

	require.NoError(t, application.UpdateVariable(db, app.ID, &vs[1], &vs[1], u))

	v1, err = application.LoadVariable(context.TODO(), db, app.ID, "clear")
	require.NoError(t, err)
	assert.Equal(t, "clear_value", v1.Value)

	v2, err = application.LoadVariable(context.TODO(), db, app.ID, "secret")
	require.NoError(t, err)
	assert.Equal(t, sdk.PasswordPlaceholder, v2.Value)

	v2, err = application.LoadVariableWithDecryption(context.TODO(), db, app.ID, v2.ID, "secret")
	require.NoError(t, err)
	assert.Equal(t, "secret_value", v2.Value)

	require.NoError(t, application.DeleteVariable(db, app.ID, v2, u))

	require.NoError(t, application.DeleteAllVariables(db, app.ID))

}

func Test_DAOAllVarsAllProjects(t *testing.T) {
	db, cache := test.SetupPG(t)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)
	u, _ := assets.InsertLambdaUser(t, db, &proj.ProjectGroups[0].Group)

	app1 := sdk.Application{
		Name: "my-app",
	}
	require.NoError(t, application.Insert(db, *proj, &app1))
	app2 := sdk.Application{
		Name: "my-app2",
	}
	require.NoError(t, application.Insert(db, *proj, &app2))

	v1 := &sdk.ApplicationVariable{Name: "clear", Type: sdk.TextVariable, Value: "clear_value1"}
	v2 := &sdk.ApplicationVariable{Name: "secret", Type: sdk.SecretVariable, Value: "secret_value1"}
	v3 := &sdk.ApplicationVariable{Name: "clear", Type: sdk.TextVariable, Value: "clear_value2"}
	v4 := &sdk.ApplicationVariable{Name: "secret", Type: sdk.SecretVariable, Value: "secret_value2"}

	require.NoError(t, application.InsertVariable(db, app1.ID, v1, u))
	require.NoError(t, application.InsertVariable(db, app1.ID, v2, u))
	require.NoError(t, application.InsertVariable(db, app2.ID, v3, u))
	require.NoError(t, application.InsertVariable(db, app2.ID, v4, u))

	vars, err := application.LoadAllVariablesForAppsWithDecryption(context.TODO(), db, []int64{app1.ID, app2.ID})
	require.NoError(t, err)

	require.Len(t, vars, 2)
	require.NotNil(t, vars[app1.ID])
	require.NotNil(t, vars[app2.ID])
	require.Len(t, vars[app1.ID], 2)
	require.Len(t, vars[app2.ID], 2)

	for _, v := range vars[app1.ID] {
		switch v.Type {
		case sdk.SecretVariable:
			require.Equal(t, "secret_value1", v.Value)
		default:
			require.Equal(t, "clear_value1", v.Value)
		}
	}
	for _, v := range vars[app2.ID] {
		switch v.Type {
		case sdk.SecretVariable:
			require.Equal(t, "secret_value2", v.Value)
		default:
			require.Equal(t, "clear_value2", v.Value)
		}
	}
}
