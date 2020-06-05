package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

func Test_parseEmptyMarathonFileShouldactionpluginFail(t *testing.T) {
	filepath := "./fixtures/marathon1.json"
	app, err := parseApplicationConfigFile(filepath)
	assert.Nil(t, app)
	assert.Error(t, err)
	t.Log(err.Error())
}

func Test_parseInvalidJSONFileShouldactionpluginFail(t *testing.T) {
	filepath := "./fixtures/marathon2.json"
	app, err := parseApplicationConfigFile(filepath)
	assert.Nil(t, app)
	assert.Error(t, err)
	t.Log(err.Error())
}

func Test_parseInvalidMarathonFileShouldactionpluginFail(t *testing.T) {
	filepath := "./fixtures/marathon3.json"
	app, err := parseApplicationConfigFile(filepath)
	assert.Nil(t, app)
	assert.Error(t, err)
	t.Log(err.Error())
}

func Test_parseValidMarathonFileShouldSuccess(t *testing.T) {
	filepath := "./fixtures/marathon4.json"
	app, err := parseApplicationConfigFile(filepath)
	assert.NotNil(t, app)
	assert.NoError(t, err)
	assert.Equal(t, "redis/master", app.ID)
}

func Test_tmplApplicationConfigFile(t *testing.T) {
	filepath := "./fixtures/marathon5.json"

	q := &actionplugin.ActionQuery{
		Options: map[string]string{
			"cds.app.name": "MonApplication",
			"cds.image":    "mon_image:mon_tag/5",
		},
	}
	out, err := tmplApplicationConfigFile(q, filepath)
	t.Logf("file: %s\n", out)
	defer os.RemoveAll(out)

	assert.NoError(t, err)
	assert.NotZero(t, out)

	app, err := parseApplicationConfigFile(out)
	assert.NotNil(t, app)
	assert.NoError(t, err)

	assert.Equal(t, "/monapplication", app.ID)
	assert.Equal(t, "mon-image:mon-tag-5", app.Container.Docker.Image)
}

func Test_tmplApplicationConfigFileX(t *testing.T) {
	filepath := "./fixtures/marathon6.json"
	q := &actionplugin.ActionQuery{
		Options: map[string]string{
			"cds.env.image": "\"toto\"",
		},
	}
	out, err := tmplApplicationConfigFile(q, filepath)
	t.Logf("file: %s\n", out)
	defer os.RemoveAll(out)

	assert.NoError(t, err)
	assert.NotZero(t, out)

	app, err := parseApplicationConfigFile(out)
	assert.NoError(t, err)

	assert.NotNil(t, app)
	assert.Equal(t, "toto", app.ID, "id should be \"toto\"")

}

func Test_Run(t *testing.T) {

}
