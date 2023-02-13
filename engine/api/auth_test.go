package api

import (
	"context"
	"encoding/json"
	corpssodriver "github.com/ovh/cds/engine/api/driver/corpsso"
	"github.com/ovh/cds/engine/api/organization"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/corpsso"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gpg"
)

func Test_postAuthSignoutHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, jwtRaw := assets.InsertLambdaUser(t, db)

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
	api, db, _ := newTestAPI(t)
	api.Config.Auth.AllowedOrganizations = append(api.Config.Auth.AllowedOrganizations, "planet-express")

	newOrga := sdk.Organization{Name: "planet-express"}
	require.NoError(t, organization.Insert(context.TODO(), db, &newOrga))

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

	u, err := user.LoadByUsername(context.TODO(), db, "fry", user.LoadOptions.WithContacts)
	require.NoError(t, err)
	assert.Equal(t, "Philip J. Fry", u.Fullname)
	assert.Equal(t, "fry@planet-express.futurama", u.GetEmail())

	require.NoError(t, user.DeleteByID(db, u.ID))
}

func Test_postAuthSigninHandler_ShouldSuccessWithAKnownUser(t *testing.T) {
	api, db, _ := newTestAPI(t)

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

	expectedUser, err := user.LoadByUsername(context.TODO(), db, "fry")
	require.NoError(t, err)

	// Call a second time, same user should be used

	req = assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
		"username": "fry",
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	u, err := user.LoadByUsername(context.TODO(), db, "fry")
	require.NoError(t, err)
	require.Equal(t, expectedUser.ID, u.ID)

	require.NoError(t, user.DeleteByID(db, u.ID))
}

func Test_postAuthSigninHandler_ShouldSuccessWithAKnownUserAndAnotherConsumerType(t *testing.T) {
	api, db, _ := newTestAPI(t)

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

	u, err := user.LoadByUsername(context.TODO(), db, "fry", user.LoadOptions.WithContacts)
	require.NoError(t, err)
	require.NotNil(t, u)

	// checks that there are 2 consumers now
	consumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerTest, u.ID)
	require.NoError(t, err)
	assert.Equal(t, sdk.ConsumerTest, consumer.Type)

	t.Logf("consumer %s: %+v", consumer.Type, consumer.AuthConsumerUser.Data)

	consumer, err = authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerTest2, u.ID)
	require.NoError(t, err)
	assert.Equal(t, sdk.ConsumerTest2, consumer.Type)

	t.Logf("consumer %s: %+v", consumer.Type, consumer.AuthConsumerUser.Data)

	// tear down
	err = user.DeleteByID(db, u.ID)
	require.NoError(t, err)
}

func Test_postAuthSigninHandler_ShouldSuccessWithAKnownUserAnotherConsumerTypeAndAnotherUsername(t *testing.T) {
	api, db, _ := newTestAPI(t)

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

	u, err := user.LoadByUsername(context.TODO(), db, "fry", user.LoadOptions.WithContacts)
	require.NoError(t, err)
	require.NotNil(t, u)

	// checks that there are 2 consumers now
	consumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerTest, u.ID)
	require.NoError(t, err)
	assert.Equal(t, sdk.ConsumerTest, consumer.Type)

	t.Logf("consumer %s: %+v", consumer.Type, consumer.AuthConsumerUser.Data)

	consumer, err = authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerTest2, u.ID)
	require.NoError(t, err)
	assert.Equal(t, sdk.ConsumerTest2, consumer.Type)

	t.Logf("consumer %s: %+v", consumer.Type, consumer.AuthConsumerUser.Data)

	// tear down
	err = user.DeleteByID(db, u.ID)
	require.NoError(t, err)
}

func Test_postAuthSigninHandler_ShouldSuccessWithAnExistingUserFromCurrentSessionAndAnotherConsumerType(t *testing.T) {
	api, db, _ := newTestAPI(t)

	// clean before test
	u1, _ := user.LoadByUsername(context.TODO(), db, "fry")
	if u1 != nil {
		user.DeleteByID(db, u1.ID)
	}
	u2, _ := user.LoadByUsername(context.TODO(), db, "leela")
	if u2 != nil {
		user.DeleteByID(db, u2.ID)
	}

	uri := api.Router.GetRoute(http.MethodPost, api.postAuthSigninHandler, map[string]string{
		"consumerType": string(sdk.ConsumerTest),
	})
	require.NotEmpty(t, uri)
	req := assets.NewRequest(t, http.MethodPost, uri, sdk.AuthConsumerSigninRequest{
		"username": "fry",
	})
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var res1 sdk.AuthConsumerSigninResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res1))

	// Call a second time with another consumer
	uri = api.Router.GetRoute(http.MethodPost, api.postAuthSigninHandler, map[string]string{
		"consumerType": string(sdk.ConsumerTest2),
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, res1.Token, http.MethodPost, uri, sdk.AuthConsumerSigninRequest{
		"username": "leela",
	})
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var res2 sdk.AuthConsumerSigninResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res2))

	assert.Equal(t, res1.User.ID, res2.User.ID, "in case where we signin with an existing session, the second consumer should be linked to the current user")

	cs, err := user.LoadContactsByUserIDs(context.TODO(), db, []string{res1.User.ID})
	require.NoError(t, err)
	require.Len(t, cs, 2)
	assert.Equal(t, "fry@planet-express.futurama", cs[0].Value)
	assert.Equal(t, "leela@planet-express.futurama", cs[1].Value)

	// tear down
	require.NoError(t, user.DeleteByID(db, res1.User.ID))
}

const (
	MasterKey = `-----BEGIN PGP PUBLIC KEY BLOCK-----

mDMEXF1XRhYJKwYBBAHaRw8BAQdABEHVkfddwOIEFd7V0hsGrudgRuOlnV4/VSK6
YJGFag+0HnRlc3QtbG9ja2VyIDx0ZXN0QGxvbGNhdC5ob3N0PoiQBBMWCAA4FiEE
BN0dlUe5Vi8hx0ZsWXCoyV8Z2eQFAlxdV0YCGwMFCwkIBwIGFQoJCAsCBBYCAwEC
HgECF4AACgkQWXCoyV8Z2eQt5gEAycwThBk4CzuQ8XtPvLA/kml3Jkclgw6ACGsP
YOrnz+gA/2XOjnhYOA6S3sn9g4UMVtON8TofBMTTSqCdgrghu3kFuDgEXF1XRhIK
KwYBBAGXVQEFAQEHQGlq7X9fCeXKxlmcWgT+fFJyS1MlL2uwKQteXl8yIadwAwEI
B4h4BBgWCAAgFiEEBN0dlUe5Vi8hx0ZsWXCoyV8Z2eQFAlxdV0YCGwwACgkQWXCo
yV8Z2eR4rgD/cPn9TStAoXc4Pa+sKgAFmG3NVCNln8FtkH5cQ1g0ouUA/AzcLTL4
VQHT6ArvDWzJKKrh2PepZ5PVMS/Hwh/GDH4J
=n1Ws
-----END PGP PUBLIC KEY BLOCK-----`

	AuthKey = `-----BEGIN PGP PRIVATE KEY BLOCK-----

lFgEXGE6vBYJKwYBBAHaRw8BAQdAWHzsCcqZgcWlcYQdgC+8ZjtBhyuNpiuECnUM
Ya98JlEAAP9LldoVz0MGzqpcy+1B4IzVaEH8rTjlXYqvv8mjWAliORIgtCF0ZXN0
LWxvY2tlci0yIDx0ZXN0MkBsb2xjYXQuaG9zdD6IkAQTFggAOBYhBMViilPFEBsK
55iNjrxDRZAQNUl5BQJcYTq8AhsDBQsJCAcCBhUKCQgLAgQWAgMBAh4BAheAAAoJ
ELxDRZAQNUl5BjEA/26x13zHoICnflMQItCsxM4Ab07li0keyilfwyZ6nm73AQD3
xAuyEnt1hTo1srOafYun/LXNNGxoVsJIfMk7eVn4Boh1BBAWCAAdFiEEBN0dlUe5
Vi8hx0ZsWXCoyV8Z2eQFAlxhTRoACgkQWXCoyV8Z2eTVPwEA5n727+N1bDAm4jJY
HEWL9QOA7MH2+tZKhLXpgdiZ/oMA/25j8nvrdVkxrbQV9ODLomC2Q5whsq5ucj4/
SQUuBb4IiQIzBBABCAAdFiEExDJ6aWjwDDMFny4YxVpSg4XsU8wFAlxhZVcACgkQ
xVpSg4XsU8yJ/xAAt5/X+0AQc0a3z2CB+65gRgV4Fnb35cV59uQFNGEgSRqoVv0s
z1ehIneVtHKDv07eZ143BPjQSl1G9nqZs/4TLOOxfFLpAngRdUoZa2Y8z8YHx/gD
LBwSwrlnPR2/WjX/a/Spukv8hgWNCI/mUs2AOUzPkiyyzBubNRH1JGsONNE7nt4w
PDw6gPlX6DvTDBsxqZienG1EOatYy6NDK3b06ShCc/lYMaQi4yg6AbkTvh9IHPE1
RbBf6hPZVEI7Dwni1RLHs+6fqLrLRUHNdcOg2OUvuaZ9zFU8DWhV3M5H2+1w+0Tv
fI1tThRdgZNCwlveGkoApOK435G95bovFaWo78C+LwuKj6+s9SU/Wt5X6ycGWCJj
VGffpqFjk/GZN6jW8Bp/+4GhYymU+GSC6lkzbA07MbGnQAwd29/QgxaQbjOMEwza
DOYm6vXR8LiNNVOFnZ6ChhzwNxpaHb45zJvZ95FW546JmuCT70AyLSjArQQnpzTa
t+p1xwWlYN+LCYedPZ4JiUioSl9Cz6D/Z9rBhKBPDPVrqHE90t78/91AbQ8el/DB
uOW6mhUAyM2JHdu8oRFKp1PcOBN1FZ4nRK+SOsSqVGLGCQGTUjT4HXrZJRzwGTIV
M7FuMsden1WuaDw5hA7R+0F0P7iDiyhxDVmndntzVgWEERV/SSniuTqCD7acXQRc
YTq8EgorBgEEAZdVAQUBAQdAU1MwdRy9nzPQqWumOn8yW6hn1D3/NoNKLipzLVdp
SjADAQgHAAD/cL/rPYub3SuwSOhmYIr+SySWyM7xz8Eg9YMElj1nsQASVYh4BBgW
CAAgFiEExWKKU8UQGwrnmI2OvENFkBA1SXkFAlxhOrwCGwwACgkQvENFkBA1SXmJ
/gD/SBH6/tULYlpxqD0dO7D4wmHzWKPRtBIPHDWHtWKFAXoA/AjbE7M6aZBHWFAL
dg/94O8U5bC2T8a9CsA/q8eGuucP
=fl70
-----END PGP PRIVATE KEY BLOCK-----`
)

func Test_postAuthSigninHandler_WithCorporateSSO(t *testing.T) {
	api, db, _ := newTestAPI(t)
	api.Config.Auth.AllowedOrganizations = []string{"planet-express"}

	var cfg corpssodriver.SSOConfig
	cfg.Request.Keys.RequestSigningKey = AuthKey
	cfg.Request.RedirectMethod = "POST"
	cfg.Request.RedirectURL = "https://lolcat.local/sso/jwt"
	cfg.Token.KeySigningKey.KeySigningKey = "-----BEGIN PGP PUBLIC KEY BLOCK-----\n\nmDMEXF1XRhYJKwYBBAHaRw8BAQdABEHVkfddwOIEFd7V0hsGrudgRuOlnV4/VSK6\nYJGFag+0HnRlc3QtbG9ja2VyIDx0ZXN0QGxvbGNhdC5ob3N0PoiQBBMWCAA4FiEE\nBN0dlUe5Vi8hx0ZsWXCoyV8Z2eQFAlxdV0YCGwMFCwkIBwIGFQoJCAsCBBYCAwEC\nHgECF4AACgkQWXCoyV8Z2eQt5gEAycwThBk4CzuQ8XtPvLA/kml3Jkclgw6ACGsP\nYOrnz+gA/2XOjnhYOA6S3sn9g4UMVtON8TofBMTTSqCdgrghu3kFuDgEXF1XRhIK\nKwYBBAGXVQEFAQEHQGlq7X9fCeXKxlmcWgT+fFJyS1MlL2uwKQteXl8yIadwAwEI\nB4h4BBgWCAAgFiEEBN0dlUe5Vi8hx0ZsWXCoyV8Z2eQFAlxdV0YCGwwACgkQWXCo\nyV8Z2eR4rgD/cPn9TStAoXc4Pa+sKgAFmG3NVCNln8FtkH5cQ1g0ouUA/AzcLTL4\nVQHT6ArvDWzJKKrh2PepZ5PVMS/Hwh/GDH4J\n=n1Ws\n-----END PGP PUBLIC KEY BLOCK-----"
	cfg.Token.KeySigningKey.SigningKeyClaim = "key"
	cfg.AllowedOrganizations = api.Config.Auth.AllowedOrganizations

	require.NoError(t, organization.Insert(context.TODO(), db, &sdk.Organization{Name: "planet-express"}))

	api.AuthenticationDrivers[sdk.ConsumerCorporateSSO] = corpsso.NewDriver(cfg)

	var requestedJWS string

	t.Run("Test_getAuthAskSigninHandler_WithCorporateSSO", func(t *testing.T) {
		uri := api.Router.GetRoute(http.MethodGet, api.getAuthAskSigninHandler, map[string]string{
			"consumerType": string(sdk.ConsumerCorporateSSO),
		})
		require.NotEmpty(t, uri)

		req := assets.NewRequest(t, "GET", uri, nil)
		rec := httptest.NewRecorder()
		api.Router.Mux.ServeHTTP(rec, req)
		require.Equal(t, 200, rec.Code)

		var redirectInfo sdk.AuthDriverSigningRedirect
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &redirectInfo))

		t.Logf("Redirect info : %+v", redirectInfo)
		require.Equal(t, cfg.Request.RedirectMethod, redirectInfo.Method)
		require.Equal(t, cfg.Request.RedirectURL, redirectInfo.URL)
		require.Equal(t, "application/x-www-form-urlencoded", redirectInfo.ContentType)
		require.NotEmpty(t, redirectInfo.Body)
		require.NotEmpty(t, redirectInfo.Body["request"])
		requestedJWS = redirectInfo.Body["request"]
		var data = sdk.AuthConsumerSigninRequest{}
		data["state"] = requestedJWS
		err := api.AuthenticationDrivers[sdk.ConsumerCorporateSSO].GetDriver().(sdk.DriverWithSigninStateToken).CheckSigninStateToken(data)
		require.NoError(t, err)
	})

	t.Run("Test_postAuthSigninHandler_WithCorporateSSO", func(t *testing.T) {
		t.Cleanup(func() {
			u, _ := user.LoadByUsername(context.TODO(), api.mustDB(), "mattgroening")
			if u != nil {
				require.NoError(t, user.DeleteByID(api.mustDB(), u.ID))
			}
		})

		uri := api.Router.GetRoute(http.MethodPost, api.postAuthSigninHandler, map[string]string{
			"consumerType": string(sdk.ConsumerCorporateSSO),
		})
		require.NotEmpty(t, uri)

		req := assets.NewRequest(t, "POST", uri, sdk.AuthConsumerSigninRequest{
			"state": requestedJWS,
			"token": generateToken(t, "mattgroening"),
		})
		rec := httptest.NewRecorder()
		api.Router.Mux.ServeHTTP(rec, req)
		require.Equal(t, 200, rec.Code)

		var response sdk.AuthConsumerSigninResponse
		var bodyRaw = rec.Body.Bytes()
		require.NoError(t, json.Unmarshal(bodyRaw, &response))

		t.Logf("response: %s", string(bodyRaw))

		require.Equal(t, "mattgroening", response.User.GetUsername())
		require.NotEmpty(t, response.Token)

		u, err := user.LoadByUsername(context.TODO(), api.mustDB(), "mattgroening", user.LoadOptions.WithContacts, user.LoadOptions.WithOrganization)
		require.NoError(t, err)
		require.NotNil(t, u)
		require.Equal(t, "planet-express", u.Organization)
		require.Equal(t, "Mattgroening", u.Fullname)

		consumer, err := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerCorporateSSO, u.ID)
		require.NoError(t, err)
		require.Equal(t, sdk.ConsumerCorporateSSO, consumer.Type)

		t.Logf("consumer %s: %+v", consumer.Type, consumer.AuthConsumerUser.Data)
	})
}

func generateToken(t *testing.T, username string) string {
	ssoToken := corpssodriver.IssuedToken{
		RemoteUser:     username,
		RemoteUsername: strings.Title(username),
		Email:          username + "@planet-express.futurama",
		Organization:   "planet-express",
		Audience:       sdk.UUID(),
		TokenID:        sdk.UUID(),
		MFA:            true,
		IAT:            time.Now().Unix(),
	}
	privKey, err := gpg.NewPrivateKeyFromPem(AuthKey, "")
	if err != nil {
		t.Errorf("unable to parse key: %v", err)
	}
	authSigningKeyData, _ := privKey.Serialize()
	sig, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.EdDSA, Key: jose.JSONWebKey{Key: privKey.GetKey(), KeyID: privKey.KeyId()}},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("key", authSigningKeyData),
	)
	if err != nil {
		t.Error("unable to create JOSE signer", err)
		return ""
	}
	raw, err := jwt.Signed(sig).Claims(ssoToken).CompactSerialize()
	if err != nil {
		t.Error("Failed to create JWT token", err)
		return ""
	}
	return raw
}

func Test_getAuthMe(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, jwtRaw := assets.InsertLambdaUser(t, db)

	uri := api.Router.GetRoute(http.MethodGet, api.getAuthMe, nil)
	require.NotEmpty(t, uri)

	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, "GET", uri, nil)
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)
	t.Logf(rec.Body.String())
}

func TestUserSetOrganization_EnsureGroup(t *testing.T) {
	api, db, _ := newTestAPI(t)

	g1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	u1, _ := assets.InsertLambdaUser(t, db, g1)

	require.NoError(t, organization.Insert(context.TODO(), db, &sdk.Organization{Name: "org0"}))
	require.NoError(t, organization.Insert(context.TODO(), db, &sdk.Organization{Name: "org1"}))
	require.NoError(t, organization.Insert(context.TODO(), db, &sdk.Organization{Name: "org2"}))

	// Set not allowed org should return an error
	api.Config.Auth.AllowedOrganizations = []string{"org0"}
	err := api.userSetOrganization(context.TODO(), db, u1, "org1")
	require.Error(t, err)
	require.Equal(t, "forbidden (from: user organization \"org1\" is not allowed)", err.Error())

	// update org on user must return error
	api.Config.Auth.AllowedOrganizations = []string{"org1", "org2"}
	err = api.userSetOrganization(context.TODO(), db, u1, "org1")
	require.Error(t, err)
	require.Equal(t, "forbidden (from: cannot change user organization to \"org1\", value already set to \"default\")", err.Error())
}

func TestUserSetOrganization_EnsureProject(t *testing.T) {
	api, db, _ := newTestAPI(t)

	require.NoError(t, organization.Insert(context.TODO(), db, &sdk.Organization{Name: "org1"}))
	require.NoError(t, organization.Insert(context.TODO(), db, &sdk.Organization{Name: "org2"}))

	pKey := sdk.RandomString(10)
	p1 := assets.InsertTestProject(t, db, api.Cache, pKey, pKey)

	g1 := &p1.ProjectGroups[0].Group
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	require.NoError(t, group.InsertLinkGroupProject(context.TODO(), db, &group.LinkGroupProject{
		GroupID:   g2.ID,
		ProjectID: p1.ID,
		Role:      sdk.PermissionReadWriteExecute,
	}))

	u1, _ := assets.InsertLambdaUser(t, db, g1)

	// Assert project info
	require.NoError(t, project.LoadOptions.WithGroups(context.TODO(), db, p1))
	require.Equal(t, "default", p1.Organization)
	require.Len(t, p1.ProjectGroups, 2)

	// Set org on u1 should change project organization
	api.Config.Auth.AllowedOrganizations = []string{"org1", "org2"}
	err := api.userSetOrganization(context.TODO(), db, u1, "org1")
	require.Error(t, err)
	require.Equal(t, "forbidden (from: cannot change user organization to \"org1\", value already set to \"default\")", err.Error())
}
