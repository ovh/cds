package github

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/sdk"
)

var _ sdk.DriverWithRedirect = new(githubDriver)
var _ sdk.DriverWithSigninStateToken = new(githubDriver)

// NewGithubDriver returns a new Github driver for given config.
func NewGithubDriver(cdsURL, url, urlAPI, clientID, clientSecret string) sdk.Driver {
	return &githubDriver{
		cdsURL:       cdsURL,
		url:          url,
		urlAPI:       urlAPI,
		clientID:     clientID,
		clientSecret: clientSecret,
	}
}

type githubDriver struct {
	cdsURL       string
	url          string
	urlAPI       string
	clientID     string
	clientSecret string
}

func (gd githubDriver) GetSigninURI(signinState sdk.AuthSigninConsumerToken) (sdk.AuthDriverSigningRedirect, error) {
	// Generate a new state value for the auth signin request
	jws, err := authentication.NewDefaultSigninStateToken(signinState)
	if err != nil {
		return sdk.AuthDriverSigningRedirect{}, err
	}
	var result = sdk.AuthDriverSigningRedirect{
		Method: http.MethodGet,
		URL:    fmt.Sprintf("%s/login/oauth/authorize?client_id=%s&scope=user&state=%s&redirect_uri=%s", gd.url, gd.clientID, jws, gd.cdsURL+"/auth/callback/github"),
	}

	return result, nil
}

func (gd githubDriver) CheckSigninRequest(req sdk.AuthConsumerSigninRequest) error {
	code, err := req.StringE("code")
	if err != nil || code == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid gitlab code")
	}
	return nil
}

func (d githubDriver) CheckSigninStateToken(req sdk.AuthConsumerSigninRequest) error {
	// Check if state is given and if its valid
	state, err := req.StringE("state")
	if err != nil {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid state value")
	}

	return authentication.CheckDefaultSigninStateToken(state)
}

func (gd githubDriver) GetUserInfoFromDriver(ctx context.Context, req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	var info sdk.AuthDriverUserInfo

	config := &oauth2.Config{
		Endpoint: oauth2.Endpoint{
			TokenURL: fmt.Sprintf("%s/login/oauth/access_token", gd.url),
		},
	}

	ctx2 := context.WithValue(context.Background(), oauth2.HTTPClient, http.DefaultClient)
	t, err := config.Exchange(ctx2, req.String("code"),
		oauth2.SetAuthURLParam("client_id", gd.clientID),
		oauth2.SetAuthURLParam("client_secret", gd.clientSecret),
		oauth2.SetAuthURLParam("state", req.String("state")),
		oauth2.SetAuthURLParam("redirect_uri", gd.cdsURL+"/auth/callback/github"),
	)
	if err != nil {
		return info, sdk.WrapError(err, "cannot get github token with given code")
	}

	request, err := http.NewRequest(http.MethodGet, gd.urlAPI+"/user", nil)
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
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return info, sdk.WithStack(err)
	}

	var githubUser struct {
		ID    int    `json:"id"`
		Login string `json:"login"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := sdk.JSONUnmarshal(resBody, &githubUser); err != nil {
		return info, sdk.WithStack(err)
	}

	info.ExternalID = fmt.Sprintf("%d", githubUser.ID)
	info.Username = githubUser.Login
	info.Fullname = githubUser.Name
	info.Email = githubUser.Email
	return info, nil
}
