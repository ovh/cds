package application

import (
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type dbApplication sdk.Application
type dbVariable sdk.Variable

func init() {
	gorpmapping.Register(gorpmapping.New(dbApplication{}, "application", true, "id"))
}

// PostGet is a db hook
func (a *dbApplication) PostGet(db gorp.SqlExecutor) error {
	metadataStr, err := db.SelectNullStr("select metadata from application where id = $1", a.ID)
	if err != nil {
		return err
	}

	if metadataStr.Valid {
		metadata := sdk.Metadata{}
		if err := json.Unmarshal([]byte(metadataStr.String), &metadata); err != nil {
			return err
		}
		a.Metadata = metadata
	}

	pkey, errP := db.SelectStr("select projectkey from project where id = $1", a.ProjectID)
	if errP != nil {
		return errP
	}

	a.ProjectKey = pkey
	return nil
}

// PostUpdate is a db hook
func (a *dbApplication) PostUpdate(db gorp.SqlExecutor) error {
	b, err := json.Marshal(a.Metadata)
	if err != nil {
		return err
	}
	if _, err := db.Exec("update application set metadata = $2 where id = $1", a.ID, b); err != nil {
		return err
	}
	return nil
}

// PostInsert is a db hook
func (a *dbApplication) PostInsert(db gorp.SqlExecutor) error {
	return a.PostUpdate(db)
}
