package group

import (
	"database/sql"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// SharedInfraGroupName is the name of the builtin group used to share infrastructure between projects
const SharedInfraGroupName = "shared.infra"

// SharedInfraGroup is the group used to share infrastructure between projects
var SharedInfraGroup *sdk.Group

var defaultGroupID int64

// CreateDefaultGlobalGroup creates a group 'public' where every user will be
func CreateDefaultGlobalGroup(db *gorp.DbMap) error {

	query := `SELECT id FROM "group" where name = $1`
	var id int64
	err := db.QueryRow(query, SharedInfraGroupName).Scan(&id)
	if err == sql.ErrNoRows {
		query = `INSERT INTO "group" (name) VALUES ($1)`
		_, err = db.Exec(query, SharedInfraGroupName)
		if err != nil {
			return err
		}
	}

	return nil
}

// AddAdminInGlobalGroup insert into new admin into global group as group admin
func AddAdminInGlobalGroup(db gorp.SqlExecutor, userID int64) error {

	query := `SELECT id FROM "group" where name = $1`
	var id int64
	err := db.QueryRow(query, SharedInfraGroupName).Scan(&id)
	if err != nil {
		return err
	}

	query = `INSERT INTO group_user (group_id, user_id, group_admin) VALUES ($1, $2, true)`
	_, err = db.Exec(query, id, userID)
	if err != nil {
		return err
	}

	return nil
}

// Initialize initializes sharedInfraGroup and Default Group
func Initialize(db *gorp.DbMap, defaultGroupName string) error {
	//Load the famous sharedInfraGroup
	var errlg error
	SharedInfraGroup, errlg = LoadGroup(db, SharedInfraGroupName)
	if errlg != nil {
		log.Critical("group.Initialize> Cannot load shared infra group: %s\n", errlg)
		return errlg
	}

	if defaultGroupName != "" {
		g, errld := LoadGroup(db, defaultGroupName)
		if errld != nil {
			log.Critical("group.Initialize> Cannot load default group '%s': %s\n", defaultGroupName, errld)
			return errld
		}
		defaultGroupID = g.ID
	}
	return nil
}
