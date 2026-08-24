package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
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

	reqData := sdk.CreateUser{
		Username:     username,
		Fullname:     fullname,
		Email:        username + "." + fullname + "@localhost.local",
		Organization: "default",
	}

	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPost, uri, reqData)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var authUser sdk.AuthentifiedUser
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &authUser))
	require.Equal(t, username, authUser.Username)
	require.Equal(t, "default", authUser.Organization)
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
	assert.False(t, result.Disabled)

	// A disabled user must stay in the listing, flagged: this is what the UI and cdsctl
	// rely on to tell a neutralized account from a live one.
	other, _ := assets.InsertLambdaUser(t, db)
	other.Disabled = true
	require.NoError(t, user.Update(context.TODO(), db, other))

	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Assert on the raw payload: the json field is what both clients read
	var raw []map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &raw))
	var found bool
	for _, u := range raw {
		if u["id"] == other.ID {
			found = true
			assert.Equal(t, true, u["disabled"], "a disabled user must be flagged in the listing payload")
		}
	}
	require.True(t, found, "a disabled user must remain in the list of all users")
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

// Test_putUserHandler_disableUser checks that disabling a user actually locks it out
// right away: it is the whole point of the feature, as an alternative to deleting a user
// and having to clean its groups first.
func Test_putUserHandler_disableUser(t *testing.T) {
	api, db, _ := newTestAPI(t)

	assets.DeleteAdmins(t, db)

	lambda, jwtLambdaRaw := assets.InsertLambdaUser(t, db)
	admin, jwtAdminRaw := assets.InsertAdminUser(t, db)

	getMeURI := api.Router.GetRoute(http.MethodGet, api.getUserHandler, map[string]string{
		"permUsernamePublic": "me",
	})
	require.NotEmpty(t, getMeURI)
	putLambdaURI := api.Router.GetRoute(http.MethodPut, api.putUserHandler, map[string]string{
		"permUsernamePublic": lambda.Username,
	})
	require.NotEmpty(t, putLambdaURI)

	callGetMe := func(jwt string) int {
		req := assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodGet, getMeURI, nil)
		rec := httptest.NewRecorder()
		api.Router.Mux.ServeHTTP(rec, req)
		return rec.Code
	}
	putLambda := func(jwt string, disabled bool) *httptest.ResponseRecorder {
		req := assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodPut, putLambdaURI, sdk.AuthentifiedUser{
			Username: lambda.Username,
			Fullname: lambda.Fullname,
			Ring:     lambda.Ring,
			Disabled: disabled,
		})
		rec := httptest.NewRecorder()
		api.Router.Mux.ServeHTTP(rec, req)
		return rec
	}

	require.Equal(t, http.StatusOK, callGetMe(jwtLambdaRaw), "lambda user should be able to authenticate before being disabled")

	t.Run("A lambda user can't disable itself", func(t *testing.T) {
		rec := putLambda(jwtLambdaRaw, true)
		require.Equal(t, http.StatusOK, rec.Code)
		var modified sdk.AuthentifiedUser
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &modified))
		require.False(t, modified.Disabled, "only an admin can change the disabled flag")
	})

	putUser := func(jwt string, target *sdk.AuthentifiedUser, ring string, disabled bool) *httptest.ResponseRecorder {
		uri := api.Router.GetRoute(http.MethodPut, api.putUserHandler, map[string]string{
			"permUsernamePublic": target.Username,
		})
		require.NotEmpty(t, uri)
		req := assets.NewJWTAuthentifiedRequest(t, jwt, http.MethodPut, uri, sdk.AuthentifiedUser{
			Username: target.Username,
			Fullname: target.Fullname,
			Ring:     ring,
			Disabled: disabled,
		})
		rec := httptest.NewRecorder()
		api.Router.Mux.ServeHTTP(rec, req)
		return rec
	}

	// An admin must never end up disabled: it would have no way to get its access back.
	t.Run("An admin can't be disabled", func(t *testing.T) {
		otherAdmin, _ := assets.InsertAdminUser(t, db)

		rec := putUser(jwtAdminRaw, otherAdmin, otherAdmin.Ring, true)
		require.Equal(t, http.StatusForbidden, rec.Code, "an admin must not be able to disable another admin")

		rec = putUser(jwtAdminRaw, admin, admin.Ring, true)
		require.Equal(t, http.StatusForbidden, rec.Code, "an admin must not be able to lock itself out")
	})

	t.Run("A user can't be promoted admin and disabled at once", func(t *testing.T) {
		target, _ := assets.InsertLambdaUser(t, db)

		rec := putUser(jwtAdminRaw, target, sdk.UserRingAdmin, true)
		require.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("A disabled user can't be promoted admin while staying disabled", func(t *testing.T) {
		target, _ := assets.InsertLambdaUser(t, db)
		require.Equal(t, http.StatusOK, putUser(jwtAdminRaw, target, target.Ring, true).Code)

		rec := putUser(jwtAdminRaw, target, sdk.UserRingAdmin, true)
		require.Equal(t, http.StatusForbidden, rec.Code)

		// Enabling and promoting in the same request is fine
		rec = putUser(jwtAdminRaw, target, sdk.UserRingAdmin, false)
		require.Equal(t, http.StatusOK, rec.Code)
		var modified sdk.AuthentifiedUser
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &modified))
		require.Equal(t, sdk.UserRingAdmin, modified.Ring)
		require.False(t, modified.Disabled)
	})

	t.Run("An admin can disable a user and its sessions are revoked", func(t *testing.T) {
		rec := putLambda(jwtAdminRaw, true)
		require.Equal(t, http.StatusOK, rec.Code)
		var modified sdk.AuthentifiedUser
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &modified))
		require.True(t, modified.Disabled)

		// The session was revoked, so the JWT can no longer be resolved to a consumer
		require.Equal(t, http.StatusUnauthorized, callGetMe(jwtLambdaRaw))
	})

	t.Run("An admin can enable a user back", func(t *testing.T) {
		rec := putLambda(jwtAdminRaw, false)
		require.Equal(t, http.StatusOK, rec.Code)
		var modified sdk.AuthentifiedUser
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &modified))
		require.False(t, modified.Disabled)
	})

	// A session created before the user was disabled is revoked, but a session that
	// somehow survives must also be rejected: the check has to live in the middleware.
	t.Run("A disabled user is rejected even with a valid session", func(t *testing.T) {
		other, jwtOtherRaw := assets.InsertLambdaUser(t, db)
		require.Equal(t, http.StatusOK, callGetMe(jwtOtherRaw))

		other.Disabled = true
		require.NoError(t, user.Update(context.TODO(), db, other))

		require.Equal(t, http.StatusForbidden, callGetMe(jwtOtherRaw))
	})
}

func Test_deleteUserHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	assets.DeleteAdmins(t, db)

	initial1, jwtInitial1Raw := assets.InsertLambdaUser(t, db)
	initial2, _ := assets.InsertLambdaUser(t, db)
	initial3Group := &sdk.Group{Name: sdk.RandomString(10)}
	initial3, _ := assets.InsertLambdaUser(t, db, initial3Group)
	admin1, jwtAdmin1Raw := assets.InsertAdminUser(t, db)
	admin2, _ := assets.InsertAdminUser(t, db)

	cases := []struct {
		Name                 string
		JWT                  string
		TargetUsername       string
		ExpectedStatus       int
		ExpectedBodyContains string
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
			// The error should tell which group prevents the deletion
			Name:                 "A user can't be removed if last admin of a group",
			JWT:                  jwtAdmin1Raw,
			TargetUsername:       initial3.Username,
			ExpectedStatus:       http.StatusForbidden,
			ExpectedBodyContains: initial3Group.Name,
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
			if c.ExpectedBodyContains != "" {
				assert.Contains(t, rec.Body.String(), c.ExpectedBodyContains)
			}
		})
	}
}
