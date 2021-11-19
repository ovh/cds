package plugin

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadOptionFunc for plugin.
type LoadOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.GRPCPlugin) error

// LoadOptions provides all options on plugin loads functions.
var LoadOptions = struct {
	WithIntegrationModelName LoadOptionFunc
}{
	WithIntegrationModelName: loadIntegrationModelName,
}

func loadIntegrationModelName(ctx context.Context, db gorp.SqlExecutor, ps ...*sdk.GRPCPlugin) error {
	for _, p := range ps {
		if p.IntegrationModelID != nil {
			var err error
			p.Integration, err = db.SelectStr("SELECT name FROM integration_model WHERE id = $1", p.IntegrationModelID)
			if err != nil {
				return sdk.WrapError(err, "unable to get integration model name for id %d", p.IntegrationModelID)
			}
		}
	}
	return nil
}
