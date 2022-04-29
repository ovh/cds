package gitlab

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rockbears/log"
	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

type authorizeResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// Error match Gitlab error format
type Error struct {
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

//AuthorizeRedirect returns the request token, the Authorize URL
func (g *gitlabConsumer) AuthorizeRedirect(ctx context.Context) (string, string, error) {
	_, end := telemetry.Span(ctx, "gitlab.AuthorizeRedirect")
	defer end()

	// See https://docs.gitlab.com/ce/api/oauth2.html

	requestToken, err := sdk.GenerateHash()
	if err != nil {
		return "", "", err
	}

	val := url.Values{}
	val.Add("redirect_uri", g.AuthorizationCallbackURL)
	val.Add("client_id", g.appID)
	val.Add("response_type", "code")
	val.Add("state", requestToken)

	url := fmt.Sprintf("%s/oauth/authorize?%s", g.URL, val.Encode())
	return requestToken, url, nil
}

func (g *gitlabConsumer) postForm(path string, data url.Values, headers map[string][]string) (int, []byte, error) {
	body := strings.NewReader(data.Encode())

	req, err := http.NewRequest(http.MethodPost, g.URL+path, body)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "CDS-gl_client_id="+g.appID)
	for k, h := range headers {
		for i := range h {
			req.Header.Add(k, h[i])
		}
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, err
	}

	if res.StatusCode > 400 {
		glErr := &Error{}
		if err := sdk.JSONUnmarshal(resBody, glErr); err == nil {
			return res.StatusCode, resBody, fmt.Errorf("%s: %s", glErr.Error, glErr.Description)
		}
	}

	return res.StatusCode, resBody, nil
}

//AuthorizeToken returns the authorized token (and its secret)
//from the request token and the verifier got on authorize url
func (g *gitlabConsumer) AuthorizeToken(ctx context.Context, state, code string) (string, string, error) {
	log.Debug(ctx, "GitlabDriver.AuthorizeToken: state:%s code:%s", state, code)

	params := url.Values{}
	params.Add("client_id", g.appID)
	params.Add("client_secret", g.secret)
	params.Add("code", code)
	params.Add("grant_type", "authorization_code")
	params.Add("redirect_uri", g.AuthorizationCallbackURL)

	headers := map[string][]string{}
	headers["Accept"] = []string{"application/json"}

	status, res, err := g.postForm("/oauth/token", params, headers)
	if err != nil {
		return "", "", err
	}

	if status < 200 && status >= 400 {
		return "", "", fmt.Errorf("Gitlab error (%d) %s ", status, string(res))
	}

	glResponse := authorizeResponse{}
	if err := sdk.JSONUnmarshal(res, &glResponse); err != nil {
		return "", "", fmt.Errorf("Unable to parse gitlab response (%d) %s ", status, string(res))
	}

	return glResponse.AccessToken, state, nil
}

//GetAuthorized returns an authorized client
func (g *gitlabConsumer) GetAuthorizedClient(ctx context.Context, vcsAuth sdk.VCSAuth) (sdk.VCSAuthorizedClient, error) {
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
	}
	if vcsAuth.VCSProject != nil {
		gclient := gitlab.NewClient(httpClient, vcsAuth.VCSProject.Auth["token"])
		c := &gitlabClient{
			client:              gclient,
			uiURL:               g.uiURL,
			disableStatus:       g.disableStatus,
			disableStatusDetail: g.disableStatusDetail,
		}
		c.client.SetBaseURL(g.URL + "/api/v4")
		return c, nil
	}

	// DEPRECATED VCS
	gclient := gitlab.NewOAuthClient(httpClient, vcsAuth.AccessToken)
	c := &gitlabClient{
		client:              gclient,
		uiURL:               g.uiURL,
		disableStatus:       g.disableStatus,
		disableStatusDetail: g.disableStatusDetail,
	}
	c.client.SetBaseURL(g.URL + "/api/v4")
	return c, nil
}
