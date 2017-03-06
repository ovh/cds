package main

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk/template"
)

func TestTemplateCall(t *testing.T) {
	client := template.NewClient("cds-template-plain", "../cds-template-plain", "ID", "http://localhost:8081", true)
	defer client.Kill()

	_template, err := client.Instance()
	if err != nil {
		log.Fatal(err)
	}

	t.Log("Template initialized")

	assert.Equal(t, "cds-template-plain", _template.Name())
	assert.Equal(t, "Yvonnick Esnault <yvonnick.esnault@corp.ovh.com>", _template.Author())
	assert.Equal(t, "github.com/ovh/cds/contrib/templates/cds-template-plain/TemplatePlain", _template.Identifier())
	assert.Equal(t, "BUILD", _template.Type())

	p := _template.Parameters()
	assert.Equal(t, 3, len(p))

	params := template.NewParameters(map[string]string{})
	app, err := _template.Apply(template.NewApplyOptions("proj", "app", *params))
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(t, "app", app.Name)

}
