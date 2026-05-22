package sdk_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGitContextVCSAPI is an integration test that verifies the git context fields
// (server_url, server_type, username, token) can be used to successfully call a VCS API.
//
// Required environment variables:
//   - GITEA_URL: base URL of the Gitea instance (e.g., http://172.17.0.2:3000)
//   - GITEA_USERNAME: Gitea username
//   - GITEA_PASSWORD: Gitea password/token
//
// Run with: GITEA_URL=... GITEA_USERNAME=... GITEA_PASSWORD=... go test -v -run TestGitContextVCSAPI ./sdk/
func TestGitContextVCSAPI(t *testing.T) {
	giteaURL := os.Getenv("GITEA_URL")
	giteaUsername := os.Getenv("GITEA_USERNAME")
	giteaPassword := os.Getenv("GITEA_PASSWORD")

	if giteaURL == "" || giteaUsername == "" || giteaPassword == "" {
		t.Skip("Skipping integration test: GITEA_URL, GITEA_USERNAME, GITEA_PASSWORD must be set")
	}

	// Simulate what buildRunContext would produce
	gitContext := sdk.GitContext{
		Server:     "my-gitea-server",
		ServerURL:  giteaURL,
		ServerType: sdk.VCSTypeGitea,
		Username:   giteaUsername,
		Token:      giteaPassword,
	}

	// Verify JSON serialization includes new fields
	data, err := json.Marshal(gitContext)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))
	assert.Equal(t, giteaURL, m["server_url"])
	assert.Equal(t, "gitea", m["server_type"])
	assert.Equal(t, giteaUsername, m["username"])

	// Call the VCS API using the context values — exactly as a workflow script would
	t.Run("call_gitea_user_api", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/user", gitContext.ServerURL)
		req, err := http.NewRequest("GET", url, nil)
		require.NoError(t, err)
		req.SetBasicAuth(gitContext.Username, gitContext.Token)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "VCS API should return 200 with valid credentials from git context")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var user map[string]interface{}
		require.NoError(t, json.Unmarshal(body, &user))
		assert.Equal(t, giteaUsername, user["login"], "API response login should match git.username")
		t.Logf("Successfully authenticated as %q against %s (type=%s)", user["login"], gitContext.ServerURL, gitContext.ServerType)
	})

	// Create a test repo, then call repos API using git context fields
	t.Run("call_gitea_repo_api", func(t *testing.T) {
		repoName := "test-git-context-api"

		// Create repo
		createURL := fmt.Sprintf("%s/api/v1/user/repos", gitContext.ServerURL)
		reqBody := fmt.Sprintf(`{"name": %q, "auto_init": true}`, repoName)
		req, err := http.NewRequest("POST", createURL, strings.NewReader(reqBody))
		require.NoError(t, err)
		req.SetBasicAuth(gitContext.Username, gitContext.Token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// 201 = created, 409 = already exists
		if resp.StatusCode == http.StatusConflict {
			t.Log("Repo already exists, continuing")
		} else {
			require.Equal(t, http.StatusCreated, resp.StatusCode, "Create repo failed: %s", string(body))
		}

		// Now query the repo using the same pattern as in the workflow script:
		// curl -u "${{ git.username }}:${{ git.token }}" "${{ git.server_url }}/api/v1/repos/${{ git.repository }}"
		repository := fmt.Sprintf("%s/%s", gitContext.Username, repoName)
		repoURL := fmt.Sprintf("%s/api/v1/repos/%s", gitContext.ServerURL, repository)

		req2, err := http.NewRequest("GET", repoURL, nil)
		require.NoError(t, err)
		req2.SetBasicAuth(gitContext.Username, gitContext.Token)

		resp2, err := http.DefaultClient.Do(req2)
		require.NoError(t, err)
		defer resp2.Body.Close()

		require.Equal(t, http.StatusOK, resp2.StatusCode, "VCS API repos endpoint should return 200")

		body2, err := io.ReadAll(resp2.Body)
		require.NoError(t, err)

		var repo map[string]interface{}
		require.NoError(t, json.Unmarshal(body2, &repo))
		assert.Equal(t, repository, repo["full_name"], "repo full_name should match git.repository")
		t.Logf("Successfully queried repo %q via %s/api/v1/repos/%s", repo["full_name"], gitContext.ServerURL, repository)

		// Cleanup: delete the test repo
		deleteURL := fmt.Sprintf("%s/api/v1/repos/%s", gitContext.ServerURL, repository)
		reqDel, _ := http.NewRequest("DELETE", deleteURL, nil)
		reqDel.SetBasicAuth(gitContext.Username, gitContext.Token)
		respDel, err := http.DefaultClient.Do(reqDel)
		if err == nil {
			respDel.Body.Close()
		}
	})
}
