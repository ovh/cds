package action

import (
	"context"
	"os"
	"testing"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunInstallKeyAction(t *testing.T) {
	wk := TestWorker{t}
	wdFS := afero.NewOsFs()
	wdName := sdk.RandomString(10)
	require.NoError(t, wdFS.MkdirAll(wdName, os.FileMode(0755)))
	defer wdFS.RemoveAll(wdName) // nolint

	wdFile, err := wdFS.Open(wdName)
	require.NoError(t, err)

	ctx := workerruntime.SetWorkingDirectory(context.TODO(), wdFile)

	keyInstallAction := sdk.Action{
		Parameters: []sdk.Parameter{
			{
				Name:  "key",
				Type:  sdk.KeyParameter,
				Value: "proj-mykey",
			},
		},
	}
	secrets := []sdk.Variable{
		sdk.Variable{
			ID:    1,
			Name:  "cds.key.proj-mykey.priv",
			Value: "test",
			Type:  sdk.KeyTypeSSH,
		},
	}
	res, err := RunInstallKey(ctx, wk, keyInstallAction, nil, secrets)
	assert.NoError(t, err)
	assert.Equal(t, sdk.StatusSuccess, res.Status)
}
