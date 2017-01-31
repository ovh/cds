package bootstrap

import (
	"database/sql"

	"github.com/ovh/cds/engine/api/artifact"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
)

//InitiliazeDB inits the database
func InitiliazeDB(db *sql.DB) error {
	dbGorp := database.DBMap(db)
	if err := artifact.CreateBuiltinArtifactActions(dbGorp); err != nil {
		log.Critical("Cannot setup builtin Artifact actions: %s\n", err)
		return err
	}

	if err := group.CreateDefaultGlobalGroup(dbGorp); err != nil {
		log.Critical("Cannot setup default global group: %s\n", err)
		return err
	}

	if err := worker.CreateBuiltinActions(dbGorp); err != nil {
		log.Critical("Cannot setup builtin actions: %s\n", err)
		return err
	}

	if err := worker.CreateBuiltinEnvironments(dbGorp); err != nil {
		log.Critical("Cannot setup builtin environments: %s\n", err)
		return err
	}
	return nil
}