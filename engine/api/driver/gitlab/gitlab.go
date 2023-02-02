package gitlab

import (
	"context"
	"fmt"
	"net/http"

	"github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/sdk"
)

var _ sdk.DriverWithRedirect = new(gitlabDriver)
var _ sdk.DriverWithSigninStateToken = new(gitlabDriver)

// NewGitlabDriver returns a new Gitlab auth driver for given config.
func NewGitlabDriver(cdsURL, url, applicationID, secret string) sdk.Driver {
	return &gitlabDriver{
		cdsURL:        cdsURL,
		url:           url,
		applicationID: applicationID,
		secret:        secret,
	}
}

type gitlabDriver struct {
	cdsURL        string
	url           string
	applicationID string
	secret        string
}

func (g gitlabDriver) GetSigninURI(signinState sdk.AuthSigninConsumerToken) (sdk.AuthDriverSigningRedirect, error) {
	// Generate a new state value for the auth signin request
	jws, err := authentication.NewDefaultSigninStateToken(signinState)
	if err != nil {
		return sdk.AuthDriverSigningRedirect{}, err
	}

	var result = sdk.AuthDriverSigningRedirect{
		Method: http.MethodGet,
		URL:    fmt.Sprintf("%s/oauth/authorize?client_id=%s&response_type=code&state=%s&redirect_uri=%s", g.url, g.applicationID, jws, g.cdsURL+"/auth/callback/gitlab"),
	}

	return result, nil
}

func (g gitlabDriver) CheckSigninRequest(req sdk.AuthConsumerSigninRequest) error {
	code, err := req.StringE("code")
	if err != nil || code == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing or invalid gitlab code")
	}
	return nil
}

func (g gitlabDriver) CheckSigninStateToken(req sdk.AuthConsumerSigninRequest) error {
	// Check if state is given and if its valid
	state, err := req.StringE("state")
	if err != nil {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing state value")
	}
	return authentication.CheckDefaultSigninStateToken(state)
}

func (g gitlabDriver) GetUserInfoFromDriver(ctx context.Context, req sdk.AuthConsumerSigninRequest) (sdk.AuthDriverUserInfo, error) {
	var info sdk.AuthDriverUserInfo

	config := &oauth2.Config{
		Endpoint: oauth2.Endpoint{
			TokenURL: fmt.Sprintf("%s/oauth/token", g.url),
		},
	}

	ctx2 := context.WithValue(ctx, oauth2.HTTPClient, http.DefaultClient)
	t, err := config.Exchange(ctx2, req.String("code"),
		oauth2.SetAuthURLParam("client_id", g.applicationID),
		oauth2.SetAuthURLParam("client_secret", g.secret),
		oauth2.SetAuthURLParam("grant_type", "authorization_code"),
		oauth2.SetAuthURLParam("redirect_uri", g.cdsURL+"/auth/callback/gitlab"),
	)
	if err != nil {
		return info, sdk.WrapError(err, "cannot get gitlab token with given code")
	}

	c := gitlab.NewOAuthClient(http.DefaultClient, t.AccessToken)
	if err := c.SetBaseURL(g.url); err != nil {
		return info, sdk.WrapError(err, "invalid gitlab url")
	}
	me, res, err := c.Users.CurrentUser()
	if err != nil {
		return info, sdk.WrapError(err, "cannot get current user from gitlab")
	}
	if res.StatusCode != 200 {
		return info, sdk.NewErrorFrom(sdk.ErrUnknownError, "cannot get current user from gitlab")
	}

	info.ExternalID = fmt.Sprintf("%d", me.ID)
	info.Username = me.Username
	info.Fullname = me.Name
	info.Email = me.Email
	return info, nil
}
