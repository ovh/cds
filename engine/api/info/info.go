package info

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// InsertInfo insert a new worker info in database
func InsertInfo(db gorp.SqlExecutor, info sdk.Info) error {
	dbmsg := Info(info)
	if err := db.Insert(&dbmsg); err != nil {
		return err
	}
	info = sdk.Info(dbmsg)
	return nil
}

// UpdateInfo update a worker model. If worker info have SpawnErr -> clear them
func UpdateInfo(db gorp.SqlExecutor, info sdk.Info) error {
	info.Updated = time.Now()
	dbmsg := Info(info)
	if _, err := db.Update(&dbmsg); err != nil {
		return err
	}
	return nil
}

// LoadInfoByID loads info by id
func LoadInfoByID(db gorp.SqlExecutor, id int64) (*sdk.Info, error) {
	dbInfo := Info{}
	query := fmt.Sprintf(`select * from info where id=$1`)
	args := []interface{}{id}
	if err := db.SelectOne(&dbInfo, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WrapError(sdk.ErrInfoNotFound, "LoadInfoByID>")
		}
		return nil, sdk.WrapError(err, "LoadInfoByID>")
	}
	info := sdk.Info(dbInfo)
	return &info, nil
}

// LoadInfos retrieves infos from database
func LoadInfos(db gorp.SqlExecutor) ([]sdk.Info, error) {
	res := []Info{}
	query := fmt.Sprintf(`select * from info`)
	if _, err := db.Select(&res, query); err != nil {
		return nil, sdk.WrapError(err, "LoadAllInfos> ")
	}

	infos := make([]sdk.Info, len(res))
	for i := range res {
		p := res[i]
		infos[i] = sdk.Info(p)
	}

	return infos, nil
}

// DeleteInfo removes info from database
func DeleteInfo(db gorp.SqlExecutor, ID int64) error {
	m := Info(sdk.Info{ID: ID})
	count, err := db.Delete(&m)
	if err != nil {
		return err
	}
	if count == 0 {
		return sdk.ErrNoInfo
	}
	return nil
}
