package sdk

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetContextConfig(t *testing.T) {

	j := JobIntegrationsContext{
		ModelName: ArtifactoryIntegrationModelName,
		Config: JobIntegratiosContextConfig{
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
