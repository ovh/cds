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
