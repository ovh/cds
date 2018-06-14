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

// MarkAsRead mark the broadcast as read for an user
func MarkAsRead(db gorp.SqlExecutor, broadcastID, userID int64) error {
	brr := broadcastRead{
		BroadcastID: broadcastID,
		UserID:      userID,
	}
	err := db.Insert(&brr)

	return sdk.WrapError(err, "MarkAsRead>")
}

// LoadByID loads broadcast by id
func LoadByID(db gorp.SqlExecutor, id int64, u *sdk.User) (*sdk.Broadcast, error) {
	var projectKey sql.NullString
	query := `
		SELECT
			broadcast.id,
			broadcast.title,
			broadcast.content,
			broadcast.level,
			broadcast.created,
			broadcast.updated,
			broadcast.archived,
			broadcast.project_id,
			project.projectkey,
			(broadcast_read.broadcast_id IS NOT NULL)::boolean AS read
			FROM broadcast
				LEFT JOIN broadcast_read ON broadcast.id = broadcast_read.broadcast_id AND broadcast_read.user_id = $1
				LEFT JOIN project ON broadcast.project_id = project.id
		WHERE broadcast.id = $2
	`
	var broadcast sdk.Broadcast
	err := db.QueryRow(query, u.ID, id).Scan(&broadcast.ID, &broadcast.Title, &broadcast.Content, &broadcast.Level,
		&broadcast.Created, &broadcast.Updated, &broadcast.Archived, &broadcast.ProjectID, &projectKey, &broadcast.Read)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sdk.WrapError(sdk.ErrBroadcastNotFound, "LoadByID>")
		}
		return nil, sdk.WrapError(err, "LoadByID>")
	}

	if projectKey.Valid {
		broadcast.ProjectKey = projectKey.String
	}

	return &broadcast, nil
}

// LoadAll retrieves broadcasts from database
func LoadAll(db gorp.SqlExecutor, u *sdk.User) ([]sdk.Broadcast, error) {
	query := `
	SELECT
		broadcast.id,
		broadcast.title,
		broadcast.content,
		broadcast.level,
		broadcast.created,
		broadcast.updated,
		broadcast.archived,
		broadcast.project_id,
		project.projectkey,
		(broadcast_read.broadcast_id IS NOT NULL)::boolean AS read
		FROM broadcast
			LEFT JOIN broadcast_read ON broadcast.id = broadcast_read.broadcast_id AND broadcast_read.user_id = $1
			LEFT JOIN project ON broadcast.project_id = project.id
	ORDER BY updated DESC
	`

	rows, err := db.Query(query, u.ID)
	if err != nil {
		return nil, sdk.WrapError(err, "LoadAllBroadcasts> Cannot query")
	}

	broadcasts := []sdk.Broadcast{}
	for rows.Next() {
		var projectKey sql.NullString
		var broadcast sdk.Broadcast
		err := rows.Scan(&broadcast.ID, &broadcast.Title, &broadcast.Content, &broadcast.Level,
			&broadcast.Created, &broadcast.Updated, &broadcast.Archived, &broadcast.ProjectID, &projectKey, &broadcast.Read)

		if err != nil {
			return nil, sdk.WrapError(err, "LoadAllBroadcasts> cannot scan row")
		}
		if projectKey.Valid {
			broadcast.ProjectKey = projectKey.String
		}
		broadcasts = append(broadcasts, broadcast)
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

func deleteOldBroadcasts(db *gorp.DbMap) error {
	query := `DELETE
		FROM broadcast
		WHERE archived = true
		AND age(updated) > interval '30 days'
	`
	_, err := db.Exec(query)
	return err
}
