package plugin

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestInsertUpdateLoadDelete(t *testing.T) {
	p := sdk.GRPCPlugin{
		Author:      "me",
		Description: "desc",
		Name:        "test_plugin",
		Type:        sdk.GRPCPluginDeploymentIntegration,
		Binaries: []sdk.GRPCPluginBinary{
			{
				OS:   "linux",
				Arch: "adm64",
				Name: "blabla",
			},
		},
	}

	db, _, end := test.SetupPG(t)
	defer end()
	test.NoError(t, Insert(db, &p))
	assert.NotEqual(t, 0, p.ID)

	test.NoError(t, Update(db, &p))

	_, err := LoadByName(db, "test_plugin")
	test.NoError(t, err)

	all, err := LoadAll(db)
	test.NoError(t, err)
	assert.Len(t, all, 1)

	// Init store
	cfg := objectstore.Config{
		Kind: objectstore.Filesystem,
		Options: objectstore.ConfigOptions{
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: path.Join(os.TempDir(), "store"),
			},
		},
	}

	storage, errO := objectstore.Init(context.Background(), cfg)
	test.NoError(t, errO)

	test.NoError(t, Delete(db, storage, &p))

}
