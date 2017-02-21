package project

import (
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type dbProject sdk.Project
type dbVariable sdk.Variable

func init() {
	gorpmapping.Register(gorpmapping.New(dbProject{}, "project", true, "id"))
}

// PostGet is a db hook
func (p *dbProject) PostGet(db gorp.SqlExecutor) error {
	metadataStr, err := db.SelectNullStr("select metadata from project where id = $1", p.ID)
	if err != nil {
		return err
	}

	if metadataStr.Valid {
		metadata := sdk.Metadata{}
		if err := json.Unmarshal([]byte(metadataStr.String), &metadata); err != nil {
			return err
		}
		p.Metadata = metadata
	}
	return nil
}

// PostUpdate is a db hook
func (p *dbProject) PostUpdate(db gorp.SqlExecutor) error {
	b, err := json.Marshal(p.Metadata)
	if err != nil {
		return err
	}
	if _, err := db.Exec("update project set metadata = $2 where id = $1", p.ID, b); err != nil {
		return err
	}
	return nil
}

// PostInsert is a db hook
func (p *dbProject) PostInsert(db gorp.SqlExecutor) error {
	return p.PostUpdate(db)
}
