package main

import (
	"encoding/json"
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
	require.False(t, globSupportedType(sdk.V2WorkflowRunResultTypeStaticFiles))
	require.True(t, globSupportedType(sdk.V2WorkflowRunResultTypeDocker))
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
	// docker patterns are image references, staticPrefix only splits on "/": see
	// TestDockerPatternToPath for the rewrite that puts the image folder in the prefix
	require.Equal(t, "ovhcom", staticPrefix("ovhcom/*:*-1234"))
	require.Equal(t, "ovhcom", staticPrefix("ovhcom/venom:*"))
}

func TestRepoCriteria(t *testing.T) {
	require.Equal(t, `{"repo":{"$match":"proj-debian-*"}}`, repoCriteria("proj-debian"))
	require.Equal(t, `{"$or":[{"repo":{"$match":"proj-cds-*"}},{"repo":{"$match":"proj-generic-*"}}]}`, repoCriteria("proj-cds"))
}

func TestDockerRepoCriteria(t *testing.T) {
	require.Equal(t, `{"repo":{"$eq":"proj-docker-snapshot"}}`, dockerRepoCriteria("proj-docker", "snapshot"))
}

// TestDockerPatternToPath locks the rewrite feeding staticPrefix: the ":" of an image
// reference is a "/" in the artifactory layout, and the image folder belongs to the prefix.
func TestDockerPatternToPath(t *testing.T) {
	require.Equal(t, "busybox/*", dockerPatternToPath("busybox:*"))
	require.Equal(t, "ovhcom/venom/*", dockerPatternToPath("ovhcom/venom:*"))
	require.Equal(t, "ovhcom/*/*-1234", dockerPatternToPath("ovhcom/*:*-1234"))
	// no tag part to rewrite
	require.Equal(t, "ovhcom/**", dockerPatternToPath("ovhcom/**"))
	// exclusions keep their "!" and are rewritten too
	require.Equal(t, "ovhcom/venom/* !ovhcom/venom/latest-*", dockerPatternToPath("ovhcom/venom:* !ovhcom/venom:latest-*"))

	// without the rewrite, "busybox:*" has no static folder and the AQL scans the whole repo
	require.Equal(t, "", staticPrefix("busybox:*"))
	require.Equal(t, "busybox", staticPrefix(dockerPatternToPath("busybox:*")))
	require.Equal(t, "ovhcom/venom", staticPrefix(dockerPatternToPath("ovhcom/venom:*")))
	require.Equal(t, "ovhcom", staticPrefix(dockerPatternToPath("ovhcom/*:*-1234")))
}

// TestDockerArchPath locks which items the enumeration keeps apart: the per-architecture
// manifests of a manifest list, so the detail is built without going back to artifactory.
func TestDockerArchPath(t *testing.T) {
	require.Equal(t, "ovhcom/venom/sha256:1899ab", dockerArchPath(grpcplugins.SearchResult{Path: "ovhcom/venom/sha256:1899ab", Name: "manifest.json"}))
	require.Equal(t, "busybox/sha256:1899ab", dockerArchPath(grpcplugins.SearchResult{Path: "busybox/sha256:1899ab", Name: "manifest.json"}))
	// tag folders are candidates, not architectures
	require.Equal(t, "", dockerArchPath(grpcplugins.SearchResult{Path: "ovhcom/venom/v1.3.0-1234", Name: "manifest.json"}))
	require.Equal(t, "", dockerArchPath(grpcplugins.SearchResult{Path: "ovhcom/venom/latest-1234", Name: "list.manifest.json"}))
	require.Equal(t, "", dockerArchPath(grpcplugins.SearchResult{Path: "venom", Name: "manifest.json"}))
	require.Equal(t, "", dockerArchPath(grpcplugins.SearchResult{Path: ".", Name: "manifest.json"}))
}

func TestDeriveCandidate(t *testing.T) {
	// file-based: one file = one candidate, repo root included
	require.Equal(t, "pool/a.deb", deriveCandidate(grpcplugins.SearchResult{Path: "pool", Name: "a.deb"}, sdk.V2WorkflowRunResultTypeDebian))
	require.Equal(t, "a.deb", deriveCandidate(grpcplugins.SearchResult{Path: ".", Name: "a.deb"}, sdk.V2WorkflowRunResultTypeDebian))

	// oci: the candidate is the folder holding the manifest, whatever the name depth
	require.Equal(t, "services/core-platform/0.0.0-dev.50", deriveCandidate(grpcplugins.SearchResult{Path: "services/core-platform/0.0.0-dev.50", Name: "manifest.json"}, sdk.V2WorkflowRunResultTypeOCI))
	require.Equal(t, "mirror/registry/postgres/sha256:1899ab", deriveCandidate(grpcplugins.SearchResult{Path: "mirror/registry/postgres/sha256:1899ab", Name: "manifest.json"}, sdk.V2WorkflowRunResultTypeOCI))

	// docker: the same folder as oci, rebuilt into an image reference
	require.Equal(t, "ovhcom/venom:v1.3.0-1234", deriveCandidate(grpcplugins.SearchResult{Path: "ovhcom/venom/v1.3.0-1234", Name: "manifest.json"}, sdk.V2WorkflowRunResultTypeDocker))
	require.Equal(t, "ovhcom/venom:latest-1234", deriveCandidate(grpcplugins.SearchResult{Path: "ovhcom/venom/latest-1234", Name: "list.manifest.json"}, sdk.V2WorkflowRunResultTypeDocker))
	// digest folders are not taggable images
	require.Equal(t, "", deriveCandidate(grpcplugins.SearchResult{Path: "ovhcom/venom/sha256:1899ab", Name: "manifest.json"}, sdk.V2WorkflowRunResultTypeDocker))
	// a folder without an <image>/<tag> shape has no image reference
	require.Equal(t, "", deriveCandidate(grpcplugins.SearchResult{Path: "venom", Name: "manifest.json"}, sdk.V2WorkflowRunResultTypeDocker))

	// conan: the candidate is the revision folder, parent of export/
	require.Equal(t, "_/abseil/20250127.0/_/e0dcc4b8", deriveCandidate(grpcplugins.SearchResult{Path: "_/abseil/20250127.0/_/e0dcc4b8/export", Name: "conanmanifest.txt"}, sdk.V2WorkflowRunResultTypeConan))
	// conanmanifest.txt of binary packages are ignored
	require.Equal(t, "", deriveCandidate(grpcplugins.SearchResult{Path: "_/abseil/20250127.0/_/e0dcc4b8/package/37fd2c/28874d", Name: "conanmanifest.txt"}, sdk.V2WorkflowRunResultTypeConan))
}

func TestVirtualRepoFor(t *testing.T) {
	require.Equal(t, "proj-debian", virtualRepoFor("proj-debian-snapshot", "proj-debian"))
	require.Equal(t, "proj-cds", virtualRepoFor("proj-cds-release", "proj-cds"))
	// -cds also enumerates the -generic family
	require.Equal(t, "proj-generic", virtualRepoFor("proj-generic-snapshot", "proj-cds"))
	require.Equal(t, "", virtualRepoFor("other-debian-snapshot", "proj-debian"))
	require.Equal(t, "", virtualRepoFor("proj-debian2-snapshot", "proj-debian"))
}

func TestFileInfoFromItem(t *testing.T) {
	item := grpcplugins.SearchResult{
		Repo: "proj-debian-snapshot", Path: "pool", Name: "a.deb",
		ActualMd5: "md5", ActualSha1: "sha1", Sha256: "sha256", Size: 42,
	}
	fi := fileInfoFromItem(grpcplugins.ArtifactoryConfig{URL: "https://rt.local/artifactory"}, "proj-debian", item)
	require.Equal(t, "/pool/a.deb", fi.Path)
	require.Equal(t, "42", fi.Size)
	require.Equal(t, "https://rt.local/artifactory/api/storage/proj-debian/pool/a.deb", fi.URI)
	require.Equal(t, "https://rt.local/artifactory/proj-debian/pool/a.deb", fi.DownloadURI)
	require.Equal(t, "md5", fi.Checksums.Md5)
	require.Equal(t, "sha1", fi.Checksums.Sha1)
	require.Equal(t, "sha256", fi.Checksums.Sha256)

	// file at the repository root
	fi = fileInfoFromItem(grpcplugins.ArtifactoryConfig{URL: "https://rt.local/artifactory/"}, "proj-debian", grpcplugins.SearchResult{Path: ".", Name: "b.deb"})
	require.Equal(t, "/b.deb", fi.Path)
	require.Equal(t, "https://rt.local/artifactory/proj-debian/b.deb", fi.DownloadURI)
}

func TestPropertiesMap(t *testing.T) {
	props := propertiesMap([]grpcplugins.SearchResultProperty{
		{Key: "deb.distribution", Value: "focal"},
		{Key: "deb.distribution", Value: "jammy"},
		{Key: "deb.component", Value: "main"},
	})
	require.Equal(t, map[string][]string{
		"deb.distribution": {"focal", "jammy"},
		"deb.component":    {"main"},
	}, props)
}

// TestSearchResultResponseTruncation locks the JSON path of the hard-limit marker: artifactory
// reports it under the range object, not at the top level.
func TestSearchResultResponseTruncation(t *testing.T) {
	var res grpcplugins.SearchResultResponse
	require.NoError(t, json.Unmarshal([]byte(`{
		"results": [{"repo": "proj-debian-snapshot", "path": "pool", "name": "a.deb"}],
		"range": {"start_pos": 0, "end_pos": 1, "total": 1, "notification": "AQL query reached the search hard limit, results are trimmed."}
	}`), &res))
	require.Len(t, res.Results, 1)
	require.Equal(t, "AQL query reached the search hard limit, results are trimmed.", res.Range.Notification)

	var complete grpcplugins.SearchResultResponse
	require.NoError(t, json.Unmarshal([]byte(`{"results": [], "range": {"start_pos": 0, "end_pos": 0, "total": 0}}`), &complete))
	require.Equal(t, "", complete.Range.Notification)
}

// TestGlobSelection covers the candidate filtering as done by enumerateGlobMatches, on
// synthetic search results mimicking the layouts audited on real repositories.
func TestGlobSelection(t *testing.T) {
	filterCandidates := func(results []grpcplugins.SearchResult, resultType sdk.V2WorkflowRunResultType, pattern string) []string {
		g := glob.New(pattern)
		seen := map[string]struct{}{}
		var out []string
		for _, r := range results {
			if resultType == sdk.V2WorkflowRunResultTypeDocker && dockerArchPath(r) != "" {
				continue // indexed as a per-architecture manifest, never a candidate
			}
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

	// a "*" spans the ":" between image and tag: "/" is the matcher's only separator
	dockerResults := []grpcplugins.SearchResult{
		{Path: "ovhcom/venom/v1.3.0-1234", Name: "manifest.json"},
		{Path: "ovhcom/venom/latest-1234", Name: "list.manifest.json"},
		{Path: "ovhcom/venom/v1.3.0-1234", Name: "manifest.json"}, // another maturity: deduplicated
		{Path: "ovhcom/venom/v1.2.0-999", Name: "manifest.json"},
		{Path: "ovhcom/utask/v1.0.0-1234", Name: "manifest.json"},
		{Path: "other/venom/v1.3.0-1234", Name: "manifest.json"},
		{Path: "ovhcom/venom/sha256:1899ab", Name: "manifest.json"},
	}
	require.Equal(t, []string{"ovhcom/utask:v1.0.0-1234", "ovhcom/venom:latest-1234", "ovhcom/venom:v1.3.0-1234"},
		filterCandidates(dockerResults, sdk.V2WorkflowRunResultTypeDocker, "ovhcom/*:*-1234"))
	require.Equal(t, []string{"ovhcom/utask:v1.0.0-1234", "ovhcom/venom:latest-1234", "ovhcom/venom:v1.2.0-999", "ovhcom/venom:v1.3.0-1234"},
		filterCandidates(dockerResults, sdk.V2WorkflowRunResultTypeDocker, "ovhcom/*:*"))
	require.Equal(t, []string{"ovhcom/venom:v1.2.0-999", "ovhcom/venom:v1.3.0-1234"},
		filterCandidates(dockerResults, sdk.V2WorkflowRunResultTypeDocker, "ovhcom/venom:* !ovhcom/venom:latest-*"))

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
