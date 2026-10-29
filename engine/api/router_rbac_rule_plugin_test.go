package api

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

// A hatchery reads the plugin descriptors to preload their binaries in its
// workers; a consumer that is neither a hatchery, a worker nor a user cannot
func TestRbacPluginRead(t *testing.T) {
	api := &API{}

	require.Error(t, api.pluginRead(context.TODO(), nil))

	ctx := context.WithValue(context.TODO(), contextHatcheryConsumer, &sdk.AuthHatcheryConsumer{})
	require.NoError(t, api.pluginRead(ctx, nil))
}
