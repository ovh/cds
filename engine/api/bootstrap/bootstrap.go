package bootstrap

import (
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/sdk"
)

//InitiliazeDB inits the database
func InitiliazeDB(defaultValues sdk.DefaultValues, DBFunc func() *gorp.DbMap) error {
	dbGorp := DBFunc()

	if err := group.CreateDefaultGroup(dbGorp, sdk.SharedInfraGroupName); err != nil {
		return sdk.WrapError(err, "Cannot setup default %s group", sdk.SharedInfraGroupName)
	}

	if strings.TrimSpace(defaultValues.DefaultGroupName) != "" {
		if err := group.CreateDefaultGroup(dbGorp, defaultValues.DefaultGroupName); err != nil {
			return sdk.WrapError(err, "Cannot setup default %s group", defaultValues.DefaultGroupName)
		}
	}

	if err := group.InitializeDefaultGroupName(dbGorp, defaultValues.DefaultGroupName); err != nil {
		return sdk.WrapError(err, "Cannot InitializeDefaultGroupName")
	}

	if err := action.CreateBuiltinActions(dbGorp); err != nil {
		return sdk.WrapError(err, "Cannot setup builtin actions")
	}

	if err := environment.CreateBuiltinEnvironments(dbGorp); err != nil {
		return sdk.WrapError(err, "Cannot setup builtin environments")
	}

	if err := workflow.CreateBuiltinWorkflowHookModels(dbGorp); err != nil {
		return fmt.Errorf("cannot setup builtin workflow hook models: %v", err)
	}

	if err := workflow.CreateBuiltinWorkflowOutgoingHookModels(dbGorp); err != nil {
		return fmt.Errorf("cannot setup builtin workflow outgoing hook models: %v", err)
	}

	if err := integration.CreateBuiltinModels(dbGorp); err != nil {
		return fmt.Errorf("cannot setup integrations: %v", err)
	}

	return nil
}
