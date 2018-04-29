package plugin

import (
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/sdk/log"

	"github.com/ovh/cds/sdk"
)

func Insert(db gorp.SqlExecutor, p *sdk.GRPCPlugin) error {
	m := grpcPlugin(*p)
	if err := db.Insert(&m); err != nil {
		return sdk.WrapError(err, "plugin.Insert")
	}
	*p = sdk.GRPCPlugin(m)
	return nil
}

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
	s, err := gorpmapping.JSONToNullString(p.Binaries)
	if err != nil {
		return sdk.WrapError(err, "plugin.PostUpdate> unable to marshal data")
	}

	if _, err := db.Exec("UPDATE grpc_plugin SET binaries = $2 WHERE id = $1", p.ID, s); err != nil {
		return sdk.WrapError(err, "plugin.PostUpdate> unable to update data")
	}

	return nil
}

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

func LoadByName(db gorp.SqlExecutor, name string) (*sdk.GRPCPlugin, error) {
	m := grpcPlugin{}
	if err := db.SelectOne(&m, "SELECT * FROM grpc_plugin WHERE NAME = $1", name); err != nil {
		return nil, sdk.WrapError(err, "plugin.LoadByName")
	}
	p := sdk.GRPCPlugin(m)
	return &p, nil
}

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
		return sdk.WrapError(err, "plugin.PostGet> unable to get binaries for ID=%d", p.ID)
	}
	if err := gorpmapping.JSONNullString(s, &p.Binaries); err != nil {
		return sdk.WrapError(err, "plugin.PostGet")
	}
	return nil
}
