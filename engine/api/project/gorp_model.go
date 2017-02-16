package project

import (
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type dbProject sdk.Project

func init() {
	gorpmapping.Register(gorpmapping.New(dbProject{}, "project", true, "id"))
}

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
