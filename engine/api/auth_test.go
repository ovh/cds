package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/authentication"

	"github.com/ovh/cds/engine/api/user"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test/assets"
)

func Test_postAuthSignoutHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	_, jwtRaw := assets.InsertLambdaUser(db)

	uri := api.Router.GetRoute(http.MethodGet, api.getUserHandler, map[string]string{
		"permUsernamePublic": "me",
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	uri = api.Router.GetRoute(http.MethodPost, api.postAuthSignoutHandler, nil)
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodPost, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	uri = api.Router.GetRoute(http.MethodGet, api.getUserHandler, map[string]string{
		"permUsernamePublic": "me",
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtRaw, http.MethodGet, uri, nil)
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 401, rec.Code)
}

func Test_postAuthSigninHandler_ShouldSuccessWithANewUser(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	uri := api.Router.GetRoute(http.MethodPost, api.postAuthSigninHandler, map[string]string{
		"consumerType": "futurama",
	})
	require.NotEmpty(t, uri)

	req := assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"username": "fry",
	})
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	u, err := user.LoadByUsername(context.TODO(), db, "fry", user.LoadOptions.WithContacts, user.LoadOptions.WithDeprecatedUser)
	require.NoError(t, err)
	require.NotNil(t, u)

	assert.Equal(t, "Philip J. Fry", u.Fullname)
	assert.Equal(t, "fry@planet-express.futurama", u.GetEmail())

	err = user.DeleteByID(db, u.ID)
	require.NoError(t, err)
}

func Test_postAuthSigninHandler_ShouldSuccessWithAKnownUser(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	uri := api.Router.GetRoute(http.MethodPost, api.postAuthSigninHandler, map[string]string{
		"consumerType": "futurama",
	})
	require.NotEmpty(t, uri)

	req := assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"username": "fry",
	})
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	// Call a second time

	req = assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"username": "fry",
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	u, err := user.LoadByUsername(context.TODO(), db, "fry", user.LoadOptions.WithContacts, user.LoadOptions.WithDeprecatedUser)
	require.NoError(t, err)
	require.NotNil(t, u)

	err = user.DeleteByID(db, u.ID)
	require.NoError(t, err)
}

func Test_postAuthSigninHandler_ShouldSuccessWithAKnownUserAndAnotherConsumerType(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	uri := api.Router.GetRoute(http.MethodPost, api.postAuthSigninHandler, map[string]string{
		"consumerType": "futurama",
	})
	require.NotEmpty(t, uri)

	req := assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"username": "fry",
	})
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	// Call a second time with another consumer type
	uri = api.Router.GetRoute(http.MethodPost, api.postAuthSigninHandler, map[string]string{
		"consumerType": "planet-express",
	})
	require.NotEmpty(t, uri)
	req = assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"username": "fry",
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	u, err := user.LoadByUsername(context.TODO(), db, "fry", user.LoadOptions.WithContacts, user.LoadOptions.WithDeprecatedUser)
	require.NoError(t, err)
	require.NotNil(t, u)

	// checks that there are 2 consumers now
	consumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerTest, u.ID)
	require.NoError(t, err)
	assert.Equal(t, sdk.ConsumerTest, consumer.Type)

	t.Logf("consumer %s: %+v", consumer.Type, consumer.Data)

	consumer, err = authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerTest2, u.ID)
	require.NoError(t, err)
	assert.Equal(t, sdk.ConsumerTest2, consumer.Type)

	t.Logf("consumer %s: %+v", consumer.Type, consumer.Data)

	// tear down
	err = user.DeleteByID(db, u.ID)
	require.NoError(t, err)
}

func Test_postAuthSigninHandler_ShouldSuccessWithAKnownUserAnotherConsumerTypeAndAnotherUsername(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	uri := api.Router.GetRoute(http.MethodPost, api.postAuthSigninHandler, map[string]string{
		"consumerType": "futurama",
	})
	require.NotEmpty(t, uri)

	req := assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"username": "fry",
	})
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	// Call a second time with another consumer type
	uri = api.Router.GetRoute(http.MethodPost, api.postAuthSigninHandler, map[string]string{
		"consumerType": "planet-express",
	})
	require.NotEmpty(t, uri)
	req = assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"username": "philip.fry",
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	u, err := user.LoadByUsername(context.TODO(), db, "fry", user.LoadOptions.WithContacts, user.LoadOptions.WithDeprecatedUser)
	require.NoError(t, err)
	require.NotNil(t, u)

	// checks that there are 2 consumers now
	consumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerTest, u.ID)
	require.NoError(t, err)
	assert.Equal(t, sdk.ConsumerTest, consumer.Type)

	t.Logf("consumer %s: %+v", consumer.Type, consumer.Data)

	consumer, err = authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerTest2, u.ID)
	require.NoError(t, err)
	assert.Equal(t, sdk.ConsumerTest2, consumer.Type)

	t.Logf("consumer %s: %+v", consumer.Type, consumer.Data)

	// tear down
	err = user.DeleteByID(db, u.ID)
	require.NoError(t, err)
}
