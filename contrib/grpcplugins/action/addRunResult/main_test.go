package main

import (
	"sort"
	"testing"

	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/glob"
	"github.com/stretchr/testify/require"
)

func TestXxx(t *testing.T) {
	payload := `{
		"type": "V2WorkflowRunResultStaticFilesDetail",
		"data": {
		  "name": "hello",
		  "artifactory_url": "fsamin-default-static/test-static-files/",
		  "public_url": "https://rtstatic.ovhcloud.tools/fsamin/default/test-static-files"
		}
	  }`

	var detail sdk.V2WorkflowRunResultDetail

	err := sdk.JSONUnmarshal([]byte(payload), &detail)
	require.NoError(t, err)

}

func TestContainsGlob(t *testing.T) {
	require.False(t, containsGlob("pool/my-package_1.0.0_amd64.deb"))
	require.False(t, containsGlob("mychart/0.1.0-123"))
	require.True(t, containsGlob("pool/*.deb"))
	require.True(t, containsGlob("services/*/*"))
	require.True(t, containsGlob("pool/package_1.0.?_amd64.deb"))
	require.True(t, containsGlob("pool/[ab]*.deb"))
}

func TestGlobSupportedType(t *testing.T) {
	require.False(t, globSupportedType(sdk.V2WorkflowRunResultTypeDocker))
	require.False(t, globSupportedType(sdk.V2WorkflowRunResultTypeStaticFiles))
	require.True(t, globSupportedType(sdk.V2WorkflowRunResultTypeDebian))
	require.True(t, globSupportedType(sdk.V2WorkflowRunResultTypeGeneric))
	require.True(t, globSupportedType(sdk.V2WorkflowRunResultTypeOCI))
	require.True(t, globSupportedType(sdk.V2WorkflowRunResultTypeConan))
}

func TestStaticPrefix(t *testing.T) {
	require.Equal(t, "pool", staticPrefix("pool/*.deb"))
	require.Equal(t, "myns/app", staticPrefix("myns/app/*"))
	require.Equal(t, "", staticPrefix("*/*"))
	require.Equal(t, "", staticPrefix("**"))
	require.Equal(t, "mirror", staticPrefix("mirror/**"))
	// multi-patterns: common folder prefix of positive patterns
	require.Equal(t, "pool", staticPrefix("pool/a*.deb pool/b*.deb"))
	require.Equal(t, "", staticPrefix("pool/*.deb dist/*.deb"))
	// exclusion patterns don't widen the search
	require.Equal(t, "mirror", staticPrefix("mirror/** !**/sha256:*"))
	// wildcard inside a segment: the segment is not part of the prefix
	require.Equal(t, "pool", staticPrefix("pool/sub*/file.deb"))
}

func TestRepoCriteria(t *testing.T) {
	require.Equal(t, `{"repo":{"$match":"proj-debian-*"}}`, repoCriteria("proj-debian"))
	require.Equal(t, `{"$or":[{"repo":{"$match":"proj-cds-*"}},{"repo":{"$match":"proj-generic-*"}}]}`, repoCriteria("proj-cds"))
}

func TestDeriveCandidate(t *testing.T) {
	// file-based: one file = one candidate, repo root included
	require.Equal(t, "pool/a.deb", deriveCandidate(grpcplugins.SearchResult{Path: "pool", Name: "a.deb"}, sdk.V2WorkflowRunResultTypeDebian))
	require.Equal(t, "a.deb", deriveCandidate(grpcplugins.SearchResult{Path: ".", Name: "a.deb"}, sdk.V2WorkflowRunResultTypeDebian))

	// oci: the candidate is the folder holding the manifest, whatever the name depth
	require.Equal(t, "services/core-platform/0.0.0-dev.50", deriveCandidate(grpcplugins.SearchResult{Path: "services/core-platform/0.0.0-dev.50", Name: "manifest.json"}, sdk.V2WorkflowRunResultTypeOCI))
	require.Equal(t, "mirror/registry/postgres/sha256:1899ab", deriveCandidate(grpcplugins.SearchResult{Path: "mirror/registry/postgres/sha256:1899ab", Name: "manifest.json"}, sdk.V2WorkflowRunResultTypeOCI))

	// conan: the candidate is the revision folder, parent of export/
	require.Equal(t, "_/abseil/20250127.0/_/e0dcc4b8", deriveCandidate(grpcplugins.SearchResult{Path: "_/abseil/20250127.0/_/e0dcc4b8/export", Name: "conanmanifest.txt"}, sdk.V2WorkflowRunResultTypeConan))
	// conanmanifest.txt of binary packages are ignored
	require.Equal(t, "", deriveCandidate(grpcplugins.SearchResult{Path: "_/abseil/20250127.0/_/e0dcc4b8/package/37fd2c/28874d", Name: "conanmanifest.txt"}, sdk.V2WorkflowRunResultTypeConan))
}

// TestGlobSelection covers the candidate filtering as done by enumerateGlobMatches, on
// synthetic search results mimicking the layouts audited on real repositories.
func TestGlobSelection(t *testing.T) {
	filterCandidates := func(results []grpcplugins.SearchResult, resultType sdk.V2WorkflowRunResultType, pattern string) []string {
		g := glob.New(pattern)
		seen := map[string]struct{}{}
		var out []string
		for _, r := range results {
			candidate := deriveCandidate(r, resultType)
			if candidate == "" {
				continue
			}
			if _, ok := seen[candidate]; ok {
				continue
			}
			seen[candidate] = struct{}{}
			m, err := g.MatchString(candidate)
			require.NoError(t, err)
			if m != nil {
				out = append(out, candidate)
			}
		}
		sort.Strings(out) // enumerateGlobMatches sorts the same way
		return out
	}

	debianResults := []grpcplugins.SearchResult{
		{Path: "pool", Name: "pkg-a_1.0.0_amd64.deb"},
		{Path: "pool", Name: "pkg-a-dbg_1.0.0_amd64.deb"},
		{Path: "pool", Name: "pkg-a_1.0.0_amd64.deb"}, // same file in another maturity: deduplicated
		{Path: "pool/sub", Name: "pkg-b_1.0.0_amd64.deb"},
		{Path: "other", Name: "pkg-c_1.0.0_amd64.deb"},
	}
	require.Equal(t, []string{"pool/pkg-a-dbg_1.0.0_amd64.deb", "pool/pkg-a_1.0.0_amd64.deb"},
		filterCandidates(debianResults, sdk.V2WorkflowRunResultTypeDebian, "pool/*.deb"))
	require.Equal(t, []string{"pool/pkg-a_1.0.0_amd64.deb"},
		filterCandidates(debianResults, sdk.V2WorkflowRunResultTypeDebian, "pool/*.deb !pool/*-dbg*"))
	require.Equal(t, []string{"pool/pkg-a-dbg_1.0.0_amd64.deb", "pool/pkg-a_1.0.0_amd64.deb", "pool/sub/pkg-b_1.0.0_amd64.deb"},
		filterCandidates(debianResults, sdk.V2WorkflowRunResultTypeDebian, "pool/**"))

	ociResults := []grpcplugins.SearchResult{
		{Path: "services/core-platform/0.0.0-dev.50", Name: "manifest.json"},
		{Path: "services/core-platform/0.0.0-dev.51", Name: "manifest.json"},
		{Path: "mirror/api-exposition/gateway/1.46.0", Name: "manifest.json"},
		{Path: "mirror/registry/postgres/sha256:1899ab", Name: "manifest.json"},
	}
	require.Equal(t, []string{"services/core-platform/0.0.0-dev.50", "services/core-platform/0.0.0-dev.51"},
		filterCandidates(ociResults, sdk.V2WorkflowRunResultTypeOCI, "services/*/*"))
	require.Equal(t, []string{"mirror/api-exposition/gateway/1.46.0"},
		filterCandidates(ociResults, sdk.V2WorkflowRunResultTypeOCI, "mirror/** !**/sha256:*"))

	conanResults := []grpcplugins.SearchResult{
		{Path: "_/abseil/20250127.0/_/e0dcc4b8/export", Name: "conanmanifest.txt"},
		{Path: "_/abseil/20250127.0/_/e0dcc4b8/package/37fd2c/28874d", Name: "conanmanifest.txt"},
		{Path: "_/cmake/3.31.11/_/f325c933/export", Name: "conanmanifest.txt"},
	}
	require.Equal(t, []string{"_/abseil/20250127.0/_/e0dcc4b8", "_/cmake/3.31.11/_/f325c933"},
		filterCandidates(conanResults, sdk.V2WorkflowRunResultTypeConan, "_/*/*/_/*"))
	require.Equal(t, []string{"_/abseil/20250127.0/_/e0dcc4b8"},
		filterCandidates(conanResults, sdk.V2WorkflowRunResultTypeConan, "_/abseil/*/_/*"))
}
