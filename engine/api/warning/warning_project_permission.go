package warning

import (
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type missingProjectPermissionEnvWarning struct {
	commonWarn
}

func (warn missingProjectPermissionEnvWarning) events() []string {
	return []string{
		fmt.Sprintf("%T", sdk.EventProjectPermissionAdd{}),
		fmt.Sprintf("%T", sdk.EventProjectPermissionDelete{}),
	}
}

func (warn missingProjectPermissionEnvWarning) name() string {
	return sdk.WarningMissingProjectPermissionEnv
}

func (warn missingProjectPermissionEnvWarning) compute(db gorp.SqlExecutor, e sdk.Event) error {
	switch e.EventType {
	case fmt.Sprintf("%T", sdk.EventProjectPermissionAdd{}):
		payload, err := e.ToEventProjectPermissionAdd()
		if err != nil {
			return sdk.WrapError(err, "missingProjectPermissionEnvWarning.compute> Unable to get payload from EventProjectPermissionAdd")
		}
		if err := removeProjectWarning(db, warn.name(), payload.Permission.Group.Name, e.ProjectKey); err != nil {
			log.Warning("missingProjectPermissionEnvWarning.compute> Unable to remove warning from EventProjectPermissionAdd")
		}
	case fmt.Sprintf("%T", sdk.EventProjectPermissionDelete{}):
		payload, err := e.ToEventProjectPermissionDelete()
		if err != nil {
			return sdk.WrapError(err, "missingProjectPermissionEnvWarning.compute> Unable to get payload from ToEventProjectPermissionDelete")
		}
		// Check in ENV
		envs, err := group.EnvironmentsByGroupID(db, e.ProjectKey, payload.Permission.Group.ID)
		if err != nil {
			return sdk.WrapError(err, "missingProjectPermissionEnvWarning.compute> Unable to list environments")
		}
		for _, env := range envs {
			w := sdk.WarningV2{
				Key:     e.ProjectKey,
				EnvName: env,
				Element: payload.Permission.Group.Name,
				Created: time.Now(),
				Type:    warn.name(),
				MessageParams: map[string]string{
					"GroupName":       payload.Permission.Group.Name,
					"ProjectKey":      e.ProjectKey,
					"EnvironmentName": env,
				},
			}
			if err := Insert(db, w); err != nil {
				return sdk.WrapError(err, "missingProjectPermissionEnvWarning.compute> Unable to Insert environment warning %s", warn.name())
			}
		}
	}
	return nil
}

type missingProjectPermissionWorkflowWarning struct {
	commonWarn
}

func (warn missingProjectPermissionWorkflowWarning) events() []string {
	return []string{
		fmt.Sprintf("%T", sdk.EventProjectPermissionAdd{}),
		fmt.Sprintf("%T", sdk.EventProjectPermissionDelete{}),
	}
}

func (warn missingProjectPermissionWorkflowWarning) name() string {
	return sdk.WarningMissingProjectPermissionWorkflow
}

func (warn missingProjectPermissionWorkflowWarning) compute(db gorp.SqlExecutor, e sdk.Event) error {
	switch e.EventType {
	case fmt.Sprintf("%T", sdk.EventProjectPermissionAdd{}):
		payload, err := e.ToEventProjectPermissionAdd()
		if err != nil {
			return sdk.WrapError(err, "missingProjectPermissionWorkflowWarning.compute> Unable to get payload from EventProjectPermissionAdd")
		}
		if err := removeProjectWarning(db, warn.name(), payload.Permission.Group.Name, e.ProjectKey); err != nil {
			log.Warning("missingProjectPermissionWorkflowWarning.compute> Unable to remove warning from EventProjectPermissionAdd")
		}
	case fmt.Sprintf("%T", sdk.EventProjectPermissionDelete{}):
		payload, err := e.ToEventProjectPermissionDelete()
		if err != nil {
			return sdk.WrapError(err, "missingProjectPermissionWorkflowWarning.compute> Unable to get payload from ToEventProjectPermissionDelete")
		}
		workflows, err := workflow.ByGroupID(db, e.ProjectKey, payload.Permission.Group.ID)
		if err != nil {
			return sdk.WrapError(err, "missingProjectPermissionWorkflowWarning.compute> Unable to list worklflows")
		}
		for _, w := range workflows {
			w := sdk.WarningV2{
				Key:          e.ProjectKey,
				WorkflowName: w,
				Element:      payload.Permission.Group.Name,
				Created:      time.Now(),
				Type:         warn.name(),
				MessageParams: map[string]string{
					"GroupName":    payload.Permission.Group.Name,
					"ProjectKey":   e.ProjectKey,
					"WorkflowName": w,
				},
			}
			if err := Insert(db, w); err != nil {
				return sdk.WrapError(err, "missingProjectPermissionWorkflowWarning.compute> Unable to Insert warning %s", warn.name())
			}
		}
	}
	return nil
}
