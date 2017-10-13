package bitbucket

import (
	"fmt"
	"testing"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/pkg/browser"
	"github.com/stretchr/testify/assert"
)

// TestNew needs githubClientID and githubClientSecret
func TestNewClient(t *testing.T) {
	ghConsummer := getNewConsumer(t)
	assert.NotNil(t, ghConsummer)
}

func getNewConsumer(t *testing.T) sdk.VCSServer {
	log.SetLogger(t)
	cfg := test.LoadTestingConf(t)
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

	ghConsummer := New(consumerKey, []byte(consumerPrivateKey), url, cache)
	return ghConsummer
}

func TestClientAuthorizeToken(t *testing.T) {
	consumer := getNewConsumer(t)
	token, url, err := consumer.AuthorizeRedirect()
	t.Logf("token: %s", token)
	t.Logf("url: %s", url)
	assert.NotEmpty(t, token)
	assert.NotEmpty(t, url)
	test.NoError(t, err)

	err = browser.OpenURL(url)
	test.NoError(t, err)

	fmt.Println("Enter verification code: ")
	code := cli.ReadLine()

	assert.NotEmpty(t, token)
	assert.NotEmpty(t, code)

	accessToken, accessTokenSecret, err := consumer.AuthorizeToken(token, code)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, accessTokenSecret)
	test.NoError(t, err)

	t.Logf("Token is %s", accessToken)
	t.Logf("TokenSecret is %s", accessTokenSecret)

	bitbucketClient, err := consumer.GetAuthorizedClient(accessToken, accessTokenSecret)
	test.NoError(t, err)
	assert.NotNil(t, bitbucketClient)
}
