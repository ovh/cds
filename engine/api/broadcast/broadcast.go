package broadcast

import (
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// Insert insert a new worker broadcast in database
func Insert(db gorp.SqlExecutor, bc *sdk.Broadcast) error {
	dbmsg := broadcast(*bc)
	if err := db.Insert(&dbmsg); err != nil {
		return err
	}
	bc.ID = dbmsg.ID
	return nil
}

// Update update a broadcast
func Update(db gorp.SqlExecutor, bc *sdk.Broadcast) error {
	bc.Updated = time.Now()
	dbmsg := broadcast(*bc)
	if _, err := db.Update(&dbmsg); err != nil {
		return err
	}
	return nil
}

// LoadByID loads broadcast by id
func LoadByID(db gorp.SqlExecutor, id int64) (*sdk.Broadcast, error) {
	dbBroadcast := broadcast{}
	query := `select * from broadcast where id=$1`
	if err := db.SelectOne(&dbBroadcast, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WrapError(sdk.ErrBroadcastNotFound, "LoadByID>")
		}
		return nil, sdk.WrapError(err, "LoadByID>")
	}
	broadcast := sdk.Broadcast(dbBroadcast)
	if broadcast.ProjectID != nil && *broadcast.ProjectID > 0 {
		pkey, errP := db.SelectStr("select projectkey from project where id = $1", broadcast.ProjectID)
		if errP != nil {
			return nil, sdk.WrapError(errP, "LoadByID>")
		}
		broadcast.ProjectKey = pkey
	}
	return &broadcast, nil
}

// LoadAll retrieves broadcasts from database
func LoadAll(db gorp.SqlExecutor) ([]sdk.Broadcast, error) {
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
				return nil, sdk.WrapError(errP, "LoadAll>")
			}
			broadcasts[i].ProjectKey = pkey
		}
	}

	return broadcasts, nil
}

// Delete removes broadcast from database
func Delete(db gorp.SqlExecutor, ID int64) error {
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
