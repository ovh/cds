package bootstrap

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk/log"
)

//InitiliazeDB inits the database
func InitiliazeDB(DBFunc func() *gorp.DbMap) error {
	dbGorp := DBFunc()
	if err := action.CreateBuiltinArtifactActions(dbGorp); err != nil {
		log.Error("Cannot setup builtin Artifact actions: %s\n", err)
		return err
	}

	if err := group.CreateDefaultGlobalGroup(dbGorp); err != nil {
		log.Error("Cannot setup default global group: %s\n", err)
		return err
	}

	if err := action.CreateBuiltinActions(dbGorp); err != nil {
		log.Error("Cannot setup builtin actions: %s\n", err)
		return err
	}

	if err := environment.CreateBuiltinEnvironments(dbGorp); err != nil {
		log.Error("Cannot setup builtin environments: %s\n", err)
		return err
	}

	if err := group.Initialize(dbGorp, ""); err != nil {
		log.Error("Cannot setup shared infra group: %s\n", err)
		return err
	}

	return nil
}
