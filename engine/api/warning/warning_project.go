package warning

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/mitchellh/mapstructure"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func computeWithProjectEvent(db gorp.SqlExecutor, e sdk.Event) error {
	switch e.EventType {

	case "sdk.EventProjectVariableAdd":
		var varEvent sdk.EventProjectVariableAdd
		if err := mapstructure.Decode(e.Payload, &varEvent); err != nil {
			return sdk.WrapError(err, "computeWithProjectEvent> Unable to decode EventProjectVariableAdd")
		}
		return manageProjectAddVariableEvent(db, e.ProjectKey, fmt.Sprintf("cds.proj.%s", varEvent.Variable.Name))
	case "sdk.EventProjectVariableUpdate":
		var varEvent sdk.EventProjectVariableUpdate
		if err := mapstructure.Decode(e.Payload, &varEvent); err != nil {
			return sdk.WrapError(err, "computeWithProjectEvent> Unable to decode EventProjectVariableUpdate")
		}
		return manageProjectUpdateVariableEvent(db, e.ProjectKey, varEvent.NewVariable, varEvent.OldVariable)
	case "sdk.EventProjectVariableDelete":
		var varEvent sdk.EventProjectVariableDelete
		if err := mapstructure.Decode(e.Payload, &varEvent); err != nil {
			return sdk.WrapError(err, "computeWithProjectEvent> Unable to decode EventProjectVariableDelete")
		}
		return manageProjectDeleteVariableEvent(db, e.ProjectKey, fmt.Sprintf("cds.proj.%s", varEvent.Variable.Name))
	case "sdk.EventProjectPermissionAdd":
		var permEvent sdk.EventProjectPermissionAdd
		if err := mapstructure.Decode(e.Payload, &permEvent); err != nil {
			return sdk.WrapError(err, "computeWithProjectEvent> Unable to decode EventProjectPermissionAdd")
		}
		return manageProjectAddPermission(db, e.ProjectKey, permEvent.Permission)
	case "sdk.EventProjectPermissionDelete":
		// Check if permission is used on workflow
		var permEvent sdk.EventProjectPermissionDelete
		if err := mapstructure.Decode(e.Payload, &permEvent); err != nil {
			return sdk.WrapError(err, "computeWithProjectEvent> Unable to decode EventProjectPermissionDelete")
		}
		return manageProjectDeletePermission(db, e.ProjectKey, permEvent.Permission)
	case "sdk.EventProjectKeyAdd":
		var keyAddEvent sdk.EventProjectKeyAdd
		if err := mapstructure.Decode(e.Payload, &keyAddEvent); err != nil {
			return sdk.WrapError(err, "computeWithProjectEvent> Unable to decode EventProjectKeyAdd")
		}
		return manageProjectAddKey(db, e.ProjectKey, keyAddEvent.Key)

	case "sdk.EventProjectKeyDelete":
		var keyDeleteEvent sdk.EventProjectKeyDelete
		if err := mapstructure.Decode(e.Payload, &keyDeleteEvent); err != nil {
			return sdk.WrapError(err, "computeWithProjectEvent> Unable to decode EventProjectKeyDelete")
		}
		return manageProjectDeleteKey(db, e.ProjectKey, keyDeleteEvent.Key)
	case "sdk.EventProjectVCSServerDelete":
		// Check if vcs is used

	default:
		log.Debug("Event %s ignored", e.EventType)
		return nil
	}
	return nil
}

func manageProjectAddKey(db gorp.SqlExecutor, projectKey string, key sdk.ProjectKey) error {
	if err := removeProjectWarning(db, MissingProjectKey, key.Name, projectKey); err != nil {
		return sdk.WrapError(err, "manageProjectAddKey> Unable to remove warning %s for key %s on project %s", MissingProjectKey, key.Name, projectKey)
	}
	appsName, pipsName := keyIsUsed(db, projectKey, key.Name)
	if len(appsName) == 0 && len(pipsName) == 0 {
		w := sdk.WarningV2{
			Key:     projectKey,
			Element: key.Name,
			Created: time.Now(),
			Type:    UnusedProjectKey,
			MessageParams: map[string]string{
				"KeyName":    key.Name,
				"ProjectKey": projectKey,
			},
		}
		if err := Insert(db, w); err != nil {
			return sdk.WrapError(err, "manageProjectAddKey> Unable to insert warning")
		}
	}
	return nil
}

func manageProjectDeleteKey(db gorp.SqlExecutor, projectKey string, key sdk.ProjectKey) error {
	if err := removeProjectWarning(db, UnusedProjectKey, key.Name, projectKey); err != nil {
		return sdk.WrapError(err, "manageProjectDeleteKey> Unable to remove warning %s for key %s on project %s", UnusedProjectKey, key.Name, projectKey)
	}
	appsName, pipsName := keyIsUsed(db, projectKey, key.Name)
	if len(appsName) > 0 || len(pipsName) > 0 {
		w := sdk.WarningV2{
			Key:     projectKey,
			Element: key.Name,
			Created: time.Now(),
			Type:    MissingProjectKey,
			MessageParams: map[string]string{
				"KeyName":    key.Name,
				"ProjectKey": projectKey,
				"AppsName":   strings.Join(appsName, ", "),
				"PipsName":   strings.Join(pipsName, ", "),
			},
		}
		if err := Insert(db, w); err != nil {
			return sdk.WrapError(err, "manageProjectAddKey> Unable to insert warning")
		}
	}
	return nil
}

func manageProjectAddPermission(db gorp.SqlExecutor, key string, gp sdk.GroupPermission) error {
	if err := removeProjectWarning(db, MissingProjectPermissionEnv, gp.Group.Name, key); err != nil {
		return sdk.WrapError(err, "manageProjectAddPermission> Unable to remove warning %s for group %s on project %s", MissingProjectPermissionEnv, gp.Group.Name, key)
	}
	if err := removeProjectWarning(db, MissingProjectPermissionWorkflow, gp.Group.Name, key); err != nil {
		return sdk.WrapError(err, "manageProjectAddPermission> Unable to remove warning %s for group %s on project %s", MissingProjectPermissionWorkflow, gp.Group.Name, key)
	}
	return nil
}

func manageProjectDeletePermission(db gorp.SqlExecutor, key string, gp sdk.GroupPermission) error {
	// Check in ENV
	envs, err := group.EnvironmentsByGroupID(db, key, gp.Group.ID)
	if err != nil {
		return sdk.WrapError(err, "manageProjectDeletePermission> Unable to list environments")
	}
	if len(envs) > 0 {
		w := sdk.WarningV2{
			Key:     key,
			Element: gp.Group.Name,
			Created: time.Now(),
			Type:    MissingProjectPermissionEnv,
			MessageParams: map[string]string{
				"GroupName":  gp.Group.Name,
				"ProjectKey": key,
				"EnvName":    strings.Join(envs, ","),
			},
		}
		if err := Insert(db, w); err != nil {
			return sdk.WrapError(err, "manageAddVariableEvent> Unable to Insert environment warning")
		}
	}

	// Check in workflow
	workflows, err := workflow.ByGroupID(db, key, gp.Group.ID)
	if err != nil {
		return sdk.WrapError(err, "manageProjectDeletePermission> Unable to list workflows")
	}
	if len(envs) > 0 {
		w := sdk.WarningV2{
			Key:     key,
			Element: gp.Group.Name,
			Created: time.Now(),
			Type:    MissingProjectPermissionWorkflow,
			MessageParams: map[string]string{
				"GroupName":    gp.Group.Name,
				"ProjectKey":   key,
				"WorkflowName": strings.Join(workflows, ","),
			},
		}
		if err := Insert(db, w); err != nil {
			return sdk.WrapError(err, "manageProjectDeletePermission> Unable to Insert workflow warning")
		}
	}
	return nil
}

func manageProjectAddVariableEvent(db gorp.SqlExecutor, key string, varName string) error {
	if err := removeProjectWarning(db, MissingProjectVariable, varName, key); err != nil {
		return sdk.WrapError(err, "manageAddVariableEvent> Unable to remove warning")
	}

	envs, apps, pips := variableIsUsed(db, key, varName)
	if len(envs) == 0 && len(apps) == 0 && len(pips) == 0 {
		w := sdk.WarningV2{
			Key:     key,
			Element: varName,
			Created: time.Now(),
			Type:    UnusedProjectVariable,
			MessageParams: map[string]string{
				"VarName":    varName,
				"ProjectKey": key,
			},
		}
		if err := Insert(db, w); err != nil {
			return sdk.WrapError(err, "manageAddVariableEvent> Unable to Insert warning")
		}
	}
	return nil
}

func manageProjectUpdateVariableEvent(db gorp.SqlExecutor, key string, newVar sdk.Variable, oldVar sdk.Variable) error {
	if newVar.Name == oldVar.Name {
		return nil
	}

	if err := removeProjectWarning(db, UnusedProjectVariable, fmt.Sprintf("cds.proj.%s", oldVar.Name), key); err != nil {
		log.Warning("manageUpdateVariableEvent> Unable to remove oldvar warning: %v", err)
	}

	if err := removeProjectWarning(db, MissingProjectVariable, fmt.Sprintf("cds.proj.%s", newVar.Name), key); err != nil {
		log.Warning("manageUpdateVariableEvent> Unable to remove newvar warning: %v", err)
	}

	projVarName := fmt.Sprintf("cds.proj.%s", newVar.Name)
	envs, apps, pips := variableIsUsed(db, key, projVarName)
	if len(envs) == 0 && len(apps) == 0 && len(pips) == 0 {
		w := sdk.WarningV2{
			Key:     key,
			Element: projVarName,
			Created: time.Now(),
			Type:    UnusedProjectVariable,
			MessageParams: map[string]string{
				"VarName":    projVarName,
				"ProjectKey": key,
			},
		}
		if err := Insert(db, w); err != nil {
			return sdk.WrapError(err, "manageUpdateVariableEvent> Unable to Insert warning")
		}
	}
	return nil
}

func manageProjectDeleteVariableEvent(db gorp.SqlExecutor, key string, varName string) error {
	if err := removeProjectWarning(db, UnusedProjectVariable, varName, key); err != nil {
		log.Warning("manageDeleteVariableEvent> Unable to remove warning: %v", err)
	}
	envs, apps, pips := variableIsUsed(db, key, varName)
	if len(envs) == 0 && len(apps) == 0 && len(pips) == 0 {
		return nil
	}

	w := sdk.WarningV2{
		Key:     key,
		Element: varName,
		Created: time.Now(),
		Type:    MissingProjectVariable,
		MessageParams: map[string]string{
			"VarName":    varName,
			"ProjectKey": key,
			"EnvsName":   strings.Join(envs, ", "),
			"AppsName":   strings.Join(apps, ", "),
			"PipsName":   strings.Join(pips, ", "),
		},
	}
	if err := Insert(db, w); err != nil {
		return sdk.WrapError(err, "manageDeleteVariableEvent> Unable to Insert warning")
	}

	return nil
}
