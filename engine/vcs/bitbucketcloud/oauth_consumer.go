package bitbucketcloud

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

const (
	bitbucketCloudAccessTokenURL = "https://bitbucket.org/site/oauth2/access_token"
)

//AuthorizeRedirect returns the request token, the Authorize Bitbucket cloud
func (consumer *bitbucketcloudConsumer) AuthorizeRedirect(ctx context.Context) (string, string, error) {
	_, end := telemetry.Span(ctx, "bitbucketcloud.AuthorizeRedirect")
	defer end()
	requestToken, err := sdk.GenerateHash()
	if err != nil {
		return "", "", err
	}

	val := url.Values{}
	val.Add("client_id", consumer.ClientID)
	val.Add("response_type", "code")
	val.Add("state", requestToken)

	authorizeURL := fmt.Sprintf("https://bitbucket.org/site/oauth2/authorize?%s", val.Encode())

	return requestToken, authorizeURL, nil
}

//AuthorizeToken returns the authorized token (and its refresh_token)
//from the request token and the verifier got on authorize url
func (consumer *bitbucketcloudConsumer) AuthorizeToken(ctx context.Context, _, code string) (string, string, error) {
	_, end := telemetry.Span(ctx, "bitbucketcloud.AuthorizeToken")
	defer end()
	log.Debug(ctx, "AuthorizeToken> Bitbucketcloud send code %s", code)

	params := url.Values{}
	params.Add("code", code)
	params.Add("grant_type", "authorization_code")

	headers := map[string][]string{}
	headers["Accept"] = []string{"application/json"}

	status, res, err := consumer.postForm(bitbucketCloudAccessTokenURL, params, headers)
	if err != nil {
		return "", "", err
	}

	if status < 200 || status >= 400 {
		return "", "", fmt.Errorf("Bitbucket cloud error (%d) %s ", status, string(res))
	}

	var resp AccessToken
	if err := sdk.JSONUnmarshal(res, &resp); err != nil {
		return "", "", fmt.Errorf("Unable to parse bitbucketcloud response (%d) %s ", status, string(res))
	}

	return resp.AccessToken, resp.RefreshToken, nil
}

//RefreshToken returns the refreshed authorized token
func (consumer *bitbucketcloudConsumer) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	params := url.Values{}
	params.Add("refresh_token", refreshToken)
	params.Add("grant_type", "refresh_token")

	headers := map[string][]string{}
	headers["Accept"] = []string{"application/json"}

	status, res, err := consumer.postForm(bitbucketCloudAccessTokenURL, params, headers)
	if err != nil {
		return "", "", err
	}

	if status < 200 || status >= 400 {
		return "", "", fmt.Errorf("Bitbucket cloud error (%d) %s ", status, string(res))
	}

	var resp AccessToken
	if err := sdk.JSONUnmarshal(res, &resp); err != nil {
		return "", "", fmt.Errorf("Unable to parse bitbucketcloud response (%d) %s ", status, string(res))
	}

	return resp.AccessToken, resp.RefreshToken, nil
}

//keep client in memory
var instancesAuthorizedClient = map[string]*bitbucketcloudClient{}

//GetAuthorized returns an authorized client
func (consumer *bitbucketcloudConsumer) GetAuthorizedClient(ctx context.Context, accessToken, refreshToken string, created int64) (sdk.VCSAuthorizedClient, error) {
	createdTime := time.Unix(created, 0)

	c, ok := instancesAuthorizedClient[accessToken]
	if createdTime.Add(2 * time.Hour).Before(time.Now()) {
		if ok {
			delete(instancesAuthorizedClient, accessToken)
		}
		newAccessToken, _, err := consumer.RefreshToken(ctx, refreshToken)
		if err != nil {
			return nil, sdk.WrapError(err, "cannot refresh token")
		}
		c = &bitbucketcloudClient{
			ClientID:            consumer.ClientID,
			OAuthToken:          newAccessToken,
			RefreshToken:        refreshToken,
			Cache:               consumer.Cache,
			apiURL:              consumer.apiURL,
			uiURL:               consumer.uiURL,
			DisableStatus:       consumer.disableStatus,
			DisableStatusDetail: consumer.disableStatusDetail,
			proxyURL:            consumer.proxyURL,
		}
		instancesAuthorizedClient[newAccessToken] = c
	} else {
		if !ok {
			c = &bitbucketcloudClient{
				ClientID:            consumer.ClientID,
				OAuthToken:          accessToken,
				RefreshToken:        refreshToken,
				Cache:               consumer.Cache,
				apiURL:              consumer.apiURL,
				uiURL:               consumer.uiURL,
				DisableStatus:       consumer.disableStatus,
				DisableStatusDetail: consumer.disableStatusDetail,
				proxyURL:            consumer.proxyURL,
			}
			instancesAuthorizedClient[accessToken] = c
		}

	}

	return c, nil
}
