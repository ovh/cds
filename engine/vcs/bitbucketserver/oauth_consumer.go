package bitbucketserver

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

const oauth1OOB = "oob"

//AuthorizeRedirect returns the request token, the Authorize URL
func (g *bitbucketConsumer) AuthorizeRedirect(ctx context.Context) (string, string, error) {
	_, end := telemetry.Span(ctx, "bitbucketserver.AuthorizeRedirect")
	defer end()
	requestToken, err := g.RequestToken()
	if err != nil {
		log.Warn(ctx, "requestToken>%s\n", err)
		return "", "", sdk.WrapError(err, "Unable to get request token")
	}

	redirect, err := url.Parse(g.authorizationURL)
	if err != nil {
		return "", "", sdk.WrapError(err, "Unable to parse authorization url")
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
func (g *bitbucketConsumer) AuthorizeToken(ctx context.Context, token, verifier string) (string, string, error) {
	_, end := telemetry.Span(ctx, "bitbucketserver.AuthorizeToken")
	defer end()
	accessTokenURL, _ := url.Parse(g.accessTokenURL)
	req := http.Request{
		URL:    accessTokenURL,
		Method: "POST",
		Close:  true,
	}
	t := NewAccessToken(token, "", map[string]string{})
	err := g.SignParams(&req, t, map[string]string{"oauth_verifier": verifier})
	if err != nil {
		return "", "", sdk.WrapError(err, "Unable to sign params")
	}

	resp, err := httpClient.Do(&req)
	if err != nil {
		return "", "", sdk.WrapError(err, "Unable to parse get authorize url")
	}

	accessToken, err := ParseAccessToken(resp.Body)
	if err != nil {
		return "", "", sdk.WrapError(err, "Unable to parse access token")
	}

	return accessToken.Token(), accessToken.Secret(), nil
}

//GetAuthorized returns an authorized client
func (g *bitbucketConsumer) GetAuthorizedClient(ctx context.Context, vcsAuth sdk.VCSAuth) (sdk.VCSAuthorizedClient, error) {
	if vcsAuth.URL != "" {
		return &bitbucketClient{
			consumer: *g,
			proxyURL: g.proxyURL,
			username: g.username,
			token:    g.token,
		}, nil
	}
	return &bitbucketClient{
		consumer:          *g,
		accessToken:       vcsAuth.AccessToken,
		accessTokenSecret: vcsAuth.AccessTokenSecret,
		token:             g.token,
		username:          g.username,
		proxyURL:          g.proxyURL,
	}, nil
}
