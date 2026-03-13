package artifactorypluginslib

import (
	"context"
	"fmt"
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/artifact_manager/mock_artifact_manager"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newRunResult(t sdk.V2WorkflowRunResultType, integrationName string, metadata map[string]string) sdk.V2WorkflowRunResult {
	r := sdk.V2WorkflowRunResult{
		Type: t,
		Detail: sdk.V2WorkflowRunResultDetail{
			Data: &sdk.V2WorkflowRunResultGenericDetail{Name: "test-artifact"},
			Type: "V2WorkflowRunResultGenericDetail",
		},
	}
	if integrationName != "" {
		r.ArtifactManagerIntegrationName = &integrationName
	}
	if metadata != nil {
		m := sdk.V2WorkflowRunResultArtifactManagerMetadata(metadata)
		r.ArtifactManagerMetadata = &m
	}
	return r
}

func newDockerRunResult(integrationName string, metadata map[string]string, manifests []sdk.V2WorkflowRunResultDockerDetailImage) sdk.V2WorkflowRunResult {
	r := sdk.V2WorkflowRunResult{
		Type: sdk.V2WorkflowRunResultTypeDocker,
		Detail: sdk.V2WorkflowRunResultDetail{
			Data: &sdk.V2WorkflowRunResultDockerDetail{
				Name:      "my-image:latest",
				Manifests: manifests,
			},
			Type: "V2WorkflowRunResultDockerDetail",
		},
	}
	if integrationName != "" {
		r.ArtifactManagerIntegrationName = &integrationName
	}
	if metadata != nil {
		m := sdk.V2WorkflowRunResultArtifactManagerMetadata(metadata)
		r.ArtifactManagerMetadata = &m
	}
	return r
}

func newConanRunResult(integrationName string, metadata map[string]string, files []sdk.V2WorkflowRunResultConanDetailFile) sdk.V2WorkflowRunResult {
	r := sdk.V2WorkflowRunResult{
		Type: sdk.V2WorkflowRunResultTypeConan,
		Detail: sdk.V2WorkflowRunResultDetail{
			Data: &sdk.V2WorkflowRunResultConanDetail{
				Name:    "mylib",
				Version: "1.0.0",
				Files:   files,
			},
			Type: "V2WorkflowRunResultConanDetail",
		},
	}
	if integrationName != "" {
		r.ArtifactManagerIntegrationName = &integrationName
	}
	if metadata != nil {
		m := sdk.V2WorkflowRunResultArtifactManagerMetadata(metadata)
		r.ArtifactManagerMetadata = &m
	}
	return r
}

// --- Single file tests (generic) ---

func Test_checkRunResultIntegrity_AllChecksumsMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newRunResult(sdk.V2WorkflowRunResultTypeGeneric, "my-integration", map[string]string{
		"localRepository": "libs-snapshot-local",
		"path":            "com/example/test-1.0.jar",
		"md5":             "abc123def456",
		"sha1":            "sha1aaa",
		"sha256":          "sha256bbb",
	})

	mockClient.EXPECT().GetFileInfo("libs-snapshot-local", "com/example/test-1.0.jar").Return(sdk.FileInfo{
		Checksums: &sdk.FileInfoChecksum{
			Md5:    "abc123def456",
			Sha1:   "sha1aaa",
			Sha256: "sha256bbb",
		},
	}, nil)

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.NoError(t, err)
}

func Test_checkRunResultIntegrity_MD5Mismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newRunResult(sdk.V2WorkflowRunResultTypeGeneric, "my-integration", map[string]string{
		"localRepository": "libs-snapshot-local",
		"path":            "com/example/test-1.0.jar",
		"md5":             "abc123def456",
		"sha1":            "sha1aaa",
		"sha256":          "sha256bbb",
	})

	mockClient.EXPECT().GetFileInfo("libs-snapshot-local", "com/example/test-1.0.jar").Return(sdk.FileInfo{
		Checksums: &sdk.FileInfoChecksum{
			Md5:    "000000000000",
			Sha1:   "sha1aaa",
			Sha256: "sha256bbb",
		},
	}, nil)

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "MD5 mismatch")
}

func Test_checkRunResultIntegrity_SHA1Mismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newRunResult(sdk.V2WorkflowRunResultTypeGeneric, "my-integration", map[string]string{
		"localRepository": "libs-snapshot-local",
		"path":            "com/example/test-1.0.jar",
		"md5":             "abc123def456",
		"sha1":            "sha1aaa",
		"sha256":          "sha256bbb",
	})

	mockClient.EXPECT().GetFileInfo("libs-snapshot-local", "com/example/test-1.0.jar").Return(sdk.FileInfo{
		Checksums: &sdk.FileInfoChecksum{
			Md5:    "abc123def456",
			Sha1:   "wrongsha1",
			Sha256: "sha256bbb",
		},
	}, nil)

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "SHA1 mismatch")
}

func Test_checkRunResultIntegrity_SHA256Mismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newRunResult(sdk.V2WorkflowRunResultTypeGeneric, "my-integration", map[string]string{
		"localRepository": "libs-snapshot-local",
		"path":            "com/example/test-1.0.jar",
		"md5":             "abc123def456",
		"sha1":            "sha1aaa",
		"sha256":          "sha256bbb",
	})

	mockClient.EXPECT().GetFileInfo("libs-snapshot-local", "com/example/test-1.0.jar").Return(sdk.FileInfo{
		Checksums: &sdk.FileInfoChecksum{
			Md5:    "abc123def456",
			Sha1:   "sha1aaa",
			Sha256: "wrongsha256",
		},
	}, nil)

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "SHA256 mismatch")
}

func Test_checkRunResultIntegrity_NoChecksumInMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newRunResult(sdk.V2WorkflowRunResultTypeGeneric, "my-integration", map[string]string{
		"localRepository": "libs-snapshot-local",
		"path":            "com/example/test-1.0.jar",
	})

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "has no checksum")
}

func Test_checkRunResultIntegrity_NotOnArtifactory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newRunResult(sdk.V2WorkflowRunResultTypeGeneric, "", map[string]string{
		"md5": "abc123",
	})

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no Artifactory integration found")
}

func Test_checkRunResultIntegrity_NilChecksums(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newRunResult(sdk.V2WorkflowRunResultTypeGeneric, "my-integration", map[string]string{
		"localRepository": "libs-snapshot-local",
		"path":            "com/example/test-1.0.jar",
		"md5":             "abc123def456",
	})

	mockClient.EXPECT().GetFileInfo("libs-snapshot-local", "com/example/test-1.0.jar").Return(sdk.FileInfo{
		Checksums: nil,
	}, nil)

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no checksums returned by Artifactory")
}

func Test_checkRunResultIntegrity_GetFileInfoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newRunResult(sdk.V2WorkflowRunResultTypeGeneric, "my-integration", map[string]string{
		"localRepository": "libs-snapshot-local",
		"path":            "com/example/test-1.0.jar",
		"md5":             "abc123def456",
	})

	mockClient.EXPECT().GetFileInfo("libs-snapshot-local", "com/example/test-1.0.jar").Return(sdk.FileInfo{}, fmt.Errorf("connection refused"))

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to get file info for")
}

func Test_checkRunResultIntegrity_CaseInsensitive(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newRunResult(sdk.V2WorkflowRunResultTypeGeneric, "my-integration", map[string]string{
		"localRepository": "libs-snapshot-local",
		"path":            "com/example/test-1.0.jar",
		"md5":             "ABC123DEF456",
		"sha1":            "SHA1AAA",
		"sha256":          "SHA256BBB",
	})

	mockClient.EXPECT().GetFileInfo("libs-snapshot-local", "com/example/test-1.0.jar").Return(sdk.FileInfo{
		Checksums: &sdk.FileInfoChecksum{
			Md5:    "abc123def456",
			Sha1:   "sha1aaa",
			Sha256: "sha256bbb",
		},
	}, nil)

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.NoError(t, err)
}

func Test_checkRunResultIntegrity_PartialChecksums(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	// Only md5 in metadata, no sha1/sha256 => should still pass if md5 matches
	r := newRunResult(sdk.V2WorkflowRunResultTypeGeneric, "my-integration", map[string]string{
		"localRepository": "libs-snapshot-local",
		"path":            "com/example/test-1.0.jar",
		"md5":             "abc123def456",
	})

	mockClient.EXPECT().GetFileInfo("libs-snapshot-local", "com/example/test-1.0.jar").Return(sdk.FileInfo{
		Checksums: &sdk.FileInfoChecksum{
			Md5:    "abc123def456",
			Sha1:   "sha1aaa",
			Sha256: "sha256bbb",
		},
	}, nil)

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.NoError(t, err)
}

// --- Docker tests ---

func Test_checkRunResultIntegrity_DockerAllMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newDockerRunResult("my-integration", map[string]string{
		"localRepository": "docker-snapshot-local",
	}, []sdk.V2WorkflowRunResultDockerDetailImage{
		{ID: "aaa111", Path: "myimage/sha256:abc/manifest.json", MD5: "md5aaa", SHA1: "sha1aaa", SHA256: "sha256aaa"},
		{ID: "bbb222", Path: "myimage/sha256:def/manifest.json", MD5: "md5bbb", SHA1: "sha1bbb", SHA256: "sha256bbb"},
	})

	mockClient.EXPECT().GetFileInfo("docker-snapshot-local", "myimage/sha256:abc/manifest.json").Return(sdk.FileInfo{
		Checksums: &sdk.FileInfoChecksum{Md5: "md5aaa", Sha1: "sha1aaa", Sha256: "sha256aaa"},
	}, nil)
	mockClient.EXPECT().GetFileInfo("docker-snapshot-local", "myimage/sha256:def/manifest.json").Return(sdk.FileInfo{
		Checksums: &sdk.FileInfoChecksum{Md5: "md5bbb", Sha1: "sha1bbb", Sha256: "sha256bbb"},
	}, nil)

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.NoError(t, err)
}

func Test_checkRunResultIntegrity_DockerManifestMD5Mismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newDockerRunResult("my-integration", map[string]string{
		"localRepository": "docker-snapshot-local",
	}, []sdk.V2WorkflowRunResultDockerDetailImage{
		{ID: "aaa111", Path: "myimage/sha256:abc/manifest.json", MD5: "md5aaa", SHA1: "sha1aaa", SHA256: "sha256aaa"},
	})

	mockClient.EXPECT().GetFileInfo("docker-snapshot-local", "myimage/sha256:abc/manifest.json").Return(sdk.FileInfo{
		Checksums: &sdk.FileInfoChecksum{Md5: "wrongmd5", Sha1: "sha1aaa", Sha256: "sha256aaa"},
	}, nil)

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "MD5 mismatch")
	require.Contains(t, err.Error(), "aaa111")
}

func Test_checkRunResultIntegrity_DockerManifestSHA1Mismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newDockerRunResult("my-integration", map[string]string{
		"localRepository": "docker-snapshot-local",
	}, []sdk.V2WorkflowRunResultDockerDetailImage{
		{ID: "aaa111", Path: "myimage/sha256:abc/manifest.json", MD5: "md5aaa", SHA1: "sha1aaa", SHA256: "sha256aaa"},
	})

	mockClient.EXPECT().GetFileInfo("docker-snapshot-local", "myimage/sha256:abc/manifest.json").Return(sdk.FileInfo{
		Checksums: &sdk.FileInfoChecksum{Md5: "md5aaa", Sha1: "wrongsha1", Sha256: "sha256aaa"},
	}, nil)

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "SHA1 mismatch")
	require.Contains(t, err.Error(), "aaa111")
}

func Test_checkRunResultIntegrity_DockerManifestNoChecksum(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newDockerRunResult("my-integration", map[string]string{
		"localRepository": "docker-snapshot-local",
	}, []sdk.V2WorkflowRunResultDockerDetailImage{
		{ID: "aaa111", Path: "myimage/sha256:abc/manifest.json"},
	})

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "has no checksum")
}

func Test_checkRunResultIntegrity_DockerManifestNoPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newDockerRunResult("my-integration", map[string]string{
		"localRepository": "docker-snapshot-local",
	}, []sdk.V2WorkflowRunResultDockerDetailImage{
		{ID: "aaa111", MD5: "md5aaa"},
	})

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "has no path")
}

func Test_checkRunResultIntegrity_DockerGetFileInfoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newDockerRunResult("my-integration", map[string]string{
		"localRepository": "docker-snapshot-local",
	}, []sdk.V2WorkflowRunResultDockerDetailImage{
		{ID: "aaa111", Path: "myimage/sha256:abc/manifest.json", MD5: "md5aaa", SHA1: "sha1aaa", SHA256: "sha256aaa"},
	})

	mockClient.EXPECT().GetFileInfo("docker-snapshot-local", "myimage/sha256:abc/manifest.json").Return(sdk.FileInfo{}, fmt.Errorf("not found"))

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to get file info for manifest")
}

// --- Conan tests ---

func Test_checkRunResultIntegrity_ConanAllMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newConanRunResult("my-integration", map[string]string{
		"localRepository": "conan-snapshot-local",
	}, []sdk.V2WorkflowRunResultConanDetailFile{
		{FileName: "conanfile.py", Path: "mylib/1.0.0/_/_/export", MD5: "md5aaa", SHA1: "sha1aaa", SHA256: "sha256aaa"},
		{FileName: "conanmanifest.txt", Path: "mylib/1.0.0/_/_/export", MD5: "md5bbb", SHA1: "sha1bbb", SHA256: "sha256bbb"},
	})

	mockClient.EXPECT().GetFileInfo("conan-snapshot-local", "mylib/1.0.0/_/_/export/conanfile.py").Return(sdk.FileInfo{
		Checksums: &sdk.FileInfoChecksum{Md5: "md5aaa", Sha1: "sha1aaa", Sha256: "sha256aaa"},
	}, nil)
	mockClient.EXPECT().GetFileInfo("conan-snapshot-local", "mylib/1.0.0/_/_/export/conanmanifest.txt").Return(sdk.FileInfo{
		Checksums: &sdk.FileInfoChecksum{Md5: "md5bbb", Sha1: "sha1bbb", Sha256: "sha256bbb"},
	}, nil)

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.NoError(t, err)
}

func Test_checkRunResultIntegrity_ConanFileMD5Mismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newConanRunResult("my-integration", map[string]string{
		"localRepository": "conan-snapshot-local",
	}, []sdk.V2WorkflowRunResultConanDetailFile{
		{FileName: "conanfile.py", Path: "mylib/1.0.0/_/_/export", MD5: "md5aaa", SHA1: "sha1aaa", SHA256: "sha256aaa"},
	})

	mockClient.EXPECT().GetFileInfo("conan-snapshot-local", "mylib/1.0.0/_/_/export/conanfile.py").Return(sdk.FileInfo{
		Checksums: &sdk.FileInfoChecksum{Md5: "wrongmd5", Sha1: "sha1aaa", Sha256: "sha256aaa"},
	}, nil)

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "MD5 mismatch")
	require.Contains(t, err.Error(), "conanfile.py")
}

func Test_checkRunResultIntegrity_ConanFileSHA256Mismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newConanRunResult("my-integration", map[string]string{
		"localRepository": "conan-snapshot-local",
	}, []sdk.V2WorkflowRunResultConanDetailFile{
		{FileName: "conanfile.py", Path: "mylib/1.0.0/_/_/export", MD5: "md5aaa", SHA1: "sha1aaa", SHA256: "sha256aaa"},
	})

	mockClient.EXPECT().GetFileInfo("conan-snapshot-local", "mylib/1.0.0/_/_/export/conanfile.py").Return(sdk.FileInfo{
		Checksums: &sdk.FileInfoChecksum{Md5: "md5aaa", Sha1: "sha1aaa", Sha256: "wrongsha256"},
	}, nil)

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "SHA256 mismatch")
	require.Contains(t, err.Error(), "conanfile.py")
}

func Test_checkRunResultIntegrity_ConanFileNoChecksum(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newConanRunResult("my-integration", map[string]string{
		"localRepository": "conan-snapshot-local",
	}, []sdk.V2WorkflowRunResultConanDetailFile{
		{FileName: "conanfile.py", Path: "mylib/1.0.0/_/_/export"},
	})

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "has no checksum")
}

func Test_checkRunResultIntegrity_ConanGetFileInfoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newConanRunResult("my-integration", map[string]string{
		"localRepository": "conan-snapshot-local",
	}, []sdk.V2WorkflowRunResultConanDetailFile{
		{FileName: "conanfile.py", Path: "mylib/1.0.0/_/_/export", MD5: "md5aaa", SHA1: "sha1aaa", SHA256: "sha256aaa"},
	})

	mockClient.EXPECT().GetFileInfo("conan-snapshot-local", "mylib/1.0.0/_/_/export/conanfile.py").Return(sdk.FileInfo{}, fmt.Errorf("not found"))

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to get file info for")
}
