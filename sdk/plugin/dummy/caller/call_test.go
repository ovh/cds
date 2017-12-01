package main

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk/plugin"
)

func TestDummyPlugin(t *testing.T) {
	if _, err := os.Stat("../dummy"); os.IsNotExist(err) {
		t.SkipNow()
	}
	client := plugin.NewClient(context.Background(), "dummy", "../dummy", "ID", "http://localhost:8081", true)
	defer client.Kill()

	_plugin, err := client.Instance()
	if err != nil {
		log.Fatal(err)
	}

	assert.Equal(t, "dummy", _plugin.Name())
	assert.Equal(t, "François SAMIN <francois.samin@corp.ovh.com>", _plugin.Author())
	assert.Equal(t, "This is a dummy plugin", _plugin.Description())

	p := _plugin.Parameters()
	assert.Equal(t, "value1", p.GetValue("param1"))

	a := plugin.Job{
		IDPipelineBuild:    0,
		IDPipelineJobBuild: 0,
		Args:               plugin.Arguments{},
	}
	assert.Equal(t, "Fail", string(_plugin.Run(a)))

}
