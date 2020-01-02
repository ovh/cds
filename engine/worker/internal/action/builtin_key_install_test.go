package action

import (
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestRunInstallKeyAction(t *testing.T) {
	wk, ctx := setupTest(t)

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
	res, err := RunInstallKey(ctx, wk, keyInstallAction, secrets)
	assert.NoError(t, err)
	assert.Equal(t, sdk.StatusSuccess, res.Status)
}
