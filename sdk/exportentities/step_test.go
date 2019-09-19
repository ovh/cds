package exportentities_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk/exportentities"
)

var testInstallKey = exportentities.StepInstallKey("proj-mykey")
var testAdvancedInstallKey = exportentities.StepInstallKey(map[string]string{"name": "proj-mykey", "file": "myfile"})
var tests = []struct {
	Name string
	Step exportentities.Step
	Json string
	Yaml string
}{
	{
		Name: "Step with custom action",
		Step: exportentities.Step{
			StepCustom: exportentities.StepCustom{
				"group/action": map[string]string{
					"param1": "value1",
					"param2": "value2",
				},
			},
		},
		Json: `{"group/action":{"param1":"value1","param2":"value2"}}`,
		Yaml: "group/action:\n  param1: value1\n  param2: value2\n",
	},
	{
		Name: "Step with typed action",
		Step: exportentities.Step{
			ArtifactDownload: &exportentities.StepArtifactDownload{
				Path: "{{.cds.workspace}}",
			},
		},
		Json: `{"artifactDownload":{"path":"{{.cds.workspace}}"}}`,
		Yaml: "artifactDownload:\n  path: '{{.cds.workspace}}'\n",
	},
	{
		Name: "Step with typed action install key",
		Step: exportentities.Step{
			InstallKey: &testInstallKey,
		},
		Json: `{"installKey":"proj-mykey"}`,
		Yaml: "installKey: proj-mykey\n",
	},
	{
		Name: "Step with typed action install key advanced parameters",
		Step: exportentities.Step{
			InstallKey: &testAdvancedInstallKey,
		},
		Json: `{"installKey":{"file":"myfile","name":"proj-mykey"}}`,
		Yaml: "installKey:\n  file: myfile\n  name: proj-mykey\n",
	},
	{
		Name: "Step with not typed action",
		Step: exportentities.Step{
			Script: []interface{}{
				"line1",
				"line2",
			},
		},
		Json: `{"script":["line1","line2"]}`,
		Yaml: "script:\n- line1\n- line2\n",
	},
}

func TestMarshal(t *testing.T) {
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			buf, err := json.Marshal(test.Step)
			assert.NoError(t, err)
			assert.Equal(t, test.Json, string(buf), "Invalid json output")

			buf, err = yaml.Marshal(test.Step)
			assert.NoError(t, err)
			assert.Equal(t, test.Yaml, string(buf), "Invalid yaml output")
		})
	}
}

func TestUnMarshal(t *testing.T) {
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var step exportentities.Step

			assert.NoError(t, json.Unmarshal([]byte(test.Json), &step))
			assert.Equal(t, test.Step.String(), step.String(), "Invalid json unmarshal")

			assert.NoError(t, yaml.Unmarshal([]byte(test.Yaml), &step))
			assert.Equal(t, test.Step.String(), step.String(), "Invalid yaml unmarshal")
		})
	}
}
