package notification

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func storeCleaner() {
	for {
		time.Sleep(10 * time.Minute)
		db := database.DB()
		if db == nil {
			continue
		}
		t1 := time.Now().Add(-48 * time.Hour).Unix()
		query := `SELECT id FROM user_notification where creation_date <= $1`
		rows, err := db.Query(query, t1)
		if err != nil {
			log.Warning("notification.storeCleaner> unable to select notification to delete %s", err)
			continue
		}
		defer rows.Close()
		ids := []int{}
		for rows.Next() {
			var id int
			rows.Scan(&id)
			if id != 0 {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			Delete(db, ids)
		}
	}
}

//Insert save a notification in database
func Insert(db database.Querier, notif *sdk.Notif, notifType string) error {
	query := `
        INSERT INTO user_notification (type, content, creation_date)
        VALUES ($1, $2, $3)
        RETURNING id
    `
	//Perform a copy
	n := *notif
	//Erase pointer to avoid serialization
	n.ActionBuild = nil
	n.Build = nil

	content, err := json.Marshal(n)
	if err != nil {
		log.Warning("notification.Insert> Error marshalling notification :%s", err)
		return err
	}
	if err := db.QueryRow(query, notifType, content, time.Now().Unix()).Scan(&notif.ID); err != nil {
		log.Warning("notification.Insert> Error inserting notification :%s", err)
		return err
	}
	return nil
}

//Update updates a notification in database
func Update(db database.QueryExecuter, notif *sdk.Notif, status string) error {
	if notif.ID == 0 {
		return fmt.Errorf("notif is  is null")
	}
	//Perform a copy
	n := *notif
	//Erase pointer to avoid serialization
	n.ActionBuild = nil
	n.Build = nil

	content, err := json.Marshal(n)
	if err != nil {
		log.Warning("notification.Update> Error marshalling notification :%s", err)
		return err
	}

	query := `
        UPDATE user_notification SET 
        content = $1,
        status = $2
        WHERE id = $3
    `
	if _, err := db.Exec(query, content, status, n.ID); err != nil {
		log.Warning("notification.Insert> Error updating notification :%s", err)
		return err
	}

	return nil
}

//Delete remove a notification from database
func Delete(db database.QueryExecuter, ids []int) error {
	query := `
        DELETE FROM user_notification WHERE id = ANY($1::integer[])
    `

	var tparams []string
	for i := range ids {
		tparams = append(tparams, strconv.Itoa(ids[i]))
	}

	params := "{" + strings.Join(tparams, ",") + "}"

	if _, err := db.Exec(query, params); err != nil {
		log.Warning("notification.Delete> Error deleting notification :%s", err)
		return err
	}

	return nil
}
