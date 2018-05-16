package warning

import (
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/mitchellh/mapstructure"

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
	case "sdk.EventProjectPermissionDelete":
		// Check if permission is used on workflow

	case "sdk.EventProjectKeyAdd":
		// Check if key is used
		// Check if there is a warning on it

	case "sdk.EventProjectKeyDelete":
		// Check if key is used
		// Check if there is a warning on it

	case "sdk.EventProjectVCSServerDelete":
		// Check if vcs is used

	default:
		log.Debug("Event %s ignored", e.EventType)
		return nil
	}
	return nil
}

func manageProjectAddVariableEvent(db gorp.SqlExecutor, key string, varName string) error {
	if err := removeWarning(db, MissingProjectVariable, varName); err != nil {
		return sdk.WrapError(err, "manageAddVariableEvent> Unable to remove warning")
	}

	used := variableIsUsed(db, key, varName)
	if !used {
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
		if err := insert(db, w); err != nil {
			return sdk.WrapError(err, "manageAddVariableEvent> Unable to insert warning")
		}
	}
	return nil
}

func manageProjectUpdateVariableEvent(db gorp.SqlExecutor, key string, newVar sdk.Variable, oldVar sdk.Variable) error {
	if newVar.Name == oldVar.Name {
		return nil
	}

	if err := removeWarning(db, UnusedProjectVariable, fmt.Sprintf("cds.proj.%s", oldVar.Name)); err != nil {
		log.Warning("manageUpdateVariableEvent> Unable to remove oldvar warning: %v", err)
	}

	if err := removeWarning(db, MissingProjectVariable, fmt.Sprintf("cds.proj.%s", newVar.Name)); err != nil {
		log.Warning("manageUpdateVariableEvent> Unable to remove newvar warning: %v", err)
	}

	projVarName := fmt.Sprintf("cds.proj.%s", newVar.Name)
	used := variableIsUsed(db, key, projVarName)
	if !used {
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
		if err := insert(db, w); err != nil {
			return sdk.WrapError(err, "manageUpdateVariableEvent> Unable to insert warning")
		}
	}
	return nil
}

func manageProjectDeleteVariableEvent(db gorp.SqlExecutor, key string, varName string) error {
	if err := removeWarning(db, UnusedProjectVariable, varName); err != nil {
		log.Warning("manageDeleteVariableEvent> Unable to remove warning: %v", err)
	}
	used := variableIsUsed(db, key, varName)
	if !used {
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
		},
	}
	if err := insert(db, w); err != nil {
		return sdk.WrapError(err, "manageDeleteVariableEvent> Unable to insert warning")
	}

	return nil
}
