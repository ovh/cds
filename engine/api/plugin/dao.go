package plugin

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/sdk"
)

// Insert inserts a plugin
func Insert(db gorp.SqlExecutor, p *sdk.GRPCPlugin) error {
	for i := range p.Binaries {
		p.Binaries[i].FileContent = nil
		p.Binaries[i].PluginName = p.Name
	}
	return sdk.WrapError(gorpmapping.Insert(db, p), "unable to insert plugin %q", p.Name)
}

// Update updates a plugin
func Update(db gorp.SqlExecutor, p *sdk.GRPCPlugin) error {
	for i := range p.Binaries {
		p.Binaries[i].FileContent = nil
		p.Binaries[i].PluginName = p.Name
	}
	return sdk.WrapError(gorpmapping.Update(db, p), "unable to update plugin %q", p.Name)
}

// Delete deletes a plugin
func Delete(ctx context.Context, db gorp.SqlExecutor, storageDriver objectstore.Driver, p *sdk.GRPCPlugin) error {
	for _, b := range p.Binaries {
		if err := storageDriver.Delete(ctx, b); err != nil {
			log.Error(ctx, "plugin.Delete> unable to delete binary %v", b.ObjectPath)
		}
	}
	return sdk.WrapError(gorpmapping.Delete(db, p), "unable to delete plugin %q", p.Name)
}

func getAll(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) ([]sdk.GRPCPlugin, error) {
	pps := []*sdk.GRPCPlugin{}

	if err := gorpmapping.GetAll(ctx, db, q, &pps); err != nil {
		return nil, sdk.WrapError(err, "cannot get plugins")
	}
	if len(pps) > 0 {
		for i := range opts {
			if err := opts[i](ctx, db, pps...); err != nil {
				return nil, err
			}
		}
	}

	ps := make([]sdk.GRPCPlugin, len(pps))
	for i := range pps {
		ps[i] = *pps[i]
	}

	return ps, nil
}

func get(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadOptionFunc) (*sdk.GRPCPlugin, error) {
	var p sdk.GRPCPlugin

	found, err := gorpmapping.Get(ctx, db, q, &p)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get plugin")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	for i := range opts {
		if err := opts[i](ctx, db, &p); err != nil {
			return nil, err
		}
	}

	return &p, nil
}

// LoadAll GRPC plugins.
func LoadAll(ctx context.Context, db gorp.SqlExecutor) ([]sdk.GRPCPlugin, error) {
	query := gorpmapping.NewQuery("SELECT * FROM grpc_plugin")
	return getAll(ctx, db, query, LoadOptions.WithIntegrationModelName)
}

func LoadAllByType(ctx context.Context, db gorp.SqlExecutor, pluginType string) ([]sdk.GRPCPlugin, error) {
	query := gorpmapping.NewQuery("SELECT * FROM grpc_plugin WHERE type = $1 ").Args(pluginType)
	return getAll(ctx, db, query)
}

// LoadAllByIntegrationModelID load all GRPC plugins for given integration model id.
func LoadAllByIntegrationModelID(ctx context.Context, db gorp.SqlExecutor, integrationModelID int64) ([]sdk.GRPCPlugin, error) {
	query := gorpmapping.NewQuery("SELECT * FROM grpc_plugin WHERE integration_model_id = $1").Args(integrationModelID)
	return getAll(ctx, db, query, LoadOptions.WithIntegrationModelName)
}

// LoadByName retrieves in database the plugin with given name.
func LoadByName(ctx context.Context, db gorp.SqlExecutor, name string) (*sdk.GRPCPlugin, error) {
	query := gorpmapping.NewQuery("SELECT * FROM grpc_plugin WHERE name = $1").Args(name)
	return get(ctx, db, query, LoadOptions.WithIntegrationModelName)
}

// LoadByIntegrationModelIDAndType retrieves in database a single plugin associated to a integration model id with a specified type.
func LoadByIntegrationModelIDAndType(ctx context.Context, db gorp.SqlExecutor, integrationModelID int64, typePlugin string) (*sdk.GRPCPlugin, error) {
	query := gorpmapping.NewQuery("SELECT * FROM grpc_plugin WHERE integration_model_id = $1 AND type = $2").Args(integrationModelID, typePlugin)
	return get(ctx, db, query, LoadOptions.WithIntegrationModelName)
}
