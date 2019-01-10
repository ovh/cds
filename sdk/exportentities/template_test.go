package exportentities

import (
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

func TestNewTemplateAndGetTemplate(t *testing.T) {
	template := Template{
		Slug:        "my-template",
		Name:        "My template",
		Group:       "my-group",
		Description: "My description",
		Parameters: []TemplateParameter{
			{Key: "my-boolean", Type: "boolean", Required: true},
			{Key: "my-string", Type: "string", Required: true},
			{Key: "my-repository", Type: "repository", Required: true},
		},
		Workflow: "workflow.yml",
	}
	templateYaml, err := yaml.Marshal(template)
	assert.Nil(t, err)

	sdkTemplate := sdk.WorkflowTemplate{
		Slug:        "my-template",
		Name:        "My template",
		Description: "My description",
		Group: &sdk.Group{
			Name: "my-group",
		},
		Parameters: []sdk.WorkflowTemplateParameter{
			{Key: "my-boolean", Type: "boolean", Required: true},
			{Key: "my-string", Type: "string", Required: true},
			{Key: "my-repository", Type: "repository", Required: true},
		},
	}
	sdkTemplateYaml, err := yaml.Marshal(sdkTemplate)
	assert.Nil(t, err)

	exported, err := NewTemplate(sdkTemplate)
	assert.Nil(t, err)
	exportedYaml, err := yaml.Marshal(exported)
	assert.Nil(t, err)
	assert.Equal(t, templateYaml, exportedYaml)

	imported := template.GetTemplate(nil, nil, nil, nil)
	importedYaml, err := yaml.Marshal(imported)
	assert.Nil(t, err)
	assert.Equal(t, sdkTemplateYaml, importedYaml)
}
