package sdk

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntegConfigToJobContext(t *testing.T) {
	config := IntegrationConfig{}
	config["build.info.prefix"] = IntegrationConfigValue{
		Value: "pref",
	}
	config["build.info.toto"] = IntegrationConfigValue{
		Value: "tata",
	}
	config["url"] = IntegrationConfigValue{
		Value: "myurl",
	}

	result := config.ToJobRunContextConfig()

	require.Equal(t, result["url"], "myurl")
	require.NotNil(t, result["build"])

	buildMap := result["build"].(map[string]interface{})
	require.NotNil(t, buildMap["info"])

	infoMap := buildMap["info"].(map[string]interface{})
	require.Equal(t, infoMap["prefix"], "pref")
	require.Equal(t, infoMap["toto"], "tata")
}
