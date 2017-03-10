package main

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/template"
)

func TestTemplateCall(t *testing.T) {
	if _, err := os.Stat("../testtemplate"); os.IsNotExist(err) {
		t.SkipNow()
	}

	client := template.NewClient("testtemplate", "../testtemplate", "ID", "http://localhost:8081", true)
	defer client.Kill()

	_template, err := client.Instance()
	if err != nil {
		log.Fatal(err)
	}

	t.Log("Template initialized")

	assert.Equal(t, "testtemplate", _template.Name())
	assert.Equal(t, "François Samin <francois.samin@corp.ovh.com>", _template.Author())
	assert.Equal(t, "github.com/ovh/cds/sdk/template/TestTemplate", _template.Identifier())
	assert.Equal(t, "Description", _template.Description())
	assert.EqualValues(t, []sdk.TemplateParam{
		{
			Name:  "param1",
			Type:  sdk.StringVariable,
			Value: "value1",
		},
		{
			Name:  "param2",
			Type:  sdk.StringVariable,
			Value: "value2",
		},
	}, _template.Parameters())
	assert.Equal(t, "BUILD", _template.Type())

	p := _template.Parameters()
	assert.Equal(t, 2, len(p))

	params := template.NewParameters(map[string]string{})
	app, err := _template.Apply(template.NewApplyOptions("proj", "app", *params))
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(t, "myApp", app.Name)

}
