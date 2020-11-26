package bootstrap

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

//InitiliazeDB inits the database
func InitiliazeDB(ctx context.Context, defaultValues sdk.DefaultValues, DBFunc func() *gorp.DbMap) error {
	tx, err := DBFunc().Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	if err := group.CreateDefaultGroup(tx, sdk.SharedInfraGroupName); err != nil {
		return sdk.WrapError(err, "Cannot setup default %s group", sdk.SharedInfraGroupName)
	}

	if strings.TrimSpace(defaultValues.DefaultGroupName) != "" {
		if err := group.CreateDefaultGroup(tx, defaultValues.DefaultGroupName); err != nil {
			return sdk.WrapError(err, "Cannot setup default %s group", defaultValues.DefaultGroupName)
		}
	}

	if err := group.InitializeDefaultGroupName(tx, defaultValues.DefaultGroupName); err != nil {
		return sdk.WrapError(err, "Cannot InitializeDefaultGroupName")
	}

	if err := action.CreateBuiltinActions(tx); err != nil {
		return sdk.WrapError(err, "Cannot setup builtin actions")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	if err := workflow.CreateBuiltinWorkflowHookModels(DBFunc()); err != nil {
		return fmt.Errorf("cannot setup builtin workflow hook models: %v", err)
	}

	if err := workflow.CreateBuiltinWorkflowOutgoingHookModels(DBFunc()); err != nil {
		return fmt.Errorf("cannot setup builtin workflow outgoing hook models: %v", err)
	}

	if err := integration.CreateBuiltinModels(DBFunc()); err != nil {
		return fmt.Errorf("cannot setup integrations: %v", err)
	}

	return nil
}
