package sdk

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetContextConfig(t *testing.T) {
	j := JobIntegrationsContext{
		Config: JobIntegratiosContextConfig{
			"repo": map[string]interface{}{
				"build": map[string]interface{}{
					"info": map[string]interface{}{
						"data": "foo",
					},
				},
			},
			"url": "myurl",
		},
	}
	require.Equal(t, "myurl", j.Config.Get("url"))
	require.Equal(t, "foo", j.Config.Get("repo.build.info.data"))
}
