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

// DefaultValues contains default user values for init DB
type DefaultValues struct {
	DefaultGroupName string
	SharedInfraToken string
}

//InitiliazeDB inits the database
func InitiliazeDB(defaultValues DefaultValues, DBFunc func() *gorp.DbMap) error {
	dbGorp := DBFunc()
	if err := action.CreateBuiltinArtifactActions(dbGorp); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot setup builtin Artifact actions")
	}

	if err := group.CreateDefaultGroup(dbGorp, group.SharedInfraGroupName); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot setup default %s group", group.SharedInfraGroupName)
	}

	if strings.TrimSpace(defaultValues.DefaultGroupName) != "" {
		if err := group.CreateDefaultGroup(dbGorp, defaultValues.DefaultGroupName); err != nil {
			return sdk.WrapError(err, "InitiliazeDB> Cannot setup default %s group")
		}
	}

	if err := action.CreateBuiltinActions(dbGorp); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot setup builtin actions")
	}

	if err := environment.CreateBuiltinEnvironments(dbGorp); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot setup builtin environments")
	}

	if err := group.InitializeDefaultGroupName(dbGorp, defaultValues.DefaultGroupName); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot InitializeDefaultGroupName")
	}

	if err := token.Initialize(dbGorp, defaultValues.SharedInfraToken); err != nil {
		return sdk.WrapError(err, "InitiliazeDB> Cannot InitializeDefaultGroupName")
	}

	return nil
}
