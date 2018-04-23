package broadcast

import (
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// InsertBroadcast insert a new worker broadcast in database
func InsertBroadcast(db gorp.SqlExecutor, bc *sdk.Broadcast) error {
	dbmsg := broadcast(*bc)
	if err := db.Insert(&dbmsg); err != nil {
		return err
	}
	bc.ID = dbmsg.ID
	return nil
}

// UpdateBroadcast update a broadcast
func UpdateBroadcast(db gorp.SqlExecutor, bc *sdk.Broadcast) error {
	bc.Updated = time.Now()
	dbmsg := broadcast(*bc)
	if _, err := db.Update(&dbmsg); err != nil {
		return err
	}
	return nil
}

// LoadBroadcastByID loads broadcast by id
func LoadBroadcastByID(db gorp.SqlExecutor, id int64) (*sdk.Broadcast, error) {
	dbBroadcast := broadcast{}
	query := `select * from broadcast where id=$1`
	if err := db.SelectOne(&dbBroadcast, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WrapError(sdk.ErrBroadcastNotFound, "LoadBroadcastByID>")
		}
		return nil, sdk.WrapError(err, "LoadBroadcastByID>")
	}
	broadcast := sdk.Broadcast(dbBroadcast)
	if broadcast.ProjectID != nil && *broadcast.ProjectID > 0 {
		pkey, errP := db.SelectStr("select projectkey from project where id = $1", broadcast.ProjectID)
		if errP != nil {
			return nil, sdk.WrapError(errP, "LoadBroadcastByID>")
		}
		broadcast.ProjectKey = pkey
	}
	return &broadcast, nil
}

// LoadBroadcasts retrieves broadcasts from database
func LoadBroadcasts(db gorp.SqlExecutor) ([]sdk.Broadcast, error) {
	res := []broadcast{}
	if _, err := db.Select(&res, `select * from broadcast`); err != nil {
		return nil, sdk.WrapError(err, "LoadAllBroadcasts> ")
	}

	broadcasts := make([]sdk.Broadcast, len(res))
	for i := range res {
		p := res[i]
		broadcasts[i] = sdk.Broadcast(p)

		if broadcasts[i].ProjectID != nil && *broadcasts[i].ProjectID > 0 {
			pkey, errP := db.SelectStr("select projectkey from project where id = $1", broadcasts[i].ProjectID)
			if errP != nil {
				return nil, sdk.WrapError(errP, "LoadBroadcasts>")
			}
			broadcasts[i].ProjectKey = pkey
		}
	}

	return broadcasts, nil
}

// DeleteBroadcast removes broadcast from database
func DeleteBroadcast(db gorp.SqlExecutor, ID int64) error {
	m := broadcast(sdk.Broadcast{ID: ID})
	count, err := db.Delete(&m)
	if err != nil {
		return err
	}
	if count == 0 {
		return sdk.ErrNoBroadcast
	}
	return nil
}
