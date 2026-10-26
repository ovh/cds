package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getConsumersByUserHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, jwtRaw := assets.InsertLambdaUser(t, db)
	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	consumerOptions := builtin.NewConsumerOptions{
		Name:        sdk.RandomString(10),
		Description: sdk.RandomString(10),
		Duration:    0,
		Scopes:      sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeUser),
	}
	consumer, _, err := builtin.NewConsumer(context.TODO(), db, consumerOptions, localConsumer)
	require.NoError(t, err)

	uri := api.Router.GetRoute(http.MethodGet, api.getConsumersByUserHandler, map[string]string{
		"permUsername": u.Username,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var cs []sdk.AuthUserConsumer
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cs))
	require.Equal(t, 2, len(cs))
	require.True(t, slices.ContainsFunc(cs, func(c sdk.AuthUserConsumer) bool { return c.ID == localConsumer.ID }))
	require.True(t, slices.ContainsFunc(cs, func(c sdk.AuthUserConsumer) bool { return c.ID == consumer.ID }))

	uri = api.Router.GetRoute(http.MethodGet, api.getConsumersByUserHandler, map[string]string{
		"permUsername": "me",
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &cs))
	require.Len(t, cs, 2)
	require.True(t, slices.ContainsFunc(cs, func(c sdk.AuthUserConsumer) bool { return c.ID == localConsumer.ID }))
	require.True(t, slices.ContainsFunc(cs, func(c sdk.AuthUserConsumer) bool { return c.ID == consumer.ID }))
}

func Test_postConsumerByUserHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	g := assets.InsertGroup(t, db)
	u, jwtRaw := assets.InsertLambdaUser(t, db, g)
	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID)
	require.NoError(t, err)
	_, jwtRawAdmin := assets.InsertAdminUser(t, db)

	data := sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            sdk.RandomString(10),
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			GroupIDs:     []int64{g.ID},
			ScopeDetails: sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAccessToken),
		},
	}

	uri := api.Router.GetRoute(http.MethodPost, api.postConsumerByUserHandler, map[string]string{
		"permUsername": u.Username,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRawAdmin, http.MethodPost, uri, data)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 403, rec.Code)

	uri = api.Router.GetRoute(http.MethodPost, api.postConsumerByUserHandler, map[string]string{
		"permUsername": u.Username,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPost, uri, data)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 201, rec.Code)

	var created sdk.AuthConsumerCreateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	require.NotEmpty(t, created.Token)
	require.Equal(t, data.Name, created.Consumer.Name)
	require.Len(t, created.Consumer.AuthConsumerUser.GroupIDs, 1)
	require.Equal(t, g.ID, created.Consumer.AuthConsumerUser.GroupIDs[0])
	require.Len(t, created.Consumer.AuthConsumerUser.ScopeDetails, 1)
	require.Equal(t, sdk.AuthConsumerScopeAccessToken, created.Consumer.AuthConsumerUser.ScopeDetails[0].Scope)
	require.Equal(t, localConsumer.ID, *created.Consumer.ParentID)
}

// A non-admin cannot scope a consumer to a group they do not belong to.
// See docs/content/docs/concepts/authentication.md (Groups).
func Test_postConsumerByUserHandler_RejectForeignGroup(t *testing.T) {
	api, db, _ := newTestAPI(t)

	ownGroup := assets.InsertGroup(t, db)
	foreignGroup := assets.InsertGroup(t, db)
	u, jwtRaw := assets.InsertLambdaUser(t, db, ownGroup)

	data := sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            sdk.RandomString(10),
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			GroupIDs:     []int64{foreignGroup.ID},
			ScopeDetails: sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAccessToken),
		},
	}

	uri := api.Router.GetRoute(http.MethodPost, api.postConsumerByUserHandler, map[string]string{
		"permUsername": u.Username,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPost, uri, data)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)

	require.Equal(t, 400, rec.Code,
		"a user must not be able to mint a consumer for a group they are not a member of; got status %d body %s",
		rec.Code, rec.Body.String())
}

// A child of a group-restricted builtin token inherits the parent's groups, not
// all the user's groups. See docs/content/docs/concepts/authentication.md (Groups).
func Test_postConsumerByUserHandler_EmptyChildInheritsRestrictedParent(t *testing.T) {
	api, db, _ := newTestAPI(t)

	g1 := assets.InsertGroup(t, db)
	g2 := assets.InsertGroup(t, db)
	u, _ := assets.InsertLambdaUser(t, db, g1, g2)

	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	// Parent builtin token restricted to g1 only, with the AccessToken scope so
	// it can reach the consumer-creation route.
	parentOpts := builtin.NewConsumerOptions{
		Name:     sdk.RandomString(10),
		Duration: 0,
		GroupIDs: []int64{g1.ID},
		Scopes:   sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAccessToken),
	}
	parentConsumer, _, err := builtin.NewConsumer(context.TODO(), db, parentOpts, localConsumer)
	require.NoError(t, err)

	session, err := authentication.NewSession(context.TODO(), db, &parentConsumer.AuthConsumer, 5*time.Minute)
	require.NoError(t, err)
	parentJWT, err := authentication.NewSessionJWT(session, "")
	require.NoError(t, err)

	// Child created with no groups: it must inherit the parent's restriction (g1),
	// not resolve to all of the user's groups (g1 and g2).
	data := sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            sdk.RandomString(10),
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			GroupIDs:     []int64{},
			ScopeDetails: sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAccessToken),
		},
	}

	uri := api.Router.GetRoute(http.MethodPost, api.postConsumerByUserHandler, map[string]string{
		"permUsername": u.Username,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, parentJWT, http.MethodPost, uri, data)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 201, rec.Code, "body: %s", rec.Body.String())

	var created sdk.AuthConsumerCreateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	require.Len(t, created.Consumer.AuthConsumerUser.GroupIDs, 1,
		"a child of a group-restricted token must inherit that restriction, not widen to all the user's groups; body %s",
		rec.Body.String())
	require.Equal(t, g1.ID, created.Consumer.AuthConsumerUser.GroupIDs[0])
}

// A CDS admin on a first level consumer can scope a consumer to any group,
// including ones they do not belong to. See
// docs/content/docs/concepts/authentication.md (Groups).
func Test_postConsumerByUserHandler_AdminCanScopeToForeignGroup(t *testing.T) {
	api, db, _ := newTestAPI(t)

	adminUser, jwtAdmin := assets.InsertAdminUser(t, db)
	foreignGroup := assets.InsertGroup(t, db) // the admin is not a member of it

	data := sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            sdk.RandomString(10),
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			GroupIDs:     []int64{foreignGroup.ID},
			ScopeDetails: sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAccessToken),
		},
	}

	uri := api.Router.GetRoute(http.MethodPost, api.postConsumerByUserHandler, map[string]string{
		"permUsername": adminUser.Username,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtAdmin, http.MethodPost, uri, data)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 201, rec.Code,
		"an admin must be able to scope a consumer to a group they are not a member of; body %s", rec.Body.String())

	var created sdk.AuthConsumerCreateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	require.Len(t, created.Consumer.AuthConsumerUser.GroupIDs, 1)
	require.Equal(t, foreignGroup.ID, created.Consumer.AuthConsumerUser.GroupIDs[0])
}

// A child with no groups from an unrestricted parent stays a wildcard (all the
// user's groups). See docs/content/docs/concepts/authentication.md (Groups).
func Test_postConsumerByUserHandler_UnrestrictedParentKeepsAllGroups(t *testing.T) {
	api, db, _ := newTestAPI(t)

	g1 := assets.InsertGroup(t, db)
	g2 := assets.InsertGroup(t, db)
	u, jwtRaw := assets.InsertLambdaUser(t, db, g1, g2)

	// Parent is the user's local session: it carries no group restriction.
	data := sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            sdk.RandomString(10),
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			GroupIDs:     []int64{},
			ScopeDetails: sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAccessToken),
		},
	}

	uri := api.Router.GetRoute(http.MethodPost, api.postConsumerByUserHandler, map[string]string{
		"permUsername": u.Username,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPost, uri, data)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 201, rec.Code, "body: %s", rec.Body.String())

	var created sdk.AuthConsumerCreateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	// Empty stored group list means all the user's groups; the child must keep it.
	require.Len(t, created.Consumer.AuthConsumerUser.GroupIDs, 0,
		"an unrestricted parent's child with no groups must keep all the user's groups (empty), not be narrowed; body %s",
		rec.Body.String())
}

// A wildcard builtin parent (no groups) can mint a child restricted to a group
// the user belongs to; a group outside the user's memberships is rejected.
// See docs/content/docs/concepts/authentication.md (Groups).
func Test_postConsumerByUserHandler_WildcardBuiltinParentCanRestrictChild(t *testing.T) {
	api, db, _ := newTestAPI(t)

	g1 := assets.InsertGroup(t, db)
	g2 := assets.InsertGroup(t, db)
	u, _ := assets.InsertLambdaUser(t, db, g1, g2)

	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	// Parent builtin token with no group restriction (wildcard), AccessToken scope.
	parentOpts := builtin.NewConsumerOptions{
		Name:     sdk.RandomString(10),
		Duration: 0,
		Scopes:   sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAccessToken),
	}
	parentConsumer, _, err := builtin.NewConsumer(context.TODO(), db, parentOpts, localConsumer)
	require.NoError(t, err)
	require.Len(t, parentConsumer.AuthConsumerUser.GroupIDs, 0) // wildcard parent

	session, err := authentication.NewSession(context.TODO(), db, &parentConsumer.AuthConsumer, 5*time.Minute)
	require.NoError(t, err)
	parentJWT, err := authentication.NewSessionJWT(session, "")
	require.NoError(t, err)

	uri := api.Router.GetRoute(http.MethodPost, api.postConsumerByUserHandler, map[string]string{
		"permUsername": u.Username,
	})
	require.NotEmpty(t, uri)

	// Child restricted to g1, a group the user belongs to: allowed, child carries g1.
	okData := sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            sdk.RandomString(10),
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			GroupIDs:     []int64{g1.ID},
			ScopeDetails: sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAccessToken),
		},
	}
	req := assets.NewJWTAuthentifiedRequest(t, parentJWT, http.MethodPost, uri, okData)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 201, rec.Code, "body: %s", rec.Body.String())

	var created sdk.AuthConsumerCreateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	require.Len(t, created.Consumer.AuthConsumerUser.GroupIDs, 1)
	require.Equal(t, g1.ID, created.Consumer.AuthConsumerUser.GroupIDs[0])

	// Child restricted to a group the user is not a member of: rejected.
	foreignGroup := assets.InsertGroup(t, db)
	koData := sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            sdk.RandomString(10),
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			GroupIDs:     []int64{foreignGroup.ID},
			ScopeDetails: sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAccessToken),
		},
	}
	req = assets.NewJWTAuthentifiedRequest(t, parentJWT, http.MethodPost, uri, koData)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 400, rec.Code, "body: %s", rec.Body.String())
}

func Test_postAdminConsumerByUserHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	g := assets.InsertGroup(t, db)
	targetUser, jwtRawTarget := assets.InsertLambdaUser(t, db, g)
	_, jwtRawAdmin := assets.InsertAdminUser(t, db)

	data := sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            sdk.RandomString(10),
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			GroupIDs:     []int64{g.ID},
			ScopeDetails: sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAccessToken),
		},
	}

	uri := api.Router.GetRoute(http.MethodPost, api.postAdminConsumerByUserHandler, map[string]string{
		"permUsername": targetUser.Username,
	})
	require.NotEmpty(t, uri)

	// A non-admin user can't use the admin route, even for itself
	req := assets.NewJWTAuthentifiedRequest(t, jwtRawTarget, http.MethodPost, uri, data)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 403, rec.Code)

	// An admin can create a consumer on behalf of another user
	req = assets.NewJWTAuthentifiedRequest(t, jwtRawAdmin, http.MethodPost, uri, data)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 201, rec.Code)

	var created sdk.AuthConsumerCreateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	require.NotEmpty(t, created.Token)
	require.Equal(t, data.Name, created.Consumer.Name)
	require.Equal(t, sdk.ConsumerBuiltin, created.Consumer.Type)
	require.Equal(t, targetUser.ID, created.Consumer.AuthConsumerUser.AuthentifiedUserID)
	require.Nil(t, created.Consumer.ParentID, "an admin-created consumer is a root builtin consumer")
	require.Len(t, created.Consumer.AuthConsumerUser.GroupIDs, 1)
	require.Equal(t, g.ID, created.Consumer.AuthConsumerUser.GroupIDs[0])
	require.Len(t, created.Consumer.AuthConsumerUser.ScopeDetails, 1)
	require.Equal(t, sdk.AuthConsumerScopeAccessToken, created.Consumer.AuthConsumerUser.ScopeDetails[0].Scope)
}

func Test_deleteConsumerByUserHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, jwtRaw := assets.InsertLambdaUser(t, db)

	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)
	consumerOptions := builtin.NewConsumerOptions{
		Name:        sdk.RandomString(10),
		Description: sdk.RandomString(10),
		Duration:    0,
		Scopes:      sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeAccessToken),
	}
	newConsumer, _, err := builtin.NewConsumer(context.TODO(), db, consumerOptions, localConsumer)
	require.NoError(t, err)
	cs, err := authentication.LoadUserConsumersByUserID(context.TODO(), db, u.ID)
	require.NoError(t, err)
	require.Len(t, cs, 2)

	uri := api.Router.GetRoute(http.MethodDelete, api.deleteConsumerByUserHandler, map[string]string{
		"permUsername":   u.Username,
		"permConsumerID": newConsumer.ID,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodDelete, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	cs, err = authentication.LoadUserConsumersByUserID(context.TODO(), db, u.ID)
	require.NoError(t, err)
	require.Len(t, cs, 1)
}

func Test_postConsumerRegenByUserHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, jwtRaw := assets.InsertLambdaUser(t, db)
	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	// Test that we can't regen a no builtin consumer
	uri := api.Router.GetRoute(http.MethodPost, api.postConsumerRegenByUserHandler, map[string]string{
		"permUsername":   u.Username,
		"permConsumerID": localConsumer.ID,
	})
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPost, uri, sdk.AuthConsumerRegenRequest{})
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)

	consumerOptions := builtin.NewConsumerOptions{
		Name:        sdk.RandomString(10),
		Description: sdk.RandomString(10),
		Duration:    0,
		Scopes:      sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeUser, sdk.AuthConsumerScopeAccessToken),
	}

	builtinConsumer, signinToken1, err := builtin.NewConsumer(context.TODO(), db, consumerOptions, localConsumer)
	require.NoError(t, err)
	session, err := authentication.NewSession(context.TODO(), db, &builtinConsumer.AuthConsumer, 5*time.Minute)
	require.NoError(t, err, "cannot create session")
	jwt2, err := authentication.NewSessionJWT(session, "")
	require.NoError(t, err, "cannot create jwt")

	uri = api.Router.GetRoute(http.MethodGet, api.getUserHandler, map[string]string{
		"permUsernamePublic": "me",
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	req = assets.NewJWTAuthentifiedRequest(t, jwt2, http.MethodGet, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Wait 2 seconds before regen
	time.Sleep(2 * time.Second)

	uri = api.Router.GetRoute(http.MethodPost, api.postConsumerRegenByUserHandler, map[string]string{
		"permUsername":   u.Username,
		"permConsumerID": builtinConsumer.ID,
	})
	req = assets.NewJWTAuthentifiedRequest(t, jwt2, http.MethodPost, uri, sdk.AuthConsumerRegenRequest{
		RevokeSessions: true,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var response sdk.AuthConsumerCreateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

	t.Logf("%+v", response)

	session, err = authentication.NewSession(context.TODO(), db, &builtinConsumer.AuthConsumer, 5*time.Minute)
	require.NoError(t, err)
	jwt3, err := authentication.NewSessionJWT(session, "")
	require.NoError(t, err)

	uri = api.Router.GetRoute(http.MethodGet, api.getUserHandler, map[string]string{
		"permUsernamePublic": "me",
	})

	// The new token should be ok
	req = assets.NewJWTAuthentifiedRequest(t, jwt3, http.MethodGet, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// After the regen the old token should be invalidated because we choose to drop the sessions
	req = assets.NewJWTAuthentifiedRequest(t, jwt2, http.MethodGet, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	// the old signing token from the builtin consumer should be invalidated
	uri = api.Router.GetRoute(http.MethodPost, api.postAuthBuiltinSigninHandler, nil)
	req = assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"token": signinToken1,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)

	// the new signing token from the builtin consumer should be fine
	signinToken2 := response.Token
	uri = api.Router.GetRoute(http.MethodPost, api.postAuthBuiltinSigninHandler, nil)
	req = assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"token": signinToken2,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	t.Log("next use case...")
	time.Sleep(2 * time.Second)

	// Regen the latest token with an overlap duration
	uri = api.Router.GetRoute(http.MethodPost, api.postConsumerRegenByUserHandler, map[string]string{
		"permUsername":   u.Username,
		"permConsumerID": builtinConsumer.ID,
	})
	req = assets.NewJWTAuthentifiedRequest(t, jwt3, http.MethodPost, uri, sdk.AuthConsumerRegenRequest{
		RevokeSessions:  true,
		OverlapDuration: "4s", // short 4s overlap
		NewDuration:     1,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	t.Logf("here is the new consumer: %+v", response)

	signinToken3 := response.Token

	// Wait before using it
	time.Sleep(2 * time.Second)

	// the new signing token from the builtin consumer should be fine
	uri = api.Router.GetRoute(http.MethodPost, api.postAuthBuiltinSigninHandler, nil)
	req = assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"token": signinToken3,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// The old token should be ok too, because of the overlap duration
	uri = api.Router.GetRoute(http.MethodPost, api.postAuthBuiltinSigninHandler, nil)
	req = assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"token": signinToken2,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Now wait for the overlab duration to be over....
	time.Sleep(2 * time.Second)

	// Now, the old token should be rejected
	uri = api.Router.GetRoute(http.MethodPost, api.postAuthBuiltinSigninHandler, nil)
	req = assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"token": signinToken2,
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func Test_getSessionsByUserHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, jwtRaw := assets.InsertLambdaUser(t, db)
	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	consumerOptions := builtin.NewConsumerOptions{
		Name:   sdk.RandomString(10),
		Scopes: sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeUser),
	}
	consumer, _, err := builtin.NewConsumer(context.TODO(), db, consumerOptions, localConsumer)
	require.NoError(t, err)
	s2, err := authentication.NewSession(context.TODO(), db, &consumer.AuthConsumer, time.Second)
	require.NoError(t, err)
	s3, err := authentication.NewSession(context.TODO(), db, &consumer.AuthConsumer, time.Second)
	require.NoError(t, err)

	uri := api.Router.GetRoute(http.MethodGet, api.getSessionsByUserHandler, map[string]string{
		"permUsername": u.Username,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var ss []sdk.AuthSession
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &ss))
	require.Len(t, ss, 3)
	require.True(t, slices.ContainsFunc(ss, func(e sdk.AuthSession) bool { return e.ConsumerID == localConsumer.ID }))
	require.True(t, slices.ContainsFunc(ss, func(e sdk.AuthSession) bool { return e.ID == s2.ID }))
	require.True(t, slices.ContainsFunc(ss, func(e sdk.AuthSession) bool { return e.ID == s3.ID }))
}

func Test_deleteSessionByUserHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, jwtRaw := assets.InsertLambdaUser(t, db)
	localConsumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID,
		authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	consumerOptions := builtin.NewConsumerOptions{
		Name:   sdk.RandomString(10),
		Scopes: sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeUser),
	}
	consumer, _, err := builtin.NewConsumer(context.TODO(), db, consumerOptions, localConsumer)
	require.NoError(t, err)
	s2, err := authentication.NewSession(context.TODO(), db, &consumer.AuthConsumer, time.Second)
	require.NoError(t, err)

	ss, err := authentication.LoadSessionsByConsumerIDs(context.TODO(), db, []string{localConsumer.ID, consumer.ID})
	require.NoError(t, err)
	require.Len(t, ss, 2)

	uri := api.Router.GetRoute(http.MethodDelete, api.deleteSessionByUserHandler, map[string]string{
		"permUsername":  u.Username,
		"permSessionID": s2.ID,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodDelete, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	ss, err = authentication.LoadSessionsByConsumerIDs(context.TODO(), db, []string{localConsumer.ID, consumer.ID})
	require.NoError(t, err)
	require.Len(t, ss, 1)
}
