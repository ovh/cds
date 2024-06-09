package local_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication/local"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestResetConsumerToken(t *testing.T) {
	_, store := test.SetupPG(t, bootstrap.InitializeDB)

	consumerUUID := sdk.UUID()
	token, err := local.NewResetConsumerToken(store, consumerUUID)
	require.NoError(t, err)

	res, err := local.CheckResetConsumerToken(store, token)
	require.NoError(t, err)

	assert.Equal(t, consumerUUID, res)
}
