package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI_postUserHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, jwtRaw := assets.InsertAdminUser(t, db)

	uri := api.Router.GetRoute(http.MethodPost, api.postUserHandler, nil)
	require.NotEmpty(t, uri)

	username := "lambda-" + sdk.RandomString(10)
	fullname := "lambda-" + sdk.RandomString(10)

	reqData := sdk.AuthConsumerSigninRequest{
		"username": username,
		"fullname": fullname,
		"email":    username + "." + fullname + "@localhost.local",
	}

	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPost, uri, reqData)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var authUser sdk.AuthentifiedUser
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &authUser))
	t.Logf("authUser: %v", authUser)
	require.Equal(t, username, authUser.Username)
}

func Test_getUsersHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	expected, jwtRaw := assets.InsertLambdaUser(t, db)

	uri := api.Router.GetRoute(http.MethodGet, api.getUsersHandler, nil)
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var us []sdk.AuthentifiedUser
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &us))
	require.True(t, len(us) >= 1)

	var result *sdk.AuthentifiedUser
	for _, u := range us {
		if expected.ID == u.ID {
			result = &u
			break
		}
	}
	require.NotNil(t, result, "user should be in the list of all users")
	assert.Equal(t, expected.Username, result.Username)
}

func Test_getUserHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	expected, jwtRaw := assets.InsertLambdaUser(t, db)

	uri := api.Router.GetRoute(http.MethodGet, api.getUserHandler, map[string]string{
		"permUsernamePublic": expected.Username,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var u sdk.AuthentifiedUser
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &u))
	require.Equal(t, expected.ID, u.ID)
}

func Test_putUserHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	assets.DeleteAdmins(t, db)

	initial, jwtInitialRaw := assets.InsertLambdaUser(t, db)
	initialNewFullname := sdk.RandomString(10)
	admin1, jwtAdmin1Raw := assets.InsertAdminUser(t, db)
	admin2, jwtAdmin2Raw := assets.InsertAdminUser(t, db)

	cases := []struct {
		Name           string
		JWT            string
		TargetUsername string
		Data           sdk.AuthentifiedUser
		Expected       sdk.AuthentifiedUser
		ExpectedStatus int
	}{
		{
			Name:           "A lambda user can change fullname",
			JWT:            jwtInitialRaw,
			TargetUsername: initial.Username,
			Data: sdk.AuthentifiedUser{
				Username: initial.Username,
				Fullname: initialNewFullname,
				Ring:     initial.Ring,
			},
			Expected: sdk.AuthentifiedUser{
				Username:     initial.Username,
				Fullname:     initialNewFullname,
				Ring:         initial.Ring,
				Organization: "default",
			},
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "A lambda user can't change username",
			JWT:            jwtInitialRaw,
			TargetUsername: initial.Username,
			Data: sdk.AuthentifiedUser{
				Username: sdk.RandomString(10),
				Fullname: initialNewFullname,
				Ring:     initial.Ring,
			},
			Expected: sdk.AuthentifiedUser{
				Username:     initial.Username,
				Fullname:     initialNewFullname,
				Ring:         initial.Ring,
				Organization: "default",
			},
			ExpectedStatus: http.StatusForbidden,
		},
		{
			Name:           "A lambda user can't change its ring",
			JWT:            jwtInitialRaw,
			TargetUsername: initial.Username,
			Data: sdk.AuthentifiedUser{
				Username: initial.Username,
				Fullname: initialNewFullname,
				Ring:     sdk.UserRingAdmin,
			},
			Expected: sdk.AuthentifiedUser{
				Username:     initial.Username,
				Fullname:     initialNewFullname,
				Ring:         initial.Ring,
				Organization: "default",
			},
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "A admin user can change the ring of a user",
			JWT:            jwtAdmin1Raw,
			TargetUsername: initial.Username,
			Data: sdk.AuthentifiedUser{
				Username: initial.Username,
				Fullname: initialNewFullname,
				Ring:     sdk.UserRingMaintainer,
			},
			Expected: sdk.AuthentifiedUser{
				Username:     initial.Username,
				Fullname:     initialNewFullname,
				Ring:         sdk.UserRingMaintainer,
				Organization: "default",
			},
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "A admin can change its ring",
			JWT:            jwtAdmin1Raw,
			TargetUsername: admin1.Username,
			Data: sdk.AuthentifiedUser{
				Username: admin1.Username,
				Fullname: admin1.Fullname,
				Ring:     sdk.UserRingMaintainer,
			},
			Expected: sdk.AuthentifiedUser{
				Username:     admin1.Username,
				Fullname:     admin1.Fullname,
				Ring:         sdk.UserRingMaintainer,
				Organization: "default",
			},
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "A admin can't change its ring if last admin",
			JWT:            jwtAdmin2Raw,
			TargetUsername: admin2.Username,
			Data: sdk.AuthentifiedUser{
				Username:     admin2.Username,
				Fullname:     admin2.Fullname,
				Ring:         sdk.UserRingMaintainer,
				Organization: "default",
			},
			ExpectedStatus: http.StatusForbidden,
		},
		{
			Name:           "A lambda user can't change its organization",
			JWT:            jwtInitialRaw,
			TargetUsername: initial.Username,
			Data: sdk.AuthentifiedUser{
				Username:     initial.Username,
				Fullname:     initialNewFullname,
				Ring:         sdk.UserRingMaintainer,
				Organization: "my-org",
			},
			Expected: sdk.AuthentifiedUser{
				Username:     initial.Username,
				Fullname:     initialNewFullname,
				Ring:         sdk.UserRingMaintainer,
				Organization: "default",
			},
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "A admin user can't change user organization",
			JWT:            jwtAdmin2Raw,
			TargetUsername: initial.Username,
			Data: sdk.AuthentifiedUser{
				Username:     initial.Username,
				Fullname:     initialNewFullname,
				Ring:         sdk.UserRingMaintainer,
				Organization: "my-other-org",
			},
			ExpectedStatus: http.StatusForbidden,
		},
		{
			Name:           "A admin user can change username",
			JWT:            jwtAdmin2Raw,
			TargetUsername: initial.Username,
			Data: sdk.AuthentifiedUser{
				Username: initial.Username + ".updated",
				Fullname: initialNewFullname,
				Ring:     sdk.UserRingMaintainer,
			},
			Expected: sdk.AuthentifiedUser{
				Username:     initial.Username + ".updated",
				Fullname:     initialNewFullname,
				Ring:         sdk.UserRingMaintainer,
				Organization: "default",
			},
			ExpectedStatus: http.StatusOK,
		},
	}

	o := sdk.Organization{Name: "my-org"}
	require.NoError(t, organization.Insert(context.TODO(), db, &o))
	o2 := sdk.Organization{Name: "my-other-org"}
	require.NoError(t, organization.Insert(context.TODO(), db, &o2))

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			uri := api.Router.GetRoute(http.MethodPut, api.putUserHandler, map[string]string{
				"permUsernamePublic": c.TargetUsername,
			})
			require.NotEmpty(t, uri)

			req := assets.NewJWTAuthentifiedRequest(t, c.JWT, http.MethodPut, uri, c.Data)
			rec := httptest.NewRecorder()
			api.Router.Mux.ServeHTTP(rec, req)
			require.Equal(t, c.ExpectedStatus, rec.Code)

			if rec.Code != http.StatusOK {
				return
			}

			var modified sdk.AuthentifiedUser
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &modified))
			assert.Equal(t, c.Expected.Username, modified.Username)
			assert.Equal(t, c.Expected.Fullname, modified.Fullname)
			assert.Equal(t, c.Expected.Ring, modified.Ring)
			assert.Equal(t, c.Expected.Organization, modified.Organization)
		})
	}
}

func Test_deleteUserHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	assets.DeleteAdmins(t, db)

	initial1, jwtInitial1Raw := assets.InsertLambdaUser(t, db)
	initial2, _ := assets.InsertLambdaUser(t, db)
	initial3, _ := assets.InsertLambdaUser(t, db, &sdk.Group{Name: sdk.RandomString(10)})
	admin1, jwtAdmin1Raw := assets.InsertAdminUser(t, db)
	admin2, _ := assets.InsertAdminUser(t, db)

	cases := []struct {
		Name           string
		JWT            string
		TargetUsername string
		ExpectedStatus int
	}{
		{
			Name:           "A lambda user can delete himself",
			JWT:            jwtInitial1Raw,
			TargetUsername: initial1.Username,
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "A admin user can remove a user",
			JWT:            jwtAdmin1Raw,
			TargetUsername: initial2.Username,
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "A admin can remove another admin",
			JWT:            jwtAdmin1Raw,
			TargetUsername: admin2.Username,
			ExpectedStatus: http.StatusOK,
		},
		{
			Name:           "A admin can't remove himself if last admin",
			JWT:            jwtAdmin1Raw,
			TargetUsername: admin1.Username,
			ExpectedStatus: http.StatusForbidden,
		},
		{
			Name:           "A user can be removed if last admin of a group",
			JWT:            jwtAdmin1Raw,
			TargetUsername: initial3.Username,
			ExpectedStatus: http.StatusForbidden,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			uri := api.Router.GetRoute(http.MethodDelete, api.deleteUserHandler, map[string]string{
				"permUsernamePublic": c.TargetUsername,
			})
			require.NotEmpty(t, uri)

			req := assets.NewJWTAuthentifiedRequest(t, c.JWT, http.MethodDelete, uri, nil)
			rec := httptest.NewRecorder()
			api.Router.Mux.ServeHTTP(rec, req)
			assert.Equal(t, c.ExpectedStatus, rec.Code)
		})
	}
}
