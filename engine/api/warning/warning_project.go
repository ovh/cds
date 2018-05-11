package warning

import (
	"encoding/json"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func computeWithProjectEvent(db *gorp.DbMap, store cache.Store, e sdk.Event) {
	tx, errT := db.Begin()
	if errT != nil {
		log.Warning("computeWithProjectEvent> Unable to start transaction")
		return
	}
	defer tx.Rollback()

	payload, errP := json.Marshal(e.Payload)
	if errP != nil {
		log.Warning("computeWithProjectEvent> Unable to marshal event payload: %s", errP)
		return
	}

	switch e.EventType {

	case "sdk.EventProjectVariableAdd":
		var event sdk.EventProjectVariableAdd
		if err := json.Unmarshal(payload, &event); err != nil {
			log.Warning("computeWithProjectEvent> Unable to read EventProjectVariableAdd: %s", err)
		}
		manageAddVariableEvent(tx, e.ProjectKey, event.Variable.Name)
	case "sdk.EventProjectVariableUpdate":
		var event sdk.EventProjectVariableUpdate
		if err := json.Unmarshal(payload, &event); err != nil {
			log.Warning("computeWithProjectEvent> Unable to read EventProjectVariableUpdate: %s", err)
		}
		if event.NewVariable.Name == event.OldVariable.Name {
			return
		}

		if err := removeWarning(db, UnusedProjectVariable, event.OldVariable.Name); err != nil {
			log.Warning("computeWithProjectEvent.EventProjectVariableAdd> Unable to remove warning: %v", err)
		}
		if err := removeWarning(db, MissingProjectVariable, event.NewVariable.Name); err != nil {
			log.Warning("computeWithProjectEvent.EventProjectVariableAdd> Unable to remove warning: %v", err)
		}
		// If name changed, check if variable is used
		// Check if there is a warning on it

	case "sdk.EventProjectVariableDelete":
		var event sdk.EventProjectVariableDelete
		if err := json.Unmarshal(payload, &event); err != nil {
			log.Warning("computeWithProjectEvent> Unable to read EventProjectVariableDelete: %s", err)
		}

		if err := removeWarning(db, UnusedProjectVariable, event.Variable.Name); err != nil {
			log.Warning("computeWithProjectEvent.EventProjectVariableAdd> Unable to remove warning: %v", err)
		}
		// Check if variable is used
		// Check if there is a warning on it

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
		return
	}
}

func manageAddVariableEvent(db gorp.SqlExecutor, key string, varName string) {
	if err := removeWarning(db, MissingProjectVariable, varName); err != nil {
		log.Warning("computeWithProjectEvent.EventProjectVariableAdd> Unable to remove warning: %v", err)
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
			log.Warning("manageAddVariableEvent> Unable to insert warning")
		}
	}

}
