package bootstrap

import (
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/sdk"
)

//InitiliazeDB inits the database
func InitiliazeDB(defaultValues sdk.DefaultValues, DBFunc func() *gorp.DbMap) error {
	dbGorp := DBFunc()

	if err := group.CreateDefaultGroup(dbGorp, sdk.SharedInfraGroupName); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot setup default %s group", sdk.SharedInfraGroupName)
	}

	if strings.TrimSpace(defaultValues.DefaultGroupName) != "" {
		if err := group.CreateDefaultGroup(dbGorp, defaultValues.DefaultGroupName); err != nil {
			return sdk.WrapError(err, "InitiliazeDB> Cannot setup default %s group")
		}
	}

	if err := group.InitializeDefaultGroupName(dbGorp, defaultValues.DefaultGroupName); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot InitializeDefaultGroupName")
	}

	if err := token.Initialize(dbGorp, defaultValues.SharedInfraToken); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot InitializeDefaultGroupName")
	}

	if err := action.CreateBuiltinArtifactActions(dbGorp); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot setup builtin Artifact actions")
	}

	if err := action.CreateBuiltinActions(dbGorp); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot setup builtin actions")
	}

	if err := environment.CreateBuiltinEnvironments(dbGorp); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot setup builtin environments")
	}

	return nil
}
