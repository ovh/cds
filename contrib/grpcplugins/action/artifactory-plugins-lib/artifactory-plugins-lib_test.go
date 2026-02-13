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

func Test_checkRunResultIntegrity_MD5Match(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newRunResult(sdk.V2WorkflowRunResultTypeGeneric, "my-integration", map[string]string{
		"localRepository": "libs-snapshot-local",
		"path":            "com/example/test-1.0.jar",
		"md5":             "abc123def456",
	})

	mockClient.EXPECT().GetFileInfo("libs-snapshot-local", "com/example/test-1.0.jar").Return(sdk.FileInfo{
		Checksums: &sdk.FileInfoChecksum{
			Md5: "abc123def456",
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
	})

	mockClient.EXPECT().GetFileInfo("libs-snapshot-local", "com/example/test-1.0.jar").Return(sdk.FileInfo{
		Checksums: &sdk.FileInfoChecksum{
			Md5: "000000000000",
		},
	}, nil)

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "MD5 mismatch")
}

func Test_checkRunResultIntegrity_NoMD5InMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newRunResult(sdk.V2WorkflowRunResultTypeGeneric, "my-integration", map[string]string{
		"localRepository": "libs-snapshot-local",
		"path":            "com/example/test-1.0.jar",
		// no "md5" key
	})

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no MD5 checksum found")
}

func Test_checkRunResultIntegrity_NotOnArtifactory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	// No integration name => not uploaded on Artifactory
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
	require.Contains(t, err.Error(), "unable to get file info from Artifactory")
}

func Test_checkRunResultIntegrity_MD5CaseInsensitive(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_artifact_manager.NewMockArtifactManager(ctrl)

	r := newRunResult(sdk.V2WorkflowRunResultTypeGeneric, "my-integration", map[string]string{
		"localRepository": "libs-snapshot-local",
		"path":            "com/example/test-1.0.jar",
		"md5":             "ABC123DEF456",
	})

	mockClient.EXPECT().GetFileInfo("libs-snapshot-local", "com/example/test-1.0.jar").Return(sdk.FileInfo{
		Checksums: &sdk.FileInfoChecksum{
			Md5: "abc123def456",
		},
	}, nil)

	c := new(actionplugin.Common)
	err := checkRunResultIntegrity(context.Background(), c, mockClient, r)
	require.NoError(t, err)
}
