package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func insertTestVariableSetWithItems(t *testing.T, db gorpmapper.SqlExecutorWithTx, pKey, name string) sdk.ProjectVariableSet {
	vs := sdk.ProjectVariableSet{ProjectKey: pKey, Name: name}
	require.NoError(t, project.InsertVariableSet(context.TODO(), db, &vs))
	text := sdk.ProjectVariableSetItem{ProjectVariableSetID: vs.ID, Name: "MY_TEXT", Type: sdk.ProjectVariableTypeString, Value: "text-value"}
	require.NoError(t, project.InsertVariableSetItemText(context.TODO(), db, &text))
	secret := sdk.ProjectVariableSetItem{ProjectVariableSetID: vs.ID, Name: "MY_SECRET", Type: sdk.ProjectVariableTypeSecret, Value: "secret-value"}
	require.NoError(t, project.InsertVariableSetItemSecret(context.TODO(), db, &secret))
	return vs
}

func postAdminCopyVariableSet(t *testing.T, api *API, jwt, pKey, vsName string, req sdk.CopyVariableSetToProject) *httptest.ResponseRecorder {
	vars := map[string]string{"projectKey": pKey, "variableSetName": vsName}
	uri := api.Router.GetRoute(http.MethodPost, api.postAdminCopyProjectVariableSetHandler, vars)
	require.NotEmpty(t, uri)
	r := assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodPost, uri, req)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, r)
	return rec
}

func requireVariableSetItemValues(t *testing.T, db gorpmapper.SqlExecutorWithTx, vsID string) map[string]sdk.ProjectVariableSetItem {
	items, err := project.LoadVariableSetAllItem(context.TODO(), db, vsID, gorpmapper.GetAllOptions.WithDecryption)
	require.NoError(t, err)
	require.Len(t, items, 2)
	byName := make(map[string]sdk.ProjectVariableSetItem, len(items))
	for _, it := range items {
		byName[it.Name] = it
	}
	require.Equal(t, "text-value", byName["MY_TEXT"].Value)
	require.Equal(t, sdk.ProjectVariableTypeString, byName["MY_TEXT"].Type)
	require.Equal(t, "secret-value", byName["MY_SECRET"].Value)
	require.Equal(t, sdk.ProjectVariableTypeSecret, byName["MY_SECRET"].Type)
	return byName
}

func Test_postAdminCopyProjectVariableSetHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, jwtAdmin := assets.InsertAdminUser(t, db)
	src := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	dst := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	srcVS := insertTestVariableSetWithItems(t, db, src.Key, "vs-1")

	rec := postAdminCopyVariableSet(t, api, jwtAdmin, src.Key, srcVS.Name, sdk.CopyVariableSetToProject{TargetProjectKey: dst.Key})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var copied sdk.ProjectVariableSet
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &copied))
	require.Equal(t, dst.Key, copied.ProjectKey)
	require.Equal(t, srcVS.Name, copied.Name)
	require.NotEqual(t, srcVS.ID, copied.ID)
	require.Len(t, copied.Items, 2)
	for _, it := range copied.Items {
		if it.Type == sdk.ProjectVariableTypeSecret {
			require.Equal(t, sdk.PasswordPlaceholder, it.Value)
		} else {
			require.Equal(t, "text-value", it.Value)
		}
	}

	// Copied items are new rows bound to the new variable set, readable with a valid signature and decryptable.
	dstVS, err := project.LoadVariableSetByName(context.TODO(), db, dst.Key, srcVS.Name)
	require.NoError(t, err)
	require.Equal(t, copied.ID, dstVS.ID)
	dstItems := requireVariableSetItemValues(t, db, dstVS.ID)

	// The source variable set is left untouched.
	srcVSDB, err := project.LoadVariableSetByName(context.TODO(), db, src.Key, srcVS.Name)
	require.NoError(t, err)
	require.Equal(t, srcVS.ID, srcVSDB.ID)
	srcItems := requireVariableSetItemValues(t, db, srcVS.ID)

	for name, it := range dstItems {
		require.Equal(t, dstVS.ID, it.ProjectVariableSetID)
		require.NotEqual(t, srcItems[name].ID, it.ID)
	}
}

func Test_postAdminCopyProjectVariableSetHandler_WithNewName(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, jwtAdmin := assets.InsertAdminUser(t, db)
	src := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	dst := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	srcVS := insertTestVariableSetWithItems(t, db, src.Key, "vs-1")

	// Copy into another project under a new name.
	rec := postAdminCopyVariableSet(t, api, jwtAdmin, src.Key, srcVS.Name, sdk.CopyVariableSetToProject{TargetProjectKey: dst.Key, NewName: "vs-renamed"})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	dstVS, err := project.LoadVariableSetByName(context.TODO(), db, dst.Key, "vs-renamed")
	require.NoError(t, err)
	requireVariableSetItemValues(t, db, dstVS.ID)
	_, err = project.LoadVariableSetByName(context.TODO(), db, dst.Key, srcVS.Name)
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound))

	// Copy inside the same project under a new name.
	rec = postAdminCopyVariableSet(t, api, jwtAdmin, src.Key, srcVS.Name, sdk.CopyVariableSetToProject{TargetProjectKey: src.Key, NewName: "vs-duplicated"})
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	dupVS, err := project.LoadVariableSetByName(context.TODO(), db, src.Key, "vs-duplicated")
	require.NoError(t, err)
	requireVariableSetItemValues(t, db, dupVS.ID)
	requireVariableSetItemValues(t, db, srcVS.ID)
}

func Test_postAdminCopyProjectVariableSetHandler_Errors(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, jwtAdmin := assets.InsertAdminUser(t, db)
	_, jwtLambda := assets.InsertLambdaUser(t, db)
	src := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	dst := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	srcVS := insertTestVariableSetWithItems(t, db, src.Key, "vs-1")

	// An empty variable set with the same name already exists in the target project.
	conflictVS := sdk.ProjectVariableSet{ProjectKey: dst.Key, Name: "vs-existing"}
	require.NoError(t, project.InsertVariableSet(context.TODO(), db, &conflictVS))

	tests := []struct {
		name         string
		jwt          string
		vsName       string
		req          sdk.CopyVariableSetToProject
		expectedCode int
	}{
		{
			name:         "same project and same name",
			jwt:          jwtAdmin,
			vsName:       srcVS.Name,
			req:          sdk.CopyVariableSetToProject{TargetProjectKey: src.Key},
			expectedCode: http.StatusConflict,
		},
		{
			name:         "name already used in target project",
			jwt:          jwtAdmin,
			vsName:       srcVS.Name,
			req:          sdk.CopyVariableSetToProject{TargetProjectKey: dst.Key, NewName: conflictVS.Name},
			expectedCode: http.StatusConflict,
		},
		{
			name:         "unknown target project",
			jwt:          jwtAdmin,
			vsName:       srcVS.Name,
			req:          sdk.CopyVariableSetToProject{TargetProjectKey: sdk.RandomString(10)},
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "unknown source variable set",
			jwt:          jwtAdmin,
			vsName:       "unknown",
			req:          sdk.CopyVariableSetToProject{TargetProjectKey: dst.Key},
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "missing target project",
			jwt:          jwtAdmin,
			vsName:       srcVS.Name,
			req:          sdk.CopyVariableSetToProject{},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "invalid new name",
			jwt:          jwtAdmin,
			vsName:       srcVS.Name,
			req:          sdk.CopyVariableSetToProject{TargetProjectKey: dst.Key, NewName: "invalid name!"},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "non admin user",
			jwt:          jwtLambda,
			vsName:       srcVS.Name,
			req:          sdk.CopyVariableSetToProject{TargetProjectKey: dst.Key},
			expectedCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := postAdminCopyVariableSet(t, api, tt.jwt, src.Key, tt.vsName, tt.req)
			require.Equal(t, tt.expectedCode, rec.Code, rec.Body.String())
		})
	}

	// Nothing was created by the failed calls.
	vss, err := project.LoadVariableSetsByProject(context.TODO(), db, dst.Key)
	require.NoError(t, err)
	require.Len(t, vss, 1)
	items, err := project.LoadVariableSetAllItem(context.TODO(), db, conflictVS.ID)
	require.NoError(t, err)
	require.Empty(t, items)
}
