package workermodel

import (
	"database/sql"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

type dbResultWMS struct {
	WorkerModel
	GroupName string `db:"groupname"`
}

// loadAll retrieves a list of worker model in database.
func loadAll(db gorp.SqlExecutor, withPassword bool, query string, args ...interface{}) ([]sdk.Model, error) {
	wms := []dbResultWMS{}
	if _, err := db.Select(&wms, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrNoWorkerModel)
		}
		return nil, sdk.WithStack(err)
	}
	if len(wms) == 0 {
		return []sdk.Model{}, nil
	}
	r, err := scanAll(db, wms, withPassword)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// load retrieves a specific worker model in database.
func load(db gorp.SqlExecutor, withPassword bool, query string, args ...interface{}) (*sdk.Model, error) {
	wms := []dbResultWMS{}
	if _, err := db.Select(&wms, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WithStack(sdk.ErrNoWorkerModel)
		}
		return nil, err
	}
	if len(wms) == 0 {
		return nil, sdk.WithStack(sdk.ErrNoWorkerModel)
	}
	r, err := scanAll(db, wms, withPassword)
	if err != nil {
		return nil, err
	}
	if len(r) != 1 {
		return nil, sdk.WithStack(fmt.Errorf("worker model not unique"))
	}
	return &r[0], nil
}
