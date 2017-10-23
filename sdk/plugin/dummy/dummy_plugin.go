package main

import (
	"os"

	"github.com/ovh/cds/sdk/plugin"
)

type DummyPlugin struct {
	plugin.Common
}

func (d DummyPlugin) Name() string        { return "plugin-dummy" }
func (d DummyPlugin) Description() string { return "This is a dummy plugin" }
func (d DummyPlugin) Author() string      { return "François SAMIN <francois.samin@corp.ovh.com>" }

//Parameters return parameters description
func (d DummyPlugin) Parameters() plugin.Parameters {
	params := plugin.NewParameters()
	params.Add("param1", plugin.StringParameter, "this is a parameter", "value1")
	return params
}

//Run execute the action
func (d DummyPlugin) Run(a plugin.IJob) plugin.Result {
	envVariables := os.Environ()
	for _, s := range envVariables {
		plugin.SendLog(a, "PLUGIN DEBUG ENV : %s\n", s)
	}

	err := plugin.SendLog(a, "PLUGIN", "This is a log from %\n", d.Name())
	if err != nil {
		return plugin.Fail
	}
	return plugin.Success
}

func main() {
	plugin.Main(&DummyPlugin{})
}
