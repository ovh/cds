package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/pkg/browser"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/vcs"
	"github.com/ovh/cds/sdk/log"
)

// TestNew needs githubClientID and githubClientSecret
func TestNewClient(t *testing.T) {
	ghConsummer := getNewClient(t)
	assert.NotNil(t, ghConsummer)
}

func getNewClient(t *testing.T) vcs.Server {
	log.SetLogger(t)
	cfg := test.LoadTestingConf(t)
	clientID := cfg["githubClientID"]
	clientSecret := cfg["githubClientSecret"]
	redisHost := cfg["redisHost"]
	redisPassword := cfg["redisPassword"]

	if clientID == "" && clientSecret == "" {
		t.Logf("Unable to read github configuration. Skipping this tests.")
		t.SkipNow()
	}

	cache, err := cache.New(redisHost, redisPassword, 30)
	if err != nil {
		t.Fatalf("Unable to init cache (%s): %v", redisHost, err)
	}

	ghConsummer := New(clientID, clientSecret, cache)
	return ghConsummer
}

func TestClientAuthorizeToken(t *testing.T) {
	ghConsummer := getNewClient(t)
	token, url, err := ghConsummer.AuthorizeRedirect()
	t.Logf("token: %s", token)
	t.Logf("url: %s", url)
	assert.NotEmpty(t, token)
	assert.NotEmpty(t, url)
	test.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	out := make(chan http.Request, 1)

	go callbackServer(ctx, t, out)

	err = browser.OpenURL(url)
	test.NoError(t, err)

	r, ok := <-out
	t.Logf("Chan request closed? %v", !ok)
	t.Logf("OAuth request 2: %+v", r)
	assert.NotNil(t, r)

	cberr := r.FormValue("error")
	errDescription := r.FormValue("error_description")
	errURI := r.FormValue("error_uri")

	assert.Empty(t, cberr)
	assert.Empty(t, errDescription)
	assert.Empty(t, errURI)

	code := r.FormValue("code")
	state := r.FormValue("state")

	assert.NotEmpty(t, code)
	assert.NotEmpty(t, state)

	accessToken, accessTokenSecret, err := ghConsummer.AuthorizeToken(state, code)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, accessTokenSecret)
	test.NoError(t, err)

	ghClient, err := ghConsummer.GetAuthorizedClient(accessToken, accessTokenSecret)
	test.NoError(t, err)
	assert.NotNil(t, ghClient)

}

func callbackServer(ctx context.Context, t *testing.T, out chan http.Request) {
	srv := &http.Server{Addr: ":8081"}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		out <- *r
		io.WriteString(w, "Yeah !\n")
		fmt.Println("Handler")
	})

	go func() {
		fmt.Println("Starting server")
		if err := srv.ListenAndServe(); err != nil {
			// cannot panic, because this probably is an intentional close
			t.Logf("Httpserver: ListenAndServe() error: %s", err)
		}
		close(out)
	}()

	<-ctx.Done()
	fmt.Println("Stopping server")
	srv.Shutdown(ctx)
}
