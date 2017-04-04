package repogithub

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/url"
	"strings"

	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk"
)

//Github const
const (
	URL    = "https://github.com"
	APIURL = "https://api.github.com"
)

//Github const
var (
	RequestedScope = []string{"user", "repo", "admin:repo_hook", "admin:org_hook"} //https://developer.github.com/v3/oauth/#scopes
)

func generateHash() (string, error) {
	size := 128
	bs := make([]byte, size)
	if _, err := rand.Read(bs); err != nil {
		log.Critical("generateID: rand.Read failed: %s\n", err)
		return "", err
	}
	str := hex.EncodeToString(bs)
	token := []byte(str)[0:size]

	log.Debug("generateID: new generated id: %s\n", token)
	return string(token), nil
}

//GithubConsumer embeds a github oauth2 consumer
type GithubConsumer struct {
	ClientID                 string `json:"client-id"`
	ClientSecret             string `json:"client-secret"`
	AuthorizationCallbackURL string `json:"-"`
	WithHooks                bool   `json:"with-hooks"`
	WithPolling              bool   `json:"with-polling"`
	DisableSetStatus         bool   `json:"-"`
	DisableStatusURL         bool   `json:"-"`
}

//New creates a new GithubConsumer
func New(ClientID, ClientSecret, AuthorizationCallbackURL string) *GithubConsumer {
	return &GithubConsumer{
		ClientID:                 ClientID,
		ClientSecret:             ClientSecret,
		AuthorizationCallbackURL: AuthorizationCallbackURL,
	}
}

func (g *GithubConsumer) getClientSecretValue() ([]byte, error) {
	b, err := ioutil.ReadFile(g.ClientSecret)
	if err != nil {
		log.Critical("GithubConsumer> Unable to read client secret value %s : %s", g.ClientSecret, err)
		return nil, err
	}
	b = bytes.Replace(b, []byte{'\n'}, []byte{}, -1)
	return b, err
}

//Data returns a serilized version of specific data
func (g *GithubConsumer) Data() string {
	b, _ := json.Marshal(g)
	return string(b)
}

//AuthorizeRedirect returns the request token, the Authorize URL
//doc: https://developer.github.com/v3/oauth/#web-application-flow
func (g *GithubConsumer) AuthorizeRedirect() (string, string, error) {
	// GET https://github.com/login/oauth/authorize
	// with parameters : client_id, redirect_uri, scope, state
	requestToken, err := generateHash()
	if err != nil {
		return "", "", err
	}

	val := url.Values{}
	val.Add("client_id", g.ClientID)
	//Leave the default value set in github
	//val.Add("redirect_uri", g.AuthorizationCallbackURL)
	val.Add("scope", strings.Join(RequestedScope, " "))
	val.Add("state", requestToken)

	authorizeURL := fmt.Sprintf("%s/login/oauth/authorize?%s", URL, val.Encode())

	return requestToken, authorizeURL, nil
}

//AuthorizeToken returns the authorized token (and its secret)
//from the request token and the verifier got on authorize url
func (g *GithubConsumer) AuthorizeToken(state, code string) (string, string, error) {
	log.Debug("AuthorizeToken> Github send code %s for state %s", code, state)
	//POST https://github.com/login/oauth/access_token
	//Parameters:
	//	client_id
	//	client_secret
	//	code
	//	state

	secret, err := g.getClientSecretValue()
	if err != nil {
		return "", "", err
	}

	params := url.Values{}
	params.Add("client_id", g.ClientID)
	params.Add("client_secret", string(secret))
	params.Add("code", code)
	params.Add("state", state)

	headers := map[string][]string{}
	headers["Accept"] = []string{"application/json"}

	status, res, err := g.postForm("/login/oauth/access_token", params, headers)
	if err != nil {
		return "", "", err
	}

	if status < 200 && status >= 400 {
		return "", "", fmt.Errorf("Github error (%d) %s ", status, string(res))
	}

	ghResponse := map[string]string{}
	if err := json.Unmarshal(res, &ghResponse); err != nil {
		return "", "", fmt.Errorf("Unable to parse github response (%d) %s ", status, string(res))
	}

	//Github return scope as "scope":"repo,gist"
	//Check all scopes : see docs
	//	The scope attribute lists scopes attached to the token that were granted by the user. Normally, these scopes will be identical to what you requested
	//  When requesting multiple scopes, the token will be saved with a normalized list of scopes, discarding those that are implicitly included by another requested scope
	ghScope := strings.Split(ghResponse["scope"], ",")
	var allFound = true
	for _, s := range RequestedScope {
		var found bool
		for i := range ghScope {
			if ghScope[i] == s {
				found = true
				break
			}
		}
		if !found {
			allFound = false
			break
		}
	}
	if !allFound {
		return "", "", fmt.Errorf("Scopes doesn't match with request : %s %s", strings.Join(RequestedScope, " "), string(res))
	}

	return ghResponse["access_token"], state, nil
}

//keep client in memory
var instancesAuthorizedClient = map[string]sdk.RepositoriesManagerClient{}

//GetAuthorized returns an authorized client
func (g *GithubConsumer) GetAuthorized(accessToken, accessTokenSecret string) (sdk.RepositoriesManagerClient, error) {
	c := instancesAuthorizedClient[accessToken]
	if c == nil {
		c = &GithubClient{
			ClientID:         g.ClientID,
			OAuthToken:       accessToken,
			DisableSetStatus: g.DisableSetStatus,
			DisableStatusURL: g.DisableStatusURL,
		}
		instancesAuthorizedClient[accessToken] = c
	}

	return c, c.(*GithubClient).RateLimit()
}

//HooksSupported returns true if the driver technically support hook
func (g *GithubConsumer) HooksSupported() bool {
	return false
}

//PollingSupported returns true if the driver technically support polling
func (g *GithubConsumer) PollingSupported() bool {
	return true
}
