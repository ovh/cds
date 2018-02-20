package project

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"encoding/json"
	"github.com/ovh/cds/engine/api/platform"
	"github.com/ovh/cds/sdk"
)

// PostGet is a db hook
func (pp *dbProjectPlatform) PostGet(db gorp.SqlExecutor) error {
	model, err := platform.LoadModel(db, pp.PlatformModelID)
	if err != nil {
		return sdk.WrapError(err, "dbProjectPlatform.PostGet> Cannot load model")
	}
	pp.Model = model
	return nil
}

// LoadPlatformsByID load project platforms by project id
func LoadPlatformsByID(db gorp.SqlExecutor, id int64) ([]sdk.ProjectPlatform, error) {
	platforms := []sdk.ProjectPlatform{}

	var res []dbProjectPlatform
	if _, err := db.Select(&res, "SELECT * from project_platform WHERE project_id = $1", id); err != nil {
		if err == sql.ErrNoRows {
			return platforms, nil
		}
		return nil, err
	}

	platforms = make([]sdk.ProjectPlatform, len(res))
	for i := range res {
		pp := &res[i]
		if err := pp.PostGet(db); err != nil {
			return nil, sdk.WrapError(err, "LoadPlatformByID> Cannt post get")
		}
		platforms[i] = sdk.ProjectPlatform(*pp)
	}

	return platforms, nil
}

// InsertPlatform inserts a project platform
func InsertPlatform(db gorp.SqlExecutor, pp *sdk.ProjectPlatform) error {
	ppDb := dbProjectPlatform(*pp)
	if err := db.Insert(&ppDb); err != nil {
		return sdk.WrapError(err, "InsertPlatform> Cannot insert project platform")
	}
	*pp = sdk.ProjectPlatform(ppDb)
	return nil
}

// PostInsert is a db hook
func (pp *dbProjectPlatform) PostInsert(db gorp.SqlExecutor) error {
	configB, err := json.Marshal(pp.Config)
	if err != nil {
		return sdk.WrapError(err, "PostInsert.projectPlatform> Cannot post insert project platform")
	}

	if _, err := db.Exec("UPDATE project_platform set config = $1 WHERE id = $2", string(configB), pp.ID); err != nil {
		return sdk.WrapError(err, "PostInsert.projectPlatform> Cannot update config")
	}
	return nil
}
