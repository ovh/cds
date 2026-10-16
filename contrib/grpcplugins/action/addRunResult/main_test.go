package main

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"
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
		  "artifactory_url": "my-project-default-static/test-static-files/",
		  "public_url": "https://static.example.com/my-project/default/test-static-files"
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

// A literal path wrongly read as a pattern matches nothing, and with if-no-files-found
// defaulting to warn the job stays green with no run result, where it used to fail on the
// missing artifact. Below are the resolved paths the .cds repositories and the venom tests
// really pass.
func TestContainsGlobOnProductionPaths(t *testing.T) {
	for _, path := range []string{
		"pool/package-linux-amd64-1757548800.deb",
		"terraform-linux-amd64-1757548800.tar.gz",
		"mychart/0.1.0-1757548800",
		"busybox/1757548800",
		"sbt.pomo",
		"cds/model/debian7-container:1",
		"internal/my-module/myorg/0.1.0-11.sha.b43d3753.zip",
		"TEST/server.tests-results.xml",
		"pysgu/0.1/pysgu-0.1.tar.gz",
		"/my-python-lib/1.9.2/my_python_lib-1.9.2-py3-none-any.whl",
		"myorg/myprovider/0.4.0/terraform-provider-myprovider_0.4.0_linux_amd64.zip",
		"/my-project/my-application/FRAMEWORK/default/master/my-application-snapshot",
	} {
		require.False(t, containsGlob(path), "literal path %q must not be read as a pattern", path)
	}
}

// staticFiles takes a destination folder, and perform rejects a pattern for that type before
// it even reaches the type switch. The destination is a free-form action input: a bracket in
// one turns a plain upload into a hard failure.
func TestContainsGlobOnStaticFilesDestination(t *testing.T) {
	require.False(t, containsGlob("my-project/static/docs/"))
	require.True(t, containsGlob("my-project/static/docs[fr]/"))
}

func TestGlobSupportedType(t *testing.T) {
	require.False(t, globSupportedType(sdk.V2WorkflowRunResultTypeStaticFiles))
	require.True(t, globSupportedType(sdk.V2WorkflowRunResultTypeDocker))
	require.True(t, globSupportedType(sdk.V2WorkflowRunResultTypeDebian))
	require.True(t, globSupportedType(sdk.V2WorkflowRunResultTypeGeneric))
	require.True(t, globSupportedType(sdk.V2WorkflowRunResultTypeOCI))
	require.True(t, globSupportedType(sdk.V2WorkflowRunResultTypeConan))
}

// TestAqlWildcard locks the glob to AQL $match conversion. The AQL only has to contain the
// glob: a wildcard narrower than the glob silently drops artifacts, the matcher never sees
// them.
func TestAqlWildcard(t *testing.T) {
	require.Equal(t, "pool/*.deb", aqlWildcard("pool/*.deb"))
	require.Equal(t, "myorg/myimage-*/1234-*", aqlWildcard("myorg/myimage-*/1234-*"))
	require.Equal(t, "foo?.txt", aqlWildcard("foo?.txt"))
	// "**" is the AQL "*", which spans "/"
	require.Equal(t, "*", aqlWildcard("**"))
	require.Equal(t, "mirror*", aqlWildcard("mirror/**"))
	require.Equal(t, "*.zip", aqlWildcard("**/*.zip"))
	// the "/" around a "**" go with it: "a/**/b" matches "a/b", the AQL "a/*/b" would not
	require.Equal(t, "a*b", aqlWildcard("a/**/b"))
	// a class is one character
	require.Equal(t, "pool/?*.deb", aqlWildcard("pool/[ab]*.deb"))
	require.Equal(t, "fo??.txt", aqlWildcard("fo[a-z]?.txt"))
	require.Equal(t, "path*?rtifac?/*", aqlWildcard("path/**/[abc]rtifac?/*"))
	// an unterminated class is a literal for the matcher too
	require.Equal(t, "pool/[ab*.deb", aqlWildcard("pool/[ab*.deb"))
}

// TestPatternCriteria tells the incident it prevents: a -docker repository holding 61717
// manifests under one namespace (tag and digest folders of every image), and a pattern
// selecting a handful of images of one run, myorg/myimage-*:<run>-*.
// With only the static folder prefix in the AQL (myorg), the enumeration blew
// past aqlSearchLimit before the matcher saw a single candidate, and the step failed with
// "use a more specific pattern". The image and tag wildcards must reach the AQL.
func TestPatternCriteria(t *testing.T) {
	require.Equal(t, `{"path":{"$match":"myorg/myimage-*/1234-*"}}`,
		patternCriteria(dockerPatternToPath("myorg/myimage-*:1234-*"), sdk.V2WorkflowRunResultTypeDocker))

	// file-based types: folder path and file name, split on the last "/"
	require.Equal(t, `{"path":{"$match":"pool"},"name":{"$match":"glob-1234-*.deb"}}`,
		patternCriteria(trimPatternsLeadingSlash("/pool/glob-1234-*.deb"), sdk.V2WorkflowRunResultTypeDebian))
	require.Equal(t, `{"path":{"$match":"myorg/myprovider/0.24.0"},"name":{"$match":"*.zip"}}`,
		patternCriteria("myorg/myprovider/0.24.0/*.zip", sdk.V2WorkflowRunResultTypeTerraformProvider))
	require.Equal(t, `{"path":{"$match":"pool"},"name":{"$match":"?*.deb"}}`, patternCriteria("pool/[ab]*.deb", sdk.V2WorkflowRunResultTypeDebian))
	// a root pattern narrows by name only: artifactory reports the root path as ".", no
	// criterion is safer than a guess
	require.Equal(t, `{"name":{"$match":"terraform-*.tar.gz"}}`, patternCriteria("terraform-*.tar.gz", sdk.V2WorkflowRunResultTypeTerraformProvider))
	// a last segment holding "**" spans folders: only the folders before it narrow the path
	require.Equal(t, `{"path":{"$match":"pool*"}}`, patternCriteria("pool/**", sdk.V2WorkflowRunResultTypeDebian))
	require.Equal(t, `{"path":{"$match":"a*"}}`, patternCriteria("a/b**", sdk.V2WorkflowRunResultTypeDebian))
	require.Equal(t, `{"path":{"$match":"*"},"name":{"$match":"*.zip"}}`, patternCriteria("**/*.zip", sdk.V2WorkflowRunResultTypeGeneric))
	require.Equal(t, `{"path":{"$match":"a*b"},"name":{"$match":"*.zip"}}`, patternCriteria("a/**/b/*.zip", sdk.V2WorkflowRunResultTypeGeneric))
	// anywhere in the repository: nothing to narrow
	require.Equal(t, "", patternCriteria("**", sdk.V2WorkflowRunResultTypeDebian))

	// several positive patterns are an $or, exclusions only remove matches
	require.Equal(t, `{"$or":[{"path":{"$match":"pool"},"name":{"$match":"a*.deb"}},{"path":{"$match":"pool"},"name":{"$match":"b*.deb"}}]}`,
		patternCriteria("pool/a*.deb pool/b*.deb !pool/*-dbg*", sdk.V2WorkflowRunResultTypeDebian))
	require.Equal(t, "", patternCriteria("!pool/*", sdk.V2WorkflowRunResultTypeDebian))

	// oci and docker match the package folder, conan its export folder
	require.Equal(t, `{"path":{"$match":"services/*/*"}}`, patternCriteria("services/*/*", sdk.V2WorkflowRunResultTypeOCI))
	require.Equal(t, `{"path":{"$match":"mirror*"}}`, patternCriteria("mirror/** !**/sha256:*", sdk.V2WorkflowRunResultTypeOCI))
	require.Equal(t, `{"path":{"$match":"_/abseil/*/_/*/export"}}`, patternCriteria("_/abseil/*/_/*", sdk.V2WorkflowRunResultTypeConan))
	// the matcher accepts a trailing "/", an AQL path never ends with one
	require.Equal(t, `{"path":{"$match":"services/*"}}`, patternCriteria("services/*/", sdk.V2WorkflowRunResultTypeOCI))
}

// TestAqlWildcardIsSuperset locks the invariant the narrowing rests on: whatever the glob
// matcher accepts, the AQL wildcard accepts too. An AQL "*" or "?" spans "/" like the SQL
// LIKE it is translated to. The matcher lets a pattern literal skip one "/" of the content
// (pool/*x.deb accepts pool/sub/x.deb), a quirk the AQL does not serve: the layouts below
// are the realistic ones of TestGlobSelection, where it does not fire.
func TestAqlWildcardIsSuperset(t *testing.T) {
	patterns := []string{"pool/*.deb", "pool/**", "**/*.zip", "a/**/b/*.zip", "path/**/[abc]rtifac?/*", "services/*/*", "mirror/**", "myorg/*/*-1234", "_/*/*/_/*", "*", "**", "foo?.txt", "fo[a-z]?.txt"}
	paths := []string{"pool/a.deb", "pool/sub/b.deb", "a.deb", "a/b/x.zip", "a/x/y/b/x.zip", "path/to/artifact/foo1.txt", "services/core/1.0", "mirror/x/sha256:abc", "myorg/myimage/v1-1234", "_/abseil/1.0/_/rev", "foo1.txt", "fooo.txt"}
	var checked int
	for _, pattern := range patterns {
		aql := regexp.MustCompile("^" + strings.NewReplacer(`\*`, ".*", `\?`, ".").Replace(regexp.QuoteMeta(aqlWildcard(pattern))) + "$")
		for _, path := range paths {
			m, err := glob.New(pattern).MatchString(path)
			require.NoError(t, err)
			if m == nil {
				continue
			}
			checked++
			require.True(t, aql.MatchString(path), "glob %q matches %q, aql %q must too", pattern, path, aqlWildcard(pattern))
		}
	}
	require.Greater(t, checked, 10, "the lists must overlap enough to mean something")
}

// TestTrimPatternsLeadingSlash: a user writes the glob like the single path, often with a
// leading "/", while the matched candidates never have one. Without the trim nothing matches
// and addRunResult only warns that no artifact was found.
func TestTrimPatternsLeadingSlash(t *testing.T) {
	require.Equal(t, "myorg/myprovider/0.24.0/*.zip", trimPatternsLeadingSlash("/myorg/myprovider/0.24.0/*.zip"))
	require.Equal(t, "pool/*.deb !pool/*-dbg*", trimPatternsLeadingSlash("/pool/*.deb !/pool/*-dbg*"))
	require.Equal(t, "pool/*.deb", trimPatternsLeadingSlash("pool/*.deb"))
	require.Equal(t, "**/*.zip", trimPatternsLeadingSlash("/**/*.zip"))
}

// TestRepositoryForType locks the mapping between a run result type and the artifactory
// repository holding its artifacts. The suffix is the repository type autoconf creates for a
// project, not the result type: getting it wrong makes addRunResult query a repository the
// project does not have, and the artifact is reported as missing although it was uploaded.
func TestRepositoryForType(t *testing.T) {
	// freebsd is created as <prefix>-freebsd-{snapshot,release}, behind the
	// <prefix>-freebsd virtual repository
	require.Equal(t, "proj-freebsd", repositoryForType("proj", sdk.V2WorkflowRunResultTypeFreeBSD))

	// what CDS stores in its own repository shares it
	require.Equal(t, "proj-cds", repositoryForType("proj", sdk.V2WorkflowRunResultTypeGeneric))
	require.Equal(t, "proj-cds", repositoryForType("proj", sdk.V2WorkflowRunResultTypeTest))
	require.Equal(t, "proj-cds", repositoryForType("proj", sdk.V2WorkflowRunResultTypeCoverage))

	// the suffix is the repository type, which is not always the result type: python is pypi
	require.Equal(t, "proj-pypi", repositoryForType("proj", sdk.V2WorkflowRunResultTypePython))
	require.Equal(t, "proj-debian", repositoryForType("proj", sdk.V2WorkflowRunResultTypeDebian))
	require.Equal(t, "proj-oci", repositoryForType("proj", sdk.V2WorkflowRunResultTypeOCI))

	// a type with no repository of its own falls back to the bare prefix: this is why an
	// absent or misspelled "type" input queries <prefix> and reports a puzzling 404
	require.Equal(t, "proj", repositoryForType("proj", sdk.V2WorkflowRunResultTypeVariable))
	require.Equal(t, "proj", repositoryForType("proj", ""))
}

func TestRepoCriteria(t *testing.T) {
	require.Equal(t, `{"repo":{"$match":"proj-debian-*"}}`, repoCriteria("proj-debian"))
	require.Equal(t, `{"$or":[{"repo":{"$match":"proj-cds-*"}},{"repo":{"$match":"proj-generic-*"}}]}`, repoCriteria("proj-cds"))
}

func TestDockerRepoCriteria(t *testing.T) {
	require.Equal(t, `{"repo":{"$eq":"proj-docker-snapshot"}}`, dockerRepoCriteria("proj-docker", "snapshot"))
}

// TestDockerPatternToPath locks the rewrite feeding patternCriteria: the ":" of an image
// reference is a "/" in the artifactory layout, and the image folder belongs to the path.
func TestDockerPatternToPath(t *testing.T) {
	require.Equal(t, "busybox/*", dockerPatternToPath("busybox:*"))
	require.Equal(t, "ovhcom/venom/*", dockerPatternToPath("ovhcom/venom:*"))
	require.Equal(t, "ovhcom/*/*-1234", dockerPatternToPath("ovhcom/*:*-1234"))
	// no tag part to rewrite
	require.Equal(t, "ovhcom/**", dockerPatternToPath("ovhcom/**"))
	// exclusions keep their "!" and are rewritten too
	require.Equal(t, "ovhcom/venom/* !ovhcom/venom/latest-*", dockerPatternToPath("ovhcom/venom:* !ovhcom/venom:latest-*"))

	// without the rewrite, "busybox:*" narrows the path to a folder that does not exist
	require.Equal(t, `{"path":{"$match":"busybox/*"}}`, patternCriteria(dockerPatternToPath("busybox:*"), sdk.V2WorkflowRunResultTypeDocker))
	require.Equal(t, `{"path":{"$match":"busybox:*"}}`, patternCriteria("busybox:*", sdk.V2WorkflowRunResultTypeDocker))
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

// TestGlobSearchAQL locks the absence of a limit() in the enumeration query. Artifactory
// applies limit() to the database rows returned by the property include, one per property,
// not to the items: a limit(10000) on a folder holding 4984 debian packages returned 421 of
// them, reported "total": 421 and set no range notification, so the truncation was
// undetectable and the artifacts of the current run fell outside the window.
func TestGlobSearchAQL(t *testing.T) {
	aql := globSearchAQL([]string{`{"repo":{"$match":"myrepo-*"}}`, `{"type":"file"}`})
	require.Equal(t, `items.find({"$and":[{"repo":{"$match":"myrepo-*"}},{"type":"file"}]}).include("repo","path","name","actual_md5","actual_sha1","sha256","size","created","created_by","property")`, aql)
	require.NotContains(t, aql, ".limit(")
	require.NotContains(t, aql, ".offset(")
}

// TestGlobSelection covers the candidate filtering as done by enumerateGlobMatches, on
// synthetic search results mimicking the layouts audited on real repositories.
func TestGlobSelection(t *testing.T) {
	filterCandidates := func(results []grpcplugins.SearchResult, resultType sdk.V2WorkflowRunResultType, pattern string) []string {
		g := glob.New(trimPatternsLeadingSlash(pattern)) // as enumerateGlobMatches does
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

	// the layout of the oci integration test, taken from the repository it runs against:
	// three packages of the same run plus a digest folder and packages of earlier runs. The
	// <ts> suffix is what keeps the pattern off the previous runs, and mychart is excluded
	// because it is registered by its own single-path step.
	ociRunResults := []grpcplugins.SearchResult{
		{Path: "busybox/1789057565", Name: "manifest.json"},
		{Path: "myapp/0.0.1-1789057565", Name: "manifest.json"},
		{Path: "mychart/0.1.0-1789057565", Name: "manifest.json"},
		{Path: "mychart/sha256:638980735131438bbc7ab849872e60f404009afad7cc2ee22ee8d73302e93926", Name: "manifest.json"},
		{Path: "busybox/1789050000", Name: "manifest.json"},
		{Path: "myapp/0.0.1-1789050000", Name: "manifest.json"},
	}
	require.Equal(t, []string{"busybox/1789057565", "myapp/0.0.1-1789057565"},
		filterCandidates(ociRunResults, sdk.V2WorkflowRunResultTypeOCI, "*/*1789057565 !mychart/*"))

	// the debian counterpart: a pattern whose wildcard sits between a literal prefix and a
	// literal suffix, against the packages of the current run only.
	debianRunResults := []grpcplugins.SearchResult{
		{Path: "pool", Name: "glob-1789057606-a.deb"},
		{Path: "pool", Name: "glob-1789057606-b.deb"},
		{Path: "pool", Name: "glob-1789050000-a.deb"},
		{Path: "pool", Name: "package-linux-amd64-1789057606.deb"},
	}
	require.Equal(t, []string{"pool/glob-1789057606-a.deb", "pool/glob-1789057606-b.deb"},
		filterCandidates(debianRunResults, sdk.V2WorkflowRunResultTypeDebian, "pool/glob-1789057606-*.deb"))

	// a leading "/" is accepted by the single path flow, so a glob written the same way must
	// match too: the candidates carry no leading "/", the pattern has to lose it
	terraformResults := []grpcplugins.SearchResult{
		{Path: "myorg/myprovider/0.24.0", Name: "terraform-provider-myprovider_0.24.0_linux_amd64.zip"},
		{Path: "myorg/myprovider/0.24.0", Name: "terraform-provider-myprovider_0.24.0_darwin_arm64.zip"},
		{Path: "myorg/myprovider/0.23.0", Name: "terraform-provider-myprovider_0.23.0_linux_amd64.zip"},
	}
	require.Equal(t, []string{"myorg/myprovider/0.24.0/terraform-provider-myprovider_0.24.0_darwin_arm64.zip", "myorg/myprovider/0.24.0/terraform-provider-myprovider_0.24.0_linux_amd64.zip"},
		filterCandidates(terraformResults, sdk.V2WorkflowRunResultTypeTerraformProvider, "/myorg/myprovider/0.24.0/terraform-provider-myprovider_0.24.0_*.zip"))
	require.Equal(t, []string{"myorg/myprovider/0.24.0/terraform-provider-myprovider_0.24.0_linux_amd64.zip"},
		filterCandidates(terraformResults, sdk.V2WorkflowRunResultTypeTerraformProvider, "/myorg/myprovider/0.24.0/*.zip !/myorg/**/*darwin*"))

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
