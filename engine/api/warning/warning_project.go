package warning

import (
	"encoding/json"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func computeWithProjectEvent(db gorp.SqlExecutor, store cache.Store, e sdk.Event) error {

	payload, errP := json.Marshal(e.Payload)
	if errP != nil {
		return sdk.WrapError(errP, "computeWithProjectEvent> Unable to marshal event payload")
	}

	switch e.EventType {

	case "sdk.EventProjectVariableAdd":
		var event sdk.EventProjectVariableAdd
		if err := json.Unmarshal(payload, &event); err != nil {
			return sdk.WrapError(err, "computeWithProjectEvent> Unable to read EventProjectVariableAdd")
		}
		return manageAddVariableEvent(db, e.ProjectKey, event.Variable.Name)
	case "sdk.EventProjectVariableUpdate":
		var event sdk.EventProjectVariableUpdate
		if err := json.Unmarshal(payload, &event); err != nil {
			return sdk.WrapError(err, "computeWithProjectEvent> Unable to read EventProjectVariableUpdate")
		}
		return manageUpdateVariableEvent(db, e.ProjectKey, event.NewVariable, event.OldVariable)
	case "sdk.EventProjectVariableDelete":
		var event sdk.EventProjectVariableDelete
		if err := json.Unmarshal(payload, &event); err != nil {
			return sdk.WrapError(err, "computeWithProjectEvent> Unable to read EventProjectVariableDelete")
		}
		return manageDeleteVariableEvent(db, e.ProjectKey, event.Variable.Name)
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

func manageAddVariableEvent(db gorp.SqlExecutor, key string, varName string) error {
	if err := removeWarning(db, MissingProjectVariable, varName); err != nil {
		return sdk.WrapError(err, "manageAddVariableEvent> Unable to remove warning")
	}

	used := variableIsUsed(db, key, ".cds.proj."+varName)
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

func manageUpdateVariableEvent(db gorp.SqlExecutor, key string, newVar sdk.Variable, oldVar sdk.Variable) error {
	if newVar.Name == oldVar.Name {
		return nil
	}

	if err := removeWarning(db, UnusedProjectVariable, oldVar.Name); err != nil {
		log.Warning("manageUpdateVariableEvent> Unable to remove oldvar warning: %v", err)
	}
	if err := removeWarning(db, MissingProjectVariable, newVar.Name); err != nil {
		log.Warning("manageUpdateVariableEvent> Unable to remove newvar warning: %v", err)
	}

	used := variableIsUsed(db, key, ".cds.proj."+newVar.Name)
	if !used {
		w := sdk.WarningV2{
			Key:     key,
			Element: newVar.Name,
			Created: time.Now(),
			Type:    UnusedProjectVariable,
			MessageParams: map[string]string{
				"VarName":    newVar.Name,
				"ProjectKey": key,
			},
		}
		if err := insert(db, w); err != nil {
			return sdk.WrapError(err, "manageUpdateVariableEvent> Unable to insert warning")
		}
	}
	return nil
}

func manageDeleteVariableEvent(db gorp.SqlExecutor, key string, varName string) error {
	if err := removeWarning(db, UnusedProjectVariable, varName); err != nil {
		log.Warning("manageDeleteVariableEvent> Unable to remove warning: %v", err)
	}
	used := variableIsUsed(db, key, ".cds.proj."+varName)
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
