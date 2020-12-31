package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ovh/cds/engine/api/authentication"

	"github.com/ovh/cds/sdk"
	"golang.org/x/oauth2"
)

var _ sdk.AuthDriverWithRedirect = new(authDriver)
var _ sdk.AuthDriverWithSigninStateToken = new(authDriver)

// NewDriver returns a new Github auth driver for given config.
func NewDriver(signupDisabled bool, cdsURL, url, urlAPI, clientID, clientSecret string) sdk.AuthDriver {
	return &authDriver{
		signupDisabled: signupDisabled,
		cdsURL:         cdsURL,
		url:            url,
		urlAPI:         urlAPI,
		clientID:       clientID,
		clientSecret:   clientSecret,
	}
}

type authDriver struct {
	signupDisabled bool
	cdsURL         string
	url            string
	urlAPI         string
	clientID       string
	clientSecret   string
}

func (d authDriver) GetManifest() sdk.AuthDriverManifest {
	return sdk.AuthDriverManifest{
		Type:           sdk.ConsumerGithub,
		SignupDisabled: d.signupDisabled,
	}
}

func (d authDriver) GetSigninURI(signinState sdk.AuthSigninConsumerToken) (sdk.AuthDriverSigningRedirect, error) {
	// Generate a new state value for the auth signin request
	jws, err := authentication.NewDefaultSigninStateToken(signinState.Origin,
		signinState.RedirectURI, signinState.IsFirstConnection)
	if err != nil {
		return sdk.AuthDriverSigningRedirect{}, err
	}

	var result = sdk.AuthDriverSigningRedirect{
		Method: http.MethodGet,
		URL:    fmt.Sprintf("%s/login/oauth/authorize?client_id=%s&scope=user&state=%s&redirect_uri=%s", d.url, d.clientID, jws, d.cdsURL+"/auth/callback/github"),
	}

	return result, nil
}

func (d authDriver) GetSessionDuration(_ sdk.AuthDriverUserInfo, _ sdk.AuthConsumer) time.Duration {
	return time.Hour * 24 * 30 // 1 month session
}

func (d authDriver) CheckSigninRequest(req sdk.AuthConsumerSigninRequest) error {
	if code, ok := req["code"]; !ok || code == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid gitlab code")
	}
	return nil
}

func (d authDriver) CheckSigninStateToken(req sdk.AuthConsumerSigninRequest) error {
	// Check if state is given and if its valid
	state, okState := req["state"]
	if !okState {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing state value")
	}

	return authentication.CheckDefaultSigninStateToken(state)
}

func (d authDriver) GetUserInfo(ctx context.Context, req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	var info sdk.AuthDriverUserInfo

	config := &oauth2.Config{
		Endpoint: oauth2.Endpoint{
			TokenURL: fmt.Sprintf("%s/login/oauth/access_token", d.url),
		},
	}

	ctx2 := context.WithValue(context.Background(), oauth2.HTTPClient, http.DefaultClient)
	t, err := config.Exchange(ctx2, req["code"],
		oauth2.SetAuthURLParam("client_id", d.clientID),
		oauth2.SetAuthURLParam("client_secret", d.clientSecret),
		oauth2.SetAuthURLParam("state", req["state"]),
		oauth2.SetAuthURLParam("redirect_uri", d.cdsURL+"/auth/callback/github"),
	)
	if err != nil {
		return info, sdk.WrapError(err, "cannot get github token with given code")
	}

	request, err := http.NewRequest(http.MethodGet, d.urlAPI+"/user", nil)
	if err != nil {
		return info, sdk.WithStack(err)
	}
	request.Header.Set("Authorization", "token "+t.AccessToken)

	res, err := http.DefaultClient.Do(request)
	if err != nil {
		return info, sdk.WithStack(err)
	}
	if res.StatusCode != 200 {
		return info, sdk.NewErrorFrom(sdk.ErrUnknownError, "cannot get current user from github")
	}

	defer res.Body.Close()
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return info, sdk.WithStack(err)
	}

	var githubUser struct {
		ID    int    `json:"id"`
		Login string `json:"login"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(resBody, &githubUser); err != nil {
		return info, sdk.WithStack(err)
	}

	info.ExternalID = fmt.Sprintf("%d", githubUser.ID)
	info.Username = githubUser.Login
	info.Fullname = githubUser.Name
	info.Email = githubUser.Email

	return info, nil
}
