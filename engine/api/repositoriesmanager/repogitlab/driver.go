package repogitlab

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	apiURL string
	uiURL  string
)

// Init initializes repostash package
func Init(apiurl, uiurl string) {
	apiURL = apiurl
	uiURL = uiurl
}

// GitlabDriver implements RepositoryManagerDriver
type GitlabDriver struct {
	URL                      string `json:"url"`
	Secret                   string `json:"-"`
	ID                       string `json:"id"`
	AuthorizationCallbackURL string `json:"authorization-callback-url"`
}

// Error match Gitlab error format
type Error struct {
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

func NewGitlabDriver(id int64, name, URL, authorizationCallbackURL, secret string, args map[string]string, consumerData string) (*GitlabDriver, error) {
	gd := &GitlabDriver{URL: URL, Secret: secret, AuthorizationCallbackURL: authorizationCallbackURL}

	if consumerData == "" {
		id, ok := args["app-id"]
		if !ok {
			return nil, fmt.Errorf("no app-id provided to Gitlab driver")
		}
		gd.ID = id

		return gd, nil
	}

	if err := json.Unmarshal([]byte(consumerData), &gd); err != nil {
		return nil, err
	}

	return gd, nil
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
func (d *GitlabDriver) AuthorizeRedirect() (string, string, error) {
	// See https://docs.gitlab.com/ce/api/oauth2.html

	requestToken, err := generateHash()
	if err != nil {
		return "", "", err
	}

	val := url.Values{}
	val.Add("redirect_uri", d.AuthorizationCallbackURL)
	val.Add("client_id", d.ID)
	val.Add("response_type", "code")
	val.Add("state", requestToken)

	url := fmt.Sprintf("%s/oauth/authorize?%s", d.URL, val.Encode())
	return requestToken, url, nil
}

type authorizeResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

//AuthorizeToken returns the authorized token (and its secret)
//from the request token and the verifier got on authorize url
func (d *GitlabDriver) AuthorizeToken(state, code string) (string, string, error) {
	log.Debug("GitlabDriver.AuthorizeToken: state:%s code:%s", state, code)

	params := url.Values{}
	params.Add("client_id", d.ID)
	params.Add("client_secret", d.Secret)
	params.Add("code", code)
	params.Add("grant_type", "authorization_code")
	params.Add("redirect_uri", d.AuthorizationCallbackURL)

	headers := map[string][]string{}
	headers["Accept"] = []string{"application/json"}

	status, res, err := d.postForm("/oauth/token", params, headers)
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

func (g *GitlabDriver) postForm(path string, data url.Values, headers map[string][]string) (int, []byte, error) {
	body := strings.NewReader(data.Encode())

	req, err := http.NewRequest(http.MethodPost, g.URL+path, body)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "CDS-gl_client_id="+g.ID)
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

//Data returns a serilized version of specific data
func (d *GitlabDriver) Data() string {
	b, _ := json.Marshal(d)
	return string(b)
}

//GetAuthorized returns an authorized client
func (d *GitlabDriver) GetAuthorized(accessToken, accessTokenSecret string) (sdk.RepositoriesManagerClient, error) {
	client, err := NewGitlabClient(d.URL, accessToken)
	if err != nil {
		return nil, err
	}
	return client, nil
}

//HooksSupported returns true if the driver technically support hook
func (d *GitlabDriver) HooksSupported() bool {
	return true
}

//PollingSupported returns true if the driver technically support polling
func (d *GitlabDriver) PollingSupported() bool {
	return false
}
