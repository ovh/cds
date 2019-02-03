package plugin

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Insert inserts a plugin
func Insert(db gorp.SqlExecutor, p *sdk.GRPCPlugin) error {
	m := grpcPlugin(*p)
	if err := db.Insert(&m); err != nil {
		return sdk.WrapError(err, "plugin.Insert")
	}
	*p = sdk.GRPCPlugin(m)
	return nil
}

// Update updates a plugin
func Update(db gorp.SqlExecutor, p *sdk.GRPCPlugin) error {
	m := grpcPlugin(*p)
	if _, err := db.Update(&m); err != nil {
		return sdk.WrapError(err, "plugin.Update")
	}
	*p = sdk.GRPCPlugin(m)
	return nil
}

func (p *grpcPlugin) PostInsert(db gorp.SqlExecutor) error {
	return p.PostUpdate(db)
}

func (p *grpcPlugin) PostUpdate(db gorp.SqlExecutor) error {
	for i := range p.Binaries {
		p.Binaries[i].FileContent = nil
		p.Binaries[i].PluginName = p.Name
	}
	s, err := gorpmapping.JSONToNullString(p.Binaries)
	if err != nil {
		return sdk.WrapError(err, "unable to marshal data")
	}

	if _, err := db.Exec("UPDATE grpc_plugin SET binaries = $2 WHERE id = $1", p.ID, s); err != nil {
		return sdk.WrapError(err, "unable to update data")
	}

	return nil
}

// Delete deletes a plugin
func Delete(db gorp.SqlExecutor, p *sdk.GRPCPlugin) error {
	for _, b := range p.Binaries {
		if err := objectstore.Delete(b); err != nil {
			log.Error("plugin.Delete> unable to delete binary %v", b.ObjectPath)
		}
	}

	m := grpcPlugin(*p)
	if _, err := db.Delete(&m); err != nil {
		return sdk.WrapError(err, "plugin.Delete")
	}
	return nil
}

// LoadByName loads a plugin by name
func LoadByName(db gorp.SqlExecutor, name string) (*sdk.GRPCPlugin, error) {
	m := grpcPlugin{}
	if err := db.SelectOne(&m, "SELECT * FROM grpc_plugin WHERE NAME = $1", name); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "plugin %s not found", name)
		}
		return nil, sdk.WrapError(err, "plugin.LoadByName")
	}
	if err := m.PostGet(db); err != nil {
		return nil, sdk.WrapError(err, "plugin.LoadByName")
	}
	p := sdk.GRPCPlugin(m)
	return &p, nil
}

// LoadByIntegrationModelIDAndType loads a single plugin associated to a integration model id with a specified type
func LoadByIntegrationModelIDAndType(db gorp.SqlExecutor, integrationModelID int64, typePlugin string) (*sdk.GRPCPlugin, error) {
	m := grpcPlugin{}
	if err := db.SelectOne(&m, "SELECT * FROM grpc_plugin where integration_model_id = $1 and type = $2", integrationModelID, typePlugin); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "plugin not found (type: %s) for integration %d", typePlugin, integrationModelID)
		}
		return nil, sdk.WrapError(err, "plugin.LoadByIntegrationModelIDAndType")
	}
	if err := m.PostGet(db); err != nil {
		return nil, sdk.WrapError(err, "plugin.LoadByIntegrationModelIDAndType")
	}
	p := sdk.GRPCPlugin(m)
	return &p, nil
}

// LoadAllByIntegrationModelID loads all plugins associated to a integration model id
func LoadAllByIntegrationModelID(db gorp.SqlExecutor, integrationModelID int64) ([]sdk.GRPCPlugin, error) {
	m := []grpcPlugin{}
	if _, err := db.Select(&m, "SELECT * FROM grpc_plugin where integration_model_id = $1", integrationModelID); err != nil {
		return nil, sdk.WrapError(err, "plugin.LoadAllByIntegrationModelID")
	}
	res := make([]sdk.GRPCPlugin, len(m))
	for i := range m {
		p := m[i]
		if err := p.PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "LoadAllByIntegrationModelID")
		}
		res[i] = sdk.GRPCPlugin(p)
	}
	return res, nil
}

// LoadAll loads all GRPC Plugins
func LoadAll(db gorp.SqlExecutor) ([]sdk.GRPCPlugin, error) {
	m := []grpcPlugin{}
	if _, err := db.Select(&m, "SELECT * FROM grpc_plugin"); err != nil {
		return nil, sdk.WrapError(err, "plugin.LoadAll")
	}

	res := make([]sdk.GRPCPlugin, len(m))
	for i := range m {
		p := m[i]
		if err := p.PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "LoadAll")
		}
		res[i] = sdk.GRPCPlugin(p)
	}

	return res, nil
}

func (p *grpcPlugin) PostGet(db gorp.SqlExecutor) error {
	s, err := db.SelectNullStr("SELECT binaries FROM grpc_plugin WHERE ID = $1", p.ID)
	if err != nil {
		return sdk.WrapError(err, "unable to get binaries for ID=%d", p.ID)
	}
	if err := gorpmapping.JSONNullString(s, &p.Binaries); err != nil {
		return sdk.WrapError(err, "plugin.PostGet")
	}
	if p.IntegrationModelID != nil {
		var err error
		p.Integration, err = db.SelectStr("SELECT name FROM integration_model WHERE ID = $1", p.IntegrationModelID)
		if err != nil {
			return sdk.WrapError(err, "unable to get integration name for ID=%d", p.IntegrationModelID)
		}
	}
	return nil
}
