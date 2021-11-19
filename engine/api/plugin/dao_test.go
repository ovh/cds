package plugin

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
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

	db, _ := test.SetupPG(t)

	require.NoError(t, Insert(db, &p))
	assert.NotEqual(t, 0, p.ID)

	require.NoError(t, Update(db, &p))

	_, err := LoadByName(context.TODO(), db, "test_plugin")
	test.NoError(t, err)

	all, err := LoadAll(context.TODO(), db)
	require.NoError(t, err)
	assert.True(t, len(all) >= 1)
	var found bool
	for i := range all {
		if all[i].ID == p.ID {
			found = true
			break
		}
	}
	assert.True(t, found)

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

	test.NoError(t, Delete(context.TODO(), db, storage, &p))

}
