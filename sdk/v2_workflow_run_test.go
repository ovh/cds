package sdk

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetContextConfig(t *testing.T) {

	j := JobIntegrationsContext{
		ModelName: ArtifactoryIntegrationModelName,
		Config: JobIntegrationsContextConfig{
			"repo": map[string]interface{}{
				"build": map[string]interface{}{
					"info": map[string]interface{}{
						"data": "foo",
					},
				},
			},
			"url":        "myurl",
			"token":      "mytoken",
			"token_name": "username",
		},
	}
	require.Equal(t, "myurl", j.Get("url"))
	require.Equal(t, "foo", j.Get("repo.build.info.data"))
	require.Equal(t, "mytoken", j.Get(ArtifactoryConfigToken))
	require.Equal(t, "username", j.Get(ArtifactoryConfigTokenName))
}

func TestV2WorkflowRunResult_GetDetail_interface_conversion(t *testing.T) {
	var a V2WorkflowRunResult = V2WorkflowRunResult{
		Detail: V2WorkflowRunResultDetail{
			Data: V2WorkflowRunResultVariableDetail{
				Name:  "foo",
				Value: "bar",
			},
		},
	}

	_, err := a.GetDetail()
	require.NoError(t, err)

	var b V2WorkflowRunResult = V2WorkflowRunResult{
		Detail: V2WorkflowRunResultDetail{
			Data: &V2WorkflowRunResultVariableDetail{
				Name:  "foo",
				Value: "bar",
			},
		},
	}

	_, err = b.GetDetail()
	require.NoError(t, err)
}

func TestMarshalV2WorkflowRunResultReleaseDetail(t *testing.T) {
	var a = &V2WorkflowRunResult{
		IssuedAt: time.Now(),
		Status:   V2WorkflowRunResultStatusCompleted,
		Type:     V2WorkflowRunResultTypeRelease,
		Detail: V2WorkflowRunResultDetail{
			Data: V2WorkflowRunResultReleaseDetail{
				Name:    "releaseName",
				Version: "releaseVersion",
				SBOM:    []byte("{}"),
			},
		},
		ArtifactManagerMetadata: &V2WorkflowRunResultArtifactManagerMetadata{
			"releaseName":    "releaseName",
			"releaseVersion": "releaseVersion",
		},
	}

	btes, err := json.Marshal(a)
	require.NoError(t, err)

	err = JSONUnmarshal(btes, &a)
	require.NoError(t, err)
}

func TestJobIntegrationsContext_GetEmpty(t *testing.T) {
	j := JobIntegrationsContext{
		Name:      "name",
		Config:    JobIntegrationsContextConfig{},
		ModelName: "modelName",
	}
	got := j.Get("gw.token")

	require.Equal(t, "", got)
}
func TestJobIntegrationsContext_GetEmptyValue(t *testing.T) {
	j := JobIntegrationsContext{
		Name: "name",
		Config: JobIntegrationsContextConfig{
			"gw": map[string]interface{}{
				"token": "",
			},
		},
		ModelName: "modelName",
	}
	got := j.Get("gw.token")

	require.Equal(t, "", got)

}
func TestJobIntegrationsContext_GetValue(t *testing.T) {
	j := JobIntegrationsContext{
		Name: "name",
		Config: JobIntegrationsContextConfig{
			"gw": map[string]interface{}{
				"token": "value_of_token",
			},
		},
		ModelName: "modelName",
	}
	got := j.Get("gw.token")

	require.Equal(t, "value_of_token", got)
}

func TestV2Initiator_IsUnknown(t *testing.T) {
	var nilInitiator *V2Initiator
	require.True(t, nilInitiator.IsUnknown())
	require.True(t, (&V2Initiator{}).IsUnknown())
	require.True(t, (&V2Initiator{VCS: "github"}).IsUnknown())
	require.False(t, (&V2Initiator{UserID: "u1"}).IsUnknown())
	require.False(t, (&V2Initiator{VCS: "github", VCSUsername: "octocat"}).IsUnknown())
}

func TestV2Initiator_EntityOwner(t *testing.T) {
	var nilInitiator *V2Initiator
	require.Nil(t, nilInitiator.EntityOwner())

	src := &V2Initiator{
		UserID:         "u1",
		User:           &V2InitiatorUser{Username: "alice", Ring: UserRingAdmin, Email: "alice@example.com"},
		VCS:            "github",
		VCSUsername:    "alice-gh",
		IsAdminWithMFA: true,
	}
	owner := src.EntityOwner()
	require.False(t, owner.IsAdminWithMFA)
	require.Equal(t, "u1", owner.UserID)
	require.Equal(t, "github", owner.VCS)
	require.Equal(t, "alice-gh", owner.VCSUsername)
	require.Equal(t, "alice", owner.User.Username)
	require.Equal(t, "alice@example.com", owner.User.Email)

	// The copy must not alias the source
	owner.User.Username = "bob"
	require.True(t, src.IsAdminWithMFA)
	require.Equal(t, "alice", src.User.Username)

	vcsOnly := &V2Initiator{VCS: "github", VCSUsername: "octocat", IsAdminWithMFA: true}
	owner = vcsOnly.EntityOwner()
	require.Nil(t, owner.User)
	require.False(t, owner.IsAdminWithMFA)
	require.Equal(t, "github", owner.VCS)
	require.Equal(t, "octocat", owner.VCSUsername)
}
