package vcs

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

// fakeBitbucket serves the two endpoints getBranchHandler relies on: the branches listing filtered
// by filterText, and the default branch. It counts the calls so that a test can assert that the
// default branch fallback is only paid for when the listing did not answer.
type fakeBitbucket struct {
	listing            string
	defaultBranch      string
	listingCalls       int
	defaultBranchCalls int
}

func (f *fakeBitbucket) server(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/branches/default"):
			f.defaultBranchCalls++
			if f.defaultBranch == "" {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, `{"errors":[{"message":"default branch not found"}]}`)
				return
			}
			fmt.Fprint(w, f.defaultBranch)
		case strings.HasSuffix(r.URL.Path, "/branches"):
			f.listingCalls++
			require.Equal(t, "master", r.URL.Query().Get("filterText"), "the branch name must be pushed to bitbucket as filterText")
			fmt.Fprint(w, f.listing)
		default:
			t.Errorf("unexpected call to the bitbucket fake: %s %s", r.Method, r.URL.String())
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// callGetBranchHandler asks the vcs service for the branch "master" of MANAGER/manager-core-manifests
// on a bitbucketserver backed by the given fake, and returns the recorded response.
func callGetBranchHandler(t *testing.T, s *Service, bitbucketURL string) *httptest.ResponseRecorder {
	vars := map[string]string{
		"name":  "my-bitbucket",
		"owner": "MANAGER",
		"repo":  "manager-core-manifests",
	}
	uri := s.Router.GetRoute("GET", s.getBranchHandler, vars)
	require.NotEmpty(t, uri)

	req := newRequest(t, s, "GET", uri, nil, func(req *http.Request) {
		q := req.URL.Query()
		q.Set("branch", "master")
		// keep the run independent from what a previous run left in redis
		q.Set("noCache", "true")
		req.URL.RawQuery = q.Encode()
	})

	req.Header.Set(sdk.HeaderXVCSType, base64.StdEncoding.EncodeToString([]byte(sdk.VCSTypeBitbucketServer)))
	req.Header.Set(sdk.HeaderXVCSURL, base64.StdEncoding.EncodeToString([]byte(bitbucketURL)))
	req.Header.Set(sdk.HeaderXVCSUsername, base64.StdEncoding.EncodeToString([]byte("cds-bot")))
	req.Header.Set(sdk.HeaderXVCSToken, base64.StdEncoding.EncodeToString([]byte("fake-token")))
	req.Header.Set(sdk.HeaderXVCSProjectKey, base64.StdEncoding.EncodeToString([]byte("MANAGER")))

	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)
	return rec
}

// a page that matches filterText=master but does not contain the master branch itself, which is what
// bitbucket answered during the incident: filterText is a tokenized substring filter and the page is
// truncated at bitbucket's default limit, ordered by modification date.
const listingWithoutMaster = `{"size":2,"isLastPage":false,"values":[
	{"id":"refs/heads/release/master","displayId":"release/master","latestChangeset":"1111111","isDefault":false},
	{"id":"refs/heads/feature/master-fix","displayId":"feature/master-fix","latestChangeset":"2222222","isDefault":false}
]}`

// Test_getBranchHandler_bitbucketserver_defaultBranchMissingFromListing reproduces the incident: the
// branches listing omits master, but master is the default branch of the repository. An existing
// branch must never be reported as absent because the listing did not return it.
func Test_getBranchHandler_bitbucketserver_defaultBranchMissingFromListing(t *testing.T) {
	s, err := newTestService(t)
	require.NoError(t, err)

	fake := &fakeBitbucket{
		listing:       listingWithoutMaster,
		defaultBranch: `{"id":"refs/heads/master","displayId":"master","latestChangeset":"deadbeef","isDefault":true}`,
	}
	srv := fake.server(t)
	defer srv.Close()

	rec := callGetBranchHandler(t, s, srv.URL)
	require.Equal(t, 200, rec.Code, "body: %s", rec.Body.String())

	var branch sdk.VCSBranch
	require.NoError(t, sdk.JSONUnmarshal(rec.Body.Bytes(), &branch))
	assert.Equal(t, "master", branch.DisplayID)
	assert.Equal(t, "deadbeef", branch.LatestCommit)
	assert.True(t, branch.Default, "the branch comes from /branches/default, it is the default branch")
	assert.Equal(t, 1, fake.defaultBranchCalls, "the fallback on /branches/default must have been used")
}

// Test_getBranchHandler_bitbucketserver_branchFoundInListing checks the nominal path is untouched:
// when the listing answers, the default branch fallback must not cost an extra request.
func Test_getBranchHandler_bitbucketserver_branchFoundInListing(t *testing.T) {
	s, err := newTestService(t)
	require.NoError(t, err)

	fake := &fakeBitbucket{
		listing:       `{"size":1,"isLastPage":true,"values":[{"id":"refs/heads/master","displayId":"master","latestChangeset":"cafe1234","isDefault":true}]}`,
		defaultBranch: `{"id":"refs/heads/master","displayId":"master","latestChangeset":"cafe1234","isDefault":true}`,
	}
	srv := fake.server(t)
	defer srv.Close()

	rec := callGetBranchHandler(t, s, srv.URL)
	require.Equal(t, 200, rec.Code, "body: %s", rec.Body.String())

	var branch sdk.VCSBranch
	require.NoError(t, sdk.JSONUnmarshal(rec.Body.Bytes(), &branch))
	assert.Equal(t, "master", branch.DisplayID)
	assert.Equal(t, "cafe1234", branch.LatestCommit)
	assert.Equal(t, 0, fake.defaultBranchCalls, "no need to ask for the default branch when the listing answered")
}

// Test_getBranchHandler_bitbucketserver_branchDoesNotExist checks a genuinely missing branch is still
// a 404, and that the error now carries what is needed to tell "the branch is absent" apart from
// "we looked the wrong way".
func Test_getBranchHandler_bitbucketserver_branchDoesNotExist(t *testing.T) {
	s, err := newTestService(t)
	require.NoError(t, err)

	fake := &fakeBitbucket{
		listing:       listingWithoutMaster,
		defaultBranch: `{"id":"refs/heads/main","displayId":"main","latestChangeset":"deadbeef","isDefault":true}`,
	}
	srv := fake.server(t)
	defer srv.Close()

	rec := callGetBranchHandler(t, s, srv.URL)
	require.Equal(t, 404, rec.Code, "body: %s", rec.Body.String())

	var httpErr sdk.Error
	require.NoError(t, sdk.JSONUnmarshal(rec.Body.Bytes(), &httpErr))
	assert.Equal(t, sdk.ErrNoBranch.ID, httpErr.ID)
	// without these details the message is indistinguishable from a permission problem
	assert.Contains(t, httpErr.From, `filterText="master"`)
	assert.Contains(t, httpErr.From, "lastPage=false")
	assert.Contains(t, httpErr.From, `default branch is "main"`)
	assert.Contains(t, httpErr.From, "release/master")
	assert.Contains(t, httpErr.From, "feature/master-fix")
}

// Test_getBranchHandler_bitbucketserver_danglingDefaultBranch guards the fallback: a repository whose
// configured default branch points to a ref that no longer exists must not make CDS confirm a branch
// on the repository configuration alone.
func Test_getBranchHandler_bitbucketserver_danglingDefaultBranch(t *testing.T) {
	s, err := newTestService(t)
	require.NoError(t, err)

	fake := &fakeBitbucket{
		listing:       listingWithoutMaster,
		defaultBranch: `{"id":"refs/heads/master","displayId":"master","latestChangeset":"","isDefault":true}`,
	}
	srv := fake.server(t)
	defer srv.Close()

	rec := callGetBranchHandler(t, s, srv.URL)
	require.Equal(t, 404, rec.Code, "body: %s", rec.Body.String())

	var httpErr sdk.Error
	require.NoError(t, sdk.JSONUnmarshal(rec.Body.Bytes(), &httpErr))
	assert.Equal(t, sdk.ErrNoBranch.ID, httpErr.ID)
}
