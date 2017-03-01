package main

import (
	"os"
	"testing"

	"github.com/ovh/cds/sdk/plugin"
	"github.com/stretchr/testify/assert"
)

func Test_parseEmptyMarathonFileShouldFail(t *testing.T) {
	plugin.SetTrace(os.Stdout)

	filepath := "./fixtures/marathon1.json"
	app, err := parseApplicationConfigFile(nil, filepath)
	assert.Nil(t, app)
	assert.Error(t, err)
	t.Log(err.Error())
}

func Test_parseInvalidJSONFileShouldFail(t *testing.T) {
	plugin.SetTrace(os.Stdout)

	filepath := "./fixtures/marathon2.json"
	app, err := parseApplicationConfigFile(nil, filepath)
	assert.Nil(t, app)
	assert.Error(t, err)
	t.Log(err.Error())
}

func Test_parseInvalidMarathonFileShouldFail(t *testing.T) {
	plugin.SetTrace(os.Stdout)

	filepath := "./fixtures/marathon3.json"
	app, err := parseApplicationConfigFile(nil, filepath)
	assert.Nil(t, app)
	assert.Error(t, err)
	t.Log(err.Error())
}

func Test_parseValidMarathonFileShouldSuccess(t *testing.T) {
	plugin.SetTrace(os.Stdout)

	filepath := "./fixtures/marathon4.json"
	app, err := parseApplicationConfigFile(nil, filepath)
	assert.NotNil(t, app)
	assert.NoError(t, err)
	assert.Equal(t, "redis/master", app.ID)
}

func Test_tmplApplicationConfigFile(t *testing.T) {
	plugin.SetTrace(os.Stdout)

	filepath := "./fixtures/marathon5.json"
	a := plugin.Action{
		Args: plugin.Arguments{
			Data: map[string]string{
				"cds.app.name": "MonApplication",
				"cds.image":    "mon_image:mon_tag/5",
			},
		},
	}
	out, err := tmplApplicationConfigFile(&a, filepath)
	t.Logf("file: %s\n", out)
	defer os.RemoveAll(out)

	assert.NoError(t, err)
	assert.NotZero(t, out)

	app, err := parseApplicationConfigFile(nil, out)
	assert.NotNil(t, app)
	assert.NoError(t, err)

	assert.Equal(t, "/monapplication", app.ID)
	assert.Equal(t, "mon-image:mon-tag-5", app.Container.Docker.Image)
}

func Test_tmplApplicationConfigFileX(t *testing.T) {
	plugin.SetTrace(os.Stdout)

	filepath := "./fixtures/marathon6.json"
	a := plugin.Action{
		Args: plugin.Arguments{
			Data: map[string]string{
				"cds.env.image": "\"toto\"",
			},
		},
	}
	out, err := tmplApplicationConfigFile(&a, filepath)
	t.Logf("file: %s\n", out)
	defer os.RemoveAll(out)

	assert.NoError(t, err)
	assert.NotZero(t, out)

	app, err := parseApplicationConfigFile(nil, out)
	assert.NoError(t, err)

	assert.NotNil(t, app)
	assert.Equal(t, "toto", app.ID, "id should be \"toto\"")

}

func Test_Run(t *testing.T) {

}
