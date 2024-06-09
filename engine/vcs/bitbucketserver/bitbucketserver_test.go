package bitbucketserver

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	ghConsumer := getNewConsumer(t)
	assert.NotNil(t, ghConsumer)
}

func getNewConsumer(t *testing.T) sdk.VCSServer {
	log.Factory = log.NewTestingWrapper(t)
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]
	bitbucketServerUsername := cfg["bitbucketServerUsername"]
	bitbucketServerToken := cfg["bitbucketServerToken"]
	bitbucketServerURL := cfg["bitbucketServerURL"]

	if bitbucketServerUsername == "" && bitbucketServerToken == "" {
		t.Logf("Unable to read bitbucket configuration. Skipping this tests.")
		t.SkipNow()
	}

	cache, err := cache.New(redisHost, redisPassword, 0, 30)
	if err != nil {
		t.Fatalf("Unable to init cache (%s): %v", redisHost, err)
	}

	return New(bitbucketServerURL, "", "", "", cache, bitbucketServerUsername, bitbucketServerToken)
}

func getAuthorizedClient(t *testing.T) sdk.VCSAuthorizedClient {
	log.Factory = log.NewTestingWrapper(t)
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]

	bitbucketServerUsername := cfg["bitbucketServerUsername"]
	bitbucketServerToken := cfg["bitbucketServerToken"]
	bitbucketServerURL := cfg["bitbucketServerURL"]

	if bitbucketServerUsername == "" && bitbucketServerToken == "" {
		t.Logf("Unable to read bitbucket configuration. Skipping this tests.")
		t.SkipNow()
	}

	cache, err := cache.New(redisHost, redisPassword, 0, 30)
	if err != nil {
		t.Fatalf("Unable to init cache (%s): %v", redisHost, err)
	}

	consumer := New(bitbucketServerURL, "", "", "", cache, bitbucketServerUsername, bitbucketServerToken)

	vcsAuth := sdk.VCSAuth{
		Type:     sdk.VCSTypeBitbucketServer,
		Username: bitbucketServerUsername,
		Token:    bitbucketServerToken,
		URL:      bitbucketServerURL,
	}
	cli, err := consumer.GetAuthorizedClient(context.Background(), vcsAuth)
	require.NoError(t, err)
	return cli
}
