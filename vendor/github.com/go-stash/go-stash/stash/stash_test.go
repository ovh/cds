package stash

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

var (
	testURL      string
	testProject  string
	testRepo     string
	testUser     string
	testPassword string
	consumerKey  string
	privateKey   string
	accessKey    string
	accessSecret string
	hookKey      string
	client       *Client
)

func init() {
	testURL = os.Getenv("STASH_URL")
	testProject = os.Getenv("STASH_PROJECT")
	testRepo = os.Getenv("STASH_REPO")
	testUser = os.Getenv("STASH_USER")
	testPassword = os.Getenv("STASH_PASSWORD")
	consumerKey = os.Getenv("STASH_CONSUMER_KEY")
	privateKey = os.Getenv("STASH_PRIVATE_KEY")
	accessKey = os.Getenv("STASH_ACCESS_TOKEN")
	accessSecret = os.Getenv("STASH_ACCESS_SECRET")
	hookKey = os.Getenv("STASH_HOOK")

	switch {
	case len(testURL) == 0:
		panic(errors.New("must set the STASH_URL environment variable"))
	case len(testProject) == 0:
		panic(errors.New("must set the STASH_PROJECT environment variable"))
	case len(testRepo) == 0:
		panic(errors.New("must set the STASH_REPO environment variable"))
	case len(testUser) == 0:
		panic(errors.New("must set the STASH_USER environment variable"))
	case len(testPassword) == 0:
		panic(errors.New("must set the STASH_PASSWORD environment variable"))
	case len(consumerKey) == 0:
		panic(errors.New("must set the STASH_CONSUMER_KEY environment variable"))
	case len(privateKey) == 0:
		panic(errors.New("must set the STASH_PRIVATE_KEY environment variable"))
	case len(accessKey) == 0:
		panic(errors.New("must set the STASH_ACCESS_TOKEN environment variable"))
	case len(accessSecret) == 0:
		panic(errors.New("must set the STASH_ACCESS_SECRET environment variable"))
	case len(hookKey) == 0:
		panic(errors.New("must set the STASH_HOOK environment variable"))
	}

	client = New(
		testURL,
		consumerKey,
		accessKey,
		accessSecret,
		privateKey,
	)
}

func TestGetFullApiUrl(t *testing.T) {
	url := client.GetFullApiUrl("core")
	if url != fmt.Sprintf("%s/rest/api/1.0", testURL) {
		t.Errorf("Core API URL is invalid, got: ", url)
	}
}
