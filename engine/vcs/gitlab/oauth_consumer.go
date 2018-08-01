package gitlab

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
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

func generateHash() (string, error) {
	size := 128
	bs := make([]byte, size)
	if _, err := rand.Read(bs); err != nil {
		log.Error("generateID: rand.Read failed: %s\n", err)
		return "", err
	}
	str := hex.EncodeToString(bs)
	token := []byte(str)[0:size]

	log.Debug("generateID: new generated id: %s\n", token)
	return string(token), nil
}

//AuthorizeRedirect returns the request token, the Authorize URL
func (g *gitlabConsumer) AuthorizeRedirect(ctx context.Context) (string, string, error) {
	// See https://docs.gitlab.com/ce/api/oauth2.html

	requestToken, err := generateHash()
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
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, err
	}

	if res.StatusCode > 400 {
		glErr := &Error{}
		if err := json.Unmarshal(resBody, glErr); err == nil {
			return res.StatusCode, resBody, fmt.Errorf("%s: %s", glErr.Error, glErr.Description)
		}
	}

	return res.StatusCode, resBody, nil
}

//AuthorizeToken returns the authorized token (and its secret)
//from the request token and the verifier got on authorize url
func (g *gitlabConsumer) AuthorizeToken(ctx context.Context, state, code string) (string, string, error) {
	log.Debug("GitlabDriver.AuthorizeToken: state:%s code:%s", state, code)

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
	if err := json.Unmarshal(res, &glResponse); err != nil {
		return "", "", fmt.Errorf("Unable to parse gitlab response (%d) %s ", status, string(res))
	}

	return glResponse.AccessToken, state, nil
}

var instancesAuthorizedClient = map[string]*gitlabClient{}

//GetAuthorized returns an authorized client
func (g *gitlabConsumer) GetAuthorizedClient(ctx context.Context, accessToken, accessTokenSecret string) (sdk.VCSAuthorizedClient, error) {
	c, ok := instancesAuthorizedClient[accessToken]
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
	}
	if !ok {
		c = &gitlabClient{
			client:              gitlab.NewOAuthClient(httpClient, accessToken),
			uiURL:               g.uiURL,
			disableStatus:       g.disableStatus,
			disableStatusDetail: g.disableStatusDetail,
		}
		c.client.SetBaseURL(g.URL + "/api/v4")
		instancesAuthorizedClient[accessToken] = c
	}
	return c, nil
}
