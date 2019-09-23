package sdk

import "github.com/mitchellh/mapstructure"

// EventEnvironmentAdd represents the event when adding an environment
//easyjson:json
type EventEnvironmentAdd struct {
	Environment
}

// EventEnvironmentUpdate represents the event when updating an environment
//easyjson:json
type EventEnvironmentUpdate struct {
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
}

// EventEnvironmentDelete represents the event when deleting an environment
//easyjson:json
type EventEnvironmentDelete struct {
}

// EventEnvironmentVariableAdd represents the event when adding an environment variable
//easyjson:json
type EventEnvironmentVariableAdd struct {
	Variable Variable `json:"variable"`
}

// EventEnvironmentVariableUpdate represents the event when updating an environment variable
//easyjson:json
type EventEnvironmentVariableUpdate struct {
	OldVariable Variable `json:"old_variable"`
	NewVariable Variable `json:"new_variable"`
}

// EventEnvironmentVariableDelete represents the event when deleting an environment variable
//easyjson:json
type EventEnvironmentVariableDelete struct {
	Variable Variable `json:"variable"`
}

// EventEnvironmentPermissionAdd represents the event when adding an environment permission
//easyjson:json
type EventEnvironmentPermissionAdd struct {
	Permission GroupPermission `json:"group_permission"`
}

// EventEnvironmentPermissionUpdate represents the event when updating an environment permission
//easyjson:json
type EventEnvironmentPermissionUpdate struct {
	OldPermission GroupPermission `json:"old_group_permission"`
	NewPermission GroupPermission `json:"new_group_permission"`
}

// EventEnvironmentPermissionDelete represents the event when deleting an environment permission
//easyjson:json
type EventEnvironmentPermissionDelete struct {
	Permission GroupPermission `json:"group_permission"`
}

// EventEnvironmentKeyAdd represents the event when adding an environment key
//easyjson:json
type EventEnvironmentKeyAdd struct {
	Key EnvironmentKey `json:"key"`
}

// EventEnvironmentKeyDelete represents the event when deleting an environment key
//easyjson:json
type EventEnvironmentKeyDelete struct {
	Key EnvironmentKey `json:"key"`
}

// ToEventEnvironmentPermissionAdd get the payload as EventEnvironmentPermissionAdd
func (e Event) ToEventEnvironmentPermissionAdd() (EventEnvironmentPermissionAdd, error) {
	var permEvent EventEnvironmentPermissionAdd
	if err := mapstructure.Decode(e.Payload, &permEvent); err != nil {
		return permEvent, WrapError(err, "ToEventEnvironmentPermissionAdd> Unable to decode EventEnvironmentPermissionAdd")
	}
	return permEvent, nil
}

// ToEventEnvironmentPermissionDelete get the payload as EventEnvironmentPermissionDelete
func (e Event) ToEventEnvironmentPermissionDelete() (EventEnvironmentPermissionDelete, error) {
	var permEvent EventEnvironmentPermissionDelete
	if err := mapstructure.Decode(e.Payload, &permEvent); err != nil {
		return permEvent, WrapError(err, "ToEventEnvironmentPermissionDelete> Unable to decode EventEnvironmentPermissionAdd")
	}
	return permEvent, nil
}
