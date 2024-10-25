package sdk

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntegConfigToJobContext(t *testing.T) {
	pi := ProjectIntegration{
		Config: IntegrationConfig{},
		Model:  ArtifactoryIntegration,
	}
	pi.Config["build.info.prefix"] = IntegrationConfigValue{
		Value: "pref",
	}
	pi.Config["build.info.toto"] = IntegrationConfigValue{
		Value: "tata",
	}
	pi.Config["url"] = IntegrationConfigValue{
		Value: "myurl",
	}
	pi.Config["token"] = IntegrationConfigValue{
		Value: "mytoken",
	}
	pi.Config["token.name"] = IntegrationConfigValue{
		Value: "myuser",
	}

	result := pi.ToJobRunContextConfig()

	require.Equal(t, result["url"], "myurl")
	require.NotNil(t, result["build"])

	buildMap := result["build"].(map[string]interface{})
	require.NotNil(t, buildMap["info"])

	infoMap := buildMap["info"].(map[string]interface{})
	require.Equal(t, infoMap["prefix"], "pref")
	require.Equal(t, infoMap["toto"], "tata")

	require.Equal(t, result["token"], "mytoken")
	require.Equal(t, result["token_name"], "myuser")
}
