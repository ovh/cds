package group

import (
	"database/sql"

	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/log"
)

// SharedInfraGroup is the name of the builtin group used to share infrastructure between projects
const SharedInfraGroup = "shared.infra"

// Initialize some stuff
func Initialize(db *sql.DB) error {
	//Load the famous sharedInfraGroup
	sharedInfraGroup, errLoad := LoadGroup(db, SharedInfraGroup)
	if errLoad != nil {
		log.Critical("group.init> Cannot load shared infra group: %s\n", errLoad)
		return errLoad
	}
	permission.SharedInfraGroupID = sharedInfraGroup.ID
	return nil
}

// CreateDefaultGlobalGroup creates a group 'public' where every user will be
func CreateDefaultGlobalGroup(db *sql.DB) error {

	query := `SELECT id FROM "group" where name = $1`
	var id int64
	err := db.QueryRow(query, SharedInfraGroup).Scan(&id)
	if err == sql.ErrNoRows {
		query = `INSERT INTO "group" (name) VALUES ($1)`
		_, err = db.Exec(query, SharedInfraGroup)
		if err != nil {
			return err
		}
	}

	return nil
}

// AddAdminInGlobalGroup insert into new admin into global group as group admin
func AddAdminInGlobalGroup(db *sql.DB, userID int64) error {

	query := `SELECT id FROM "group" where name = $1`
	var id int64
	err := db.QueryRow(query, SharedInfraGroup).Scan(&id)
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

// AddGlobalGroupTopipeline add global group access to given pipeline
func AddGlobalGroupToPipeline(tx *sql.Tx, pipID int64) error {
	query := `SELECT id FROM "group" where name = $1`
	var id int64
	err := tx.QueryRow(query, SharedInfraGroup).Scan(&id)
	if err != nil {
		return err
	}

	query = `INSERT INTO pipeline_group (pipeline_id, group_id, role) VALUES ($1, $2, $3)`
	_, err = tx.Exec(query, pipID, id, permission.PermissionReadExecute)
	if err != nil {
		return err
	}

	return nil
}
