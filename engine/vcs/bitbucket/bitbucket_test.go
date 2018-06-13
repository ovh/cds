package bitbucket

import (
	"testing"

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

	ghConsummer := New(consumerKey, []byte(consumerPrivateKey), url, "", "", "", cache, true)
	return ghConsummer
}

func newAuthorizedClient(t *testing.T) (string, string) {
	log.SetLogger(t)
	cfg := test.LoadTestingConf(t)
	consumerKey := cfg["bitbucketConsumerKey"]
	privateKey := cfg["bitbucketPrivateKey"]
	accessToken := cfg["bitbucketAccessToken"]
	accessTokenSecret := cfg["bitbucketAccessTokenSecret"]

	url := cfg["bitbucketURL"]
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]

	if consumerKey == "" && privateKey == "" {
		t.Logf("Unable to read github configuration. Skipping this tests.")
		t.SkipNow()
	}

	cache, err := cache.New(redisHost, redisPassword, 30)
	if err != nil {
		t.Fatalf("Unable to init cache (%s): %v", redisHost, err)
	}

	ghConsummer := New(consumerKey, []byte(privateKey), url, "", "", "", cache, true)
	token, secret, err := ghConsummer.AuthorizeToken(accessToken, accessTokenSecret)
	test.NoError(t, err)

	t.Logf("token: %s", token)
	t.Logf("secret: %s", secret)

	return token, secret
}

func getAuthorizedClient(t *testing.T) sdk.VCSAuthorizedClient {
	log.SetLogger(t)
	cfg := test.LoadTestingConf(t)
	consumerKey := cfg["bitbucketConsumerKey"]
	privateKey := cfg["bitbucketPrivateKey"]
	token := cfg["bitbucketToken"]
	secret := cfg["bitbucketSecret"]
	url := cfg["bitbucketURL"]
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]

	if consumerKey == "" && privateKey == "" {
		t.Logf("Unable to read github configuration. Skipping this tests.")
		t.SkipNow()
	}

	cache, err := cache.New(redisHost, redisPassword, 30)
	if err != nil {
		t.Fatalf("Unable to init cache (%s): %v", redisHost, err)
	}

	consumer := New(consumerKey, []byte(privateKey), url, "", "", "", cache, true)
	cli, err := consumer.GetAuthorizedClient(token, secret)
	test.NoError(t, err)
	return cli
}

func TestClientAuthorizeToken(t *testing.T) {
	consumer := getNewConsumer(t)
	token, url, err := consumer.AuthorizeRedirect()
	t.Logf("token: %s", token)
	assert.NotEmpty(t, token)

	t.Logf("url: %s", url)
	assert.NotEmpty(t, url)
	test.NoError(t, err)

	err = browser.OpenURL(url)
	test.NoError(t, err)
}

func TestAuthorizedClient(t *testing.T) {
	bitbucketClient := getAuthorizedClient(t)
	assert.NotNil(t, bitbucketClient)
}
