package broadcast

import (
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// InsertBroadcast insert a new worker broadcast in database
func InsertBroadcast(db gorp.SqlExecutor, bc *sdk.Broadcast) error {
	dbmsg := Broadcast(*bc)
	if err := db.Insert(&dbmsg); err != nil {
		return err
	}
	bc.ID = dbmsg.ID
	return nil
}

// UpdateBroadcast update a broadcast
func UpdateBroadcast(db gorp.SqlExecutor, bc sdk.Broadcast) error {
	bc.Updated = time.Now()
	dbmsg := Broadcast(bc)
	if _, err := db.Update(&dbmsg); err != nil {
		return err
	}
	return nil
}

// LoadBroadcastByID loads broadcast by id
func LoadBroadcastByID(db gorp.SqlExecutor, id int64) (*sdk.Broadcast, error) {
	dbBroadcast := Broadcast{}
	query := `select * from broadcast where id=$1`
	args := []interface{}{id}
	if err := db.SelectOne(&dbBroadcast, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WrapError(sdk.ErrBroadcastNotFound, "LoadBroadcastByID>")
		}
		return nil, sdk.WrapError(err, "LoadBroadcastByID>")
	}
	broadcast := sdk.Broadcast(dbBroadcast)
	return &broadcast, nil
}

// LoadBroadcasts retrieves infos from database
func LoadBroadcasts(db gorp.SqlExecutor) ([]sdk.Broadcast, error) {
	res := []Broadcast{}
	if _, err := db.Select(&res, `select * from broadcast`); err != nil {
		return nil, sdk.WrapError(err, "LoadAllBroadcasts> ")
	}

	infos := make([]sdk.Broadcast, len(res))
	for i := range res {
		p := res[i]
		infos[i] = sdk.Broadcast(p)
	}

	return infos, nil
}

// DeleteBroadcast removes broadcast from database
func DeleteBroadcast(db gorp.SqlExecutor, ID int64) error {
	m := Broadcast(sdk.Broadcast{ID: ID})
	count, err := db.Delete(&m)
	if err != nil {
		return err
	}
	if count == 0 {
		return sdk.ErrNoBroadcast
	}
	return nil
}
