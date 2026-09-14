package grpcplugins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

const (
	// Digest of the config blob, as carried by the mono-arch manifest served below. This is
	// also what dockerPush reads locally from the image it just pushed (image.ID[7:19]).
	testDockerConfigDigest = "sha256:1a2b3c4d5e6f7890abcdef0123456789abcdef0123456789abcdef0123456789"
	testDockerConfigID     = "1a2b3c4d5e6f"

	// Checksum of the manifest file itself, as reported by artifactory.
	testDockerManifestSha256 = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	testDockerManifestID     = "fedcba987654"

	testDockerArchDigest = "sha256:aaaabbbbccccddddeeeeffff00001111222233334444555566667777888899990"
	testDockerArchID     = "aaaabbbbcccc"
)

// The only network call FinalizeRunResultDockerDetailFromFiles makes. Every other piece of
// artifactory data it needs comes from the DockerManifestFiles the caller hands over.
func dockerManifestServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func dockerManifestFiles(downloadURI string, multiArch bool) DockerManifestFiles {
	fi := ArtifactoryFileInfo{
		Repo:        "myrepo-docker-snapshot",
		Path:        "/myimage/1.0/manifest.json",
		DownloadURI: downloadURI,
		URI:         downloadURI,
	}
	fi.Checksums.Md5 = "d41d8cd98f00b204e9800998ecf8427e"
	fi.Checksums.Sha1 = "da39a3ee5e6b4b0d3255bfef95601890afd80709"
	fi.Checksums.Sha256 = testDockerManifestSha256

	archFi := ArtifactoryFileInfo{Path: "/myimage/" + testDockerArchDigest + "/manifest.json"}
	archFi.Checksums.Md5 = "0cc175b9c0f1b6a831c399e269772661"
	archFi.Checksums.Sha1 = "86f7e437faa5a7fce15d1ddcb9eaeaea377667b8"
	archFi.Checksums.Sha256 = testDockerArchDigest

	return DockerManifestFiles{
		FolderPath: "/myimage/1.0",
		FileInfo:   fi,
		MultiArch:  multiArch,
		ArchFileInfo: func(string) (*ArtifactoryFileInfo, error) {
			return &archFi, nil
		},
	}
}

func finalizeDockerDetail(t *testing.T, srv *httptest.Server, imageID string, multiArch bool) *sdk.V2WorkflowRunResult {
	t.Helper()

	img := Img{Repository: "myimage", Tag: "1.0", ImageID: imageID}
	result := sdk.V2WorkflowRunResult{Type: sdk.V2WorkflowRunResultTypeDocker}
	c := &actionplugin.Common{HTTPClient: srv.Client()}

	err := FinalizeRunResultDockerDetailFromFiles(context.TODO(), c, ArtifactoryConfig{URL: srv.URL, Token: "tok"},
		&result, "myrepo-docker.artifactory.local/myimage:1.0", &img,
		"myrepo-docker-snapshot", "myrepo-docker", "snapshot", dockerManifestFiles(srv.URL, multiArch))
	require.NoError(t, err)

	require.Equal(t, sdk.V2WorkflowRunResultStatusCompleted, result.Status)
	require.Equal(t, "/myimage/1.0", result.ArtifactManagerMetadata.Get("dir"))
	return &result
}

func dockerDetail(t *testing.T, result *sdk.V2WorkflowRunResult) *sdk.V2WorkflowRunResultDockerDetail {
	t.Helper()
	detail, ok := result.Detail.Data.(*sdk.V2WorkflowRunResultDockerDetail)
	require.True(t, ok, "detail must be a docker detail, got %T", result.Detail.Data)
	require.Equal(t, "myrepo-docker.artifactory.local/myimage:1.0", detail.Name)
	return detail
}

// dockerPush hands over the ImageID it read from the local docker daemon, but the mono-arch
// branch overwrites it with the config blob digest of the manifest stored in artifactory, and
// that is what reaches the "id" metadata. The pre-glob code set "id" before the overwrite and
// kept the local value. Same digest for an image built and pushed by docker, and nothing in
// engine, cli or ui reads that metadata: it only reads repository, localRepository, maturity,
// name, path, dir, downloadURI, md5 and cdn_api_ref_hash.
func TestFinalizeRunResultDockerDetailFromFiles_MonoArchWithLocalImageID(t *testing.T) {
	srv := dockerManifestServer(t, `{"config":{"digest":"`+testDockerConfigDigest+`"}}`)

	result := finalizeDockerDetail(t, srv, "localimageid", false)

	require.Equal(t, testDockerConfigID, result.ArtifactManagerMetadata.Get("id"))
	detail := dockerDetail(t, result)
	require.Len(t, detail.Manifests, 1)
	require.Equal(t, testDockerConfigID, detail.Manifests[0].ID)
	require.Equal(t, "myimage/1.0/manifest.json", detail.Manifests[0].Path)
	require.Equal(t, testDockerManifestSha256, detail.Manifests[0].SHA256)
}

// addRunResult has no local image to read an ImageID from. The mono-arch branch resolves one
// from the manifest it downloads.
func TestFinalizeRunResultDockerDetailFromFiles_MonoArchWithoutLocalImageID(t *testing.T) {
	srv := dockerManifestServer(t, `{"config":{"digest":"`+testDockerConfigDigest+`"}}`)

	result := finalizeDockerDetail(t, srv, "", false)

	require.Equal(t, testDockerConfigID, result.ArtifactManagerMetadata.Get("id"))
	require.Len(t, dockerDetail(t, result).Manifests, 1)
}

// A manifest list has no single config blob to fall back on: the ImageID the caller hands
// over is the only source for "id", and it must reach the metadata untouched.
func TestFinalizeRunResultDockerDetailFromFiles_MultiArchWithLocalImageID(t *testing.T) {
	srv := dockerManifestServer(t, `{"manifests":[{"digest":"`+testDockerArchDigest+`","platform":{"architecture":"amd64","os":"linux"}}]}`)

	result := finalizeDockerDetail(t, srv, "localimageid", true)

	require.Equal(t, "localimageid", result.ArtifactManagerMetadata.Get("id"))
	detail := dockerDetail(t, result)
	require.Len(t, detail.Manifests, 1)
	require.Equal(t, testDockerArchID, detail.Manifests[0].ID)
	require.Equal(t, "linux", detail.Manifests[0].OS)
	require.Equal(t, "amd64", detail.Manifests[0].Architecture)
	require.Equal(t, "/myimage/"+testDockerArchDigest+"/manifest.json", detail.Manifests[0].Path)
}

// Neither a local ImageID nor a config digest here. Without the fallback on the manifest
// checksum the run result would carry an empty id, which is what master stored.
func TestFinalizeRunResultDockerDetailFromFiles_MultiArchWithoutLocalImageID(t *testing.T) {
	srv := dockerManifestServer(t, `{"manifests":[{"digest":"`+testDockerArchDigest+`","platform":{"architecture":"amd64","os":"linux"}}]}`)

	result := finalizeDockerDetail(t, srv, "", true)

	require.Equal(t, testDockerManifestID, result.ArtifactManagerMetadata.Get("id"))
	require.Len(t, dockerDetail(t, result).Manifests, 1)
}
