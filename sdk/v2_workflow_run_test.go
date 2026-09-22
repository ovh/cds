package sdk

import (
	"encoding/json"
	"strconv"
	"strings"
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

func TestV2WorkflowRunResultVariableDetail_CheckValueSize(t *testing.T) {
	// A value of exactly the limit is allowed
	atLimit := V2WorkflowRunResultVariableDetail{Name: "foo", Value: strings.Repeat("a", MaxV2WorkflowRunResultVariableValueSize)}
	require.NoError(t, atLimit.CheckValueSize())

	// One byte more is rejected, and the error must tell the user which output is too large and what the limit is
	overLimit := V2WorkflowRunResultVariableDetail{Name: "foo", Value: strings.Repeat("a", MaxV2WorkflowRunResultVariableValueSize+1)}
	err := overLimit.CheckValueSize()
	require.Error(t, err)
	require.Equal(t, ErrInvalidData.ID, ExtractHTTPError(err).ID)
	require.Contains(t, err.Error(), "foo")
	require.Contains(t, err.Error(), strconv.Itoa(MaxV2WorkflowRunResultVariableValueSize))

	// The limit is a number of bytes, not a number of runes: 513 two-bytes runes are 1026 bytes
	multibyte := V2WorkflowRunResultVariableDetail{Name: "foo", Value: strings.Repeat("é", MaxV2WorkflowRunResultVariableValueSize/2+1)}
	require.Error(t, multibyte.CheckValueSize())
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
