package group

import (
	"database/sql"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// SharedInfraGroupName is the name of the builtin group used to share infrastructure between projects
const SharedInfraGroupName = "shared.infra"

// SharedInfraGroup is the group used to share infrastructure between projects
var SharedInfraGroup *sdk.Group

var defaultGroupID int64

// CreateDefaultGroup creates a group 'public' where every user will be
func CreateDefaultGroup(db *gorp.DbMap, groupName string) error {
	query := `SELECT id FROM "group" where name = $1`
	var id int64
	if err := db.QueryRow(query, groupName).Scan(&id); err == sql.ErrNoRows {
		log.Info("CreateDefaultGroup> create %s group in DB", groupName)
		query = `INSERT INTO "group" (name) VALUES ($1)`
		if _, err := db.Exec(query, groupName); err != nil {
			return err
		}
	} else {
		log.Info("CreateDefaultGroup> group %s already exist", groupName)
	}
	return nil
}

// AddAdminInGlobalGroup insert into new admin into global group as group admin
func AddAdminInGlobalGroup(db gorp.SqlExecutor, userID int64) error {
	query := `SELECT id FROM "group" where name = $1`
	var id int64
	if err := db.QueryRow(query, SharedInfraGroupName).Scan(&id); err != nil {
		return err
	}

	query = `INSERT INTO group_user (group_id, user_id, group_admin) VALUES ($1, $2, true)`
	if _, err := db.Exec(query, id, userID); err != nil {
		return err
	}
	return nil
}

// InitializeDefaultGroupName initializes sharedInfraGroup and Default Group
func InitializeDefaultGroupName(db *gorp.DbMap, defaultGroupName string) error {
	//Load the famous sharedInfraGroup
	var errlsg error
	SharedInfraGroup, errlsg = LoadGroup(db, SharedInfraGroupName)
	if errlsg != nil {
		return sdk.WrapError(errlsg, "group.InitializeDefaultGroupName> Cannot load shared infra group")
	}
	//Inject SharedInfraGroup.ID in permission package
	permission.SharedInfraGroupID = SharedInfraGroup.ID

	if defaultGroupName != "" {
		defaultGroup, errldg := LoadGroup(db, defaultGroupName)
		if errldg != nil {
			return sdk.WrapError(errldg, "group.InitializeDefaultGroupName> Cannot load %s group", defaultGroupName)
		}

		defaultGroupID = defaultGroup.ID
	}

	return nil
}

// IsDefaultGroupID returns true if groupID is the defaultGroupID
func IsDefaultGroupID(groupID int64) bool {
	return groupID == defaultGroupID
}
