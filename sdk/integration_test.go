package sdk

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntegConfigToJobContext(t *testing.T) {
	pi := ProjectIntegration{
		Name:   "foo",
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
	pi.Config["my_token"] = IntegrationConfigValue{
		Value: PasswordPlaceholder,
		Type:  SecretVariable,
	}
	pi.Model.PublicConfigurations = make(IntegrationConfigMap)
	pi.Model.PublicConfigurations["foo"] = IntegrationConfig{
		"my_token": IntegrationConfigValue{
			Value: "the_value_token",
			Type:  IntegrationConfigTypePassword,
		},
	}

	result := pi.ToJobRunContextConfig()

	require.Equal(t, result["url"], "myurl")
	require.NotNil(t, result["build"])

	buildMap := result["build"].(map[string]interface{})
	require.NotNil(t, buildMap["info"])

	infoMap := buildMap["info"].(map[string]interface{})
	require.Equal(t, "pref", infoMap["prefix"])
	require.Equal(t, "tata", infoMap["toto"])

	require.Equal(t, "mytoken", result["token"])
	require.Equal(t, "myuser", result["token_name"])

	t.Logf("result: %+v", result)
	require.Equal(t, "the_value_token", result["my_token"])
}
