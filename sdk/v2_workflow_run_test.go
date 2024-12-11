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
