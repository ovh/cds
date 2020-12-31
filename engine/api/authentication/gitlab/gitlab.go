package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/xanzy/go-gitlab"

	"golang.org/x/oauth2"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/sdk"
)

var _ sdk.AuthDriverWithRedirect = new(authDriver)
var _ sdk.AuthDriverWithSigninStateToken = new(authDriver)

// NewDriver returns a new Gitlab auth driver for given config.
func NewDriver(signupDisabled bool, cdsURL, url, applicationID, secret string) sdk.AuthDriver {
	return &authDriver{
		signupDisabled: signupDisabled,
		cdsURL:         cdsURL,
		url:            url,
		applicationID:  applicationID,
		secret:         secret,
	}
}

type authDriver struct {
	signupDisabled bool
	cdsURL         string
	url            string
	applicationID  string
	secret         string
}

func (d authDriver) GetManifest() sdk.AuthDriverManifest {
	return sdk.AuthDriverManifest{
		Type:           sdk.ConsumerGitlab,
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
		URL:    fmt.Sprintf("%s/oauth/authorize?client_id=%s&response_type=code&state=%s&redirect_uri=%s", d.url, d.applicationID, jws, d.cdsURL+"/auth/callback/gitlab"),
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
			TokenURL: fmt.Sprintf("%s/oauth/token", d.url),
		},
	}

	ctx2 := context.WithValue(ctx, oauth2.HTTPClient, http.DefaultClient)
	t, err := config.Exchange(ctx2, req["code"],
		oauth2.SetAuthURLParam("client_id", d.applicationID),
		oauth2.SetAuthURLParam("client_secret", d.secret),
		oauth2.SetAuthURLParam("grant_type", "authorization_code"),
		oauth2.SetAuthURLParam("redirect_uri", d.cdsURL+"/auth/callback/gitlab"),
	)
	if err != nil {
		return info, sdk.WrapError(err, "cannot get gitlab token with given code")
	}

	c := gitlab.NewOAuthClient(http.DefaultClient, t.AccessToken)
	if err := c.SetBaseURL(d.url); err != nil {
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
