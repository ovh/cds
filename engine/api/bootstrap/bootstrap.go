package bootstrap

import (
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//InitiliazeDB inits the database
func InitiliazeDB(defaultGroupName string, sharedInfraToken string, DBFunc func() *gorp.DbMap) error {
	dbGorp := DBFunc()
	if err := action.CreateBuiltinArtifactActions(dbGorp); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot setup builtin Artifact actions")
	}

	sharedInfraGroupID, errg := group.CreateDefaultGroup(dbGorp, group.SharedInfraGroupName)
	if errg != nil {
		return sdk.WrapError(errg, "InitiliazeDB> Cannot setup default %s group", group.SharedInfraGroupName)
	}

	// check if shared infra has a token. If not, take token from config file.
	// if there is no token in config file -> it's an error
	nbToken, errt := worker.CountToken(dbGorp, sharedInfraGroupID)
	if errt != nil {
		return sdk.WrapError(errt, "InitiliazeDB> Cannot count token on default %s group", group.SharedInfraGroupName)
	}

	if nbToken == 0 {
		if len(sharedInfraToken) == 0 {
			return sdk.WrapError(errg, "Invalid Configuration. You have to set token for shared infra group in your configuration")
		}
		log.Info("InitiliazeDB> create token for %s group", group.SharedInfraGroupName)
		if err := worker.InsertToken(dbGorp, sharedInfraGroupID, sharedInfraToken, sdk.Persistent); err != nil {
			return sdk.WrapError(err, "InitiliazeDB> cannot insert new token for %s", group.SharedInfraGroupName)
		}
	}

	if strings.TrimSpace(defaultGroupName) != "" {
		if _, err := group.CreateDefaultGroup(dbGorp, defaultGroupName); err != nil {
			return sdk.WrapError(err, "InitiliazeDB> Cannot setup default %s group")
		}
	}

	if err := action.CreateBuiltinActions(dbGorp); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot setup builtin actions")
	}

	if err := environment.CreateBuiltinEnvironments(dbGorp); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot setup builtin environments")
	}

	if err := group.InitializeDefaultGroupName(dbGorp, defaultGroupName); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot InitializeDefaultGroupName")
	}

	return nil
}
