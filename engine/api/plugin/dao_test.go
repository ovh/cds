package plugin

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestInsertUpdateLoadDelete(t *testing.T) {
	p := sdk.GRPCPlugin{
		Author:      "me",
		Description: "desc",
		Name:        "test_plugin",
		Type:        sdk.GRPCPluginDeploymentPlatform,
		Binaries: []sdk.GRPCPluginBinary{
			{
				OS:   "linux",
				Arch: "adm64",
				Name: "blabla",
			},
		},
	}

	db, _ := test.SetupPG(t)
	test.NoError(t, Insert(db, &p))
	assert.NotEqual(t, 0, p.ID)

	test.NoError(t, Update(db, &p))

	_, err := LoadByName(db, "test_plugin")
	test.NoError(t, err)

	all, err := LoadAll(db)
	test.NoError(t, err)
	assert.Len(t, all, 1)

	test.NoError(t, Delete(db, &p))

}
