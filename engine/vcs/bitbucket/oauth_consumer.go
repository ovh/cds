package bitbucket

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

const oauth1OOB = "oob"

//AuthorizeRedirect returns the request token, the Authorize URL
func (g *bitbucketConsumer) AuthorizeRedirect() (string, string, error) {
	requestToken, err := g.RequestToken()
	if err != nil {
		log.Warning("requestToken>%s\n", err)
		return "", "", err
	}

	redirect, err := url.Parse(g.authorizationURL)
	if err != nil {
		return "", "", err
	}

	params := make(url.Values)
	params.Add("oauth_token", requestToken.token)
	redirect.RawQuery = params.Encode()

	u := redirect.String()
	if strings.HasPrefix(u, "https://bitbucket.org/%21api/") {
		u = strings.Replace(u, "/%21api/", "/!api/", -1)
	}

	return requestToken.Token(), u, nil
}

//AuthorizeToken returns the authorized token (and its secret)
//from the request token and the verifier got on authorize url
func (g *bitbucketConsumer) AuthorizeToken(token, verifier string) (string, string, error) {
	accessTokenURL, _ := url.Parse(g.accessTokenURL)
	req := http.Request{
		URL:    accessTokenURL,
		Method: "POST",
		Close:  true,
	}
	t := NewAccessToken(token, "", map[string]string{})
	err := g.SignParams(&req, t, map[string]string{"oauth_verifier": verifier})
	if err != nil {
		return "", "", err
	}

	resp, err := httpClient.Do(&req)
	if err != nil {
		return "", "", err
	}

	accessToken, err := ParseAccessToken(resp.Body)
	if err != nil {
		return "", "", err
	}

	return accessToken.Token(), accessToken.Secret(), nil
}

//keep client in memory
var instancesAuthorizedClient = map[string]*bitbucketClient{}

//GetAuthorized returns an authorized client
func (g *bitbucketConsumer) GetAuthorizedClient(accessToken, accessTokenSecret string) (sdk.VCSAuthorizedClient, error) {
	c, ok := instancesAuthorizedClient[accessToken]
	if !ok {
		c = &bitbucketClient{
			consumer:          *g,
			accessToken:       accessToken,
			accessTokenSecret: accessTokenSecret,
		}
		instancesAuthorizedClient[accessToken] = c
	}
	return c, nil
}
