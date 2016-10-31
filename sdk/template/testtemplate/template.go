package main

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/plugin"
	"github.com/ovh/cds/sdk/template"
)

type TestTemplate struct {
	template.Common
}

func (t *TestTemplate) Init(plugin.IOptions) string {
	return "init OK"
}

func (t *TestTemplate) Name() string {
	return "testtemplate"
}

func (t *TestTemplate) Description() string {
	return "Description"
}

func (t *TestTemplate) Identifier() string {
	return "github.com/ovh/cds/sdk/template/TestTemplate"
}

func (t *TestTemplate) Author() string {
	return "François Samin <francois.samin@corp.ovh.com>"
}

func (t *TestTemplate) Type() string {
	return "BUILD"
}

func (t *TestTemplate) Parameters() []sdk.TemplateParam {
	return []sdk.TemplateParam{
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
	}
}

func (t *TestTemplate) Apply(opts template.IApplyOptions) (sdk.Application, error) {
	return sdk.Application{
		Name: "myApp",
	}, nil
}

func main() {
	p := TestTemplate{}
	template.Serve(&p)
}
