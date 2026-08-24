package exportentities_test

import (
	"testing"
	"time"

	"github.com/ovh/cds/sdk/exportentities"

	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestWorkerModelEOL checks that the end of life date survives the as-code round trip, and that the
// short YAML date form is accepted on import: an as-code file is hand written, nobody should have to
// spell a full RFC3339 timestamp there.
func TestWorkerModelEOL(t *testing.T) {
	eol := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	t.Run("eol survives the round trip", func(t *testing.T) {
		sdkWm := sdk.Model{
			Name:         "myITModel",
			Type:         "docker",
			Group:        &sdk.Group{Name: "shared.infra"},
			IsDeprecated: true,
			EOL:          &eol,
			ModelDocker:  sdk.ModelDocker{Image: "foo/model/go:latest"},
		}

		exported := exportentities.NewWorkerModel(sdkWm)
		require.NotNil(t, exported.EOL)
		assert.Equal(t, eol, *exported.EOL)

		imported := exported.GetWorkerModel()
		require.NotNil(t, imported.EOL)
		assert.Equal(t, eol, *imported.EOL)
		assert.True(t, imported.IsDeprecated)
	})

	t.Run("a short yaml date is accepted", func(t *testing.T) {
		var wm exportentities.WorkerModel
		require.NoError(t, yaml.Unmarshal([]byte(`
name: myITModel
group: shared.infra
type: docker
image: foo/model/go:latest
is_deprecated: true
eol: 2026-12-31
`), &wm))
		require.NotNil(t, wm.EOL)
		assert.Equal(t, eol, wm.EOL.UTC())
	})

	t.Run("no eol stays absent from the exported yaml", func(t *testing.T) {
		sdkWm := sdk.Model{
			Name:        "myITModel",
			Type:        "docker",
			Group:       &sdk.Group{Name: "shared.infra"},
			ModelDocker: sdk.ModelDocker{Image: "foo/model/go:latest"},
		}
		out, err := yaml.Marshal(exportentities.NewWorkerModel(sdkWm))
		require.NoError(t, err)
		assert.NotContains(t, string(out), "eol")
	})
}
