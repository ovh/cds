package exportentities_test

import (
	"testing"

	"github.com/ovh/cds/sdk/exportentities"

	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestNewWorkerModelAndGetWorkerModel(t *testing.T) {
	wm := exportentities.WorkerModel{
		Name:        "myITModel",
		Type:        "docker",
		Description: "my worker model",
		Group:       "shared.infra",
		Image:       "foo/model/go:latest",
		Shell:       "sh -c",
		Cmd:         "worker --api={{.API}} --token={{.Token}} --basedir={{.BaseDir}} --model={{.Model}} --name={{.Name}} --hatchery={{.Hatchery}} --hatchery-name={{.HatcheryName}} --insecure={{.HTTPInsecure}} --single-use",
	}
	wmYaml, err := yaml.Marshal(wm)
	test.NoError(t, err)

	sdkWm := sdk.Model{
		Name:        "myITModel",
		Type:        "docker",
		Description: "my worker model",
		Group:       &sdk.Group{Name: "shared.infra"},
		ModelDocker: sdk.ModelDocker{
			Image: "foo/model/go:latest",
			Shell: "sh -c",
			Cmd:   "worker --api={{.API}} --token={{.Token}} --basedir={{.BaseDir}} --model={{.Model}} --name={{.Name}} --hatchery={{.Hatchery}} --hatchery-name={{.HatcheryName}} --insecure={{.HTTPInsecure}} --single-use",
		},
	}
	sdkWmYaml, err := yaml.Marshal(sdkWm)
	test.NoError(t, err)

	exported := exportentities.NewWorkerModel(sdkWm)
	exportedYaml, err := yaml.Marshal(exported)
	assert.Nil(t, err)
	assert.Equal(t, string(wmYaml), string(exportedYaml))

	imported := wm.GetWorkerModel()
	importedYaml, err := yaml.Marshal(imported)
	test.NoError(t, err)
	assert.Equal(t, string(sdkWmYaml), string(importedYaml))
}
