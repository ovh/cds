package bitbucketserver

import (
	"context"
	"os"
	"testing"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/pkg/browser"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	ghConsummer := getNewConsumer(t)
	assert.NotNil(t, ghConsummer)
}

func getNewConsumer(t *testing.T) sdk.VCSServer {
	log.Factory = log.NewTestingWrapper(t)
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)
	consumerKey := cfg["bitbucketConsumerKey"]
	consumerPrivateKey := cfg["bitbucketPrivateKey"]
	url := cfg["bitbucketURL"]
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]

	if consumerKey == "" && consumerPrivateKey == "" {
		t.Logf("Unable to read bitbucket configuration. Skipping this tests.")
		t.SkipNow()
	}

	cache, err := cache.New(redisHost, redisPassword, 30)
	if err != nil {
		t.Fatalf("Unable to init cache (%s): %v", redisHost, err)
	}

	ghConsummer := NewDeprecated(consumerKey, []byte(consumerPrivateKey), url, "", "", "", "", "", cache, true)
	return ghConsummer
}

func getAuthorizedClient(t *testing.T) sdk.VCSAuthorizedClient {
	log.Factory = log.NewTestingWrapper(t)
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)
	consumerKey := cfg["bitbucketConsumerKey"]
	privateKey := cfg["bitbucketPrivateKey"]
	token := cfg["bitbucketToken"]
	secret := cfg["bitbucketSecret"]
	url := cfg["bitbucketURL"]
	username := cfg["bitbucketUsername"]
	password := cfg["bitbucketPassword"]
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]

	if consumerKey == "" && privateKey == "" {
		t.Logf("Unable to read bitbucket configuration. Skipping this tests.")
		t.SkipNow()
	}

	cache, err := cache.New(redisHost, redisPassword, 30)
	if err != nil {
		t.Fatalf("Unable to init cache (%s): %v", redisHost, err)
	}

	consumer := NewDeprecated(consumerKey, []byte(privateKey), url, "", "", "", username, password, cache, true)

	vcsAuth := sdk.VCSAuth{
		AccessToken:       token,
		AccessTokenSecret: secret,
	}
	cli, err := consumer.GetAuthorizedClient(context.Background(), vcsAuth)
	require.NoError(t, err)
	return cli
}

func TestClientAuthorizeRedirect(t *testing.T) {
	consumer := getNewConsumer(t)
	token, url, err := consumer.AuthorizeRedirect(context.Background())
	t.Logf("token: %s", token)
	assert.NotEmpty(t, token)

	t.Logf("url: %s", url)
	assert.NotEmpty(t, url)
	require.NoError(t, err)

	err = browser.OpenURL(url)
	require.NoError(t, err)
}

func TestClientAuthorizeToken(t *testing.T) {
	token := os.Getenv("TOKEN")
	verifier := os.Getenv("VERIFIER")

	if token == "" || verifier == "" {
		t.SkipNow()
	}

	consumer := getNewConsumer(t)
	accesstoken, accesstokenSecret, err := consumer.AuthorizeToken(context.Background(), token, verifier)
	require.NoError(t, err)
	assert.NotEmpty(t, accesstoken)
	assert.NotEmpty(t, accesstokenSecret)

	t.Logf("accesstoken: %s", accesstoken)
	t.Logf("accesstokenSecret: %s", accesstokenSecret)
}

func TestAuthorizedClient(t *testing.T) {
	bitbucketClient := getAuthorizedClient(t)
	assert.NotNil(t, bitbucketClient)
}
