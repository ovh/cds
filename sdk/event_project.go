package sdk

import "github.com/mitchellh/mapstructure"

// EventProjectAdd represents the event when adding a project
type EventProjectAdd struct {
	Variables   []Variable        `json:"variables"`
	Permissions []GroupPermission `json:"groups_permission"`
	Keys        []ProjectKey      `json:"keys"`
	Metadata    Metadata          `json:"metadata"`
}

// EventProjectUpdate represents the event when updating a project
type EventProjectUpdate struct {
	OldName     string   `json:"old_name"`
	NewName     string   `json:"new_name"`
	OldMetadata Metadata `json:"old_metadata"`
	NewMetadata Metadata `json:"new_metadata"`
}

// EventProjectDelete represents the event when deleting a project
type EventProjectDelete struct {
}

// EventProjectVariableAdd represents the event when adding a project variable
type EventProjectVariableAdd struct {
	Variable Variable `json:"variable"`
}

// EventProjectVariableUpdate represents the event when updating a project variable
type EventProjectVariableUpdate struct {
	OldVariable Variable `json:"old_variable"`
	NewVariable Variable `json:"new_variable"`
}

// EventProjectVariableDelete represents the event when deleting a project variable
type EventProjectVariableDelete struct {
	Variable Variable `json:"variable"`
}

// EventProjectPermissionAdd represents the event when adding a project permission
type EventProjectPermissionAdd struct {
	Permission GroupPermission `json:"group_permission"`
}

// EventProjectPermissionUpdate represents the event when updating a project permission
type EventProjectPermissionUpdate struct {
	OldPermission GroupPermission `json:"old_group_permission"`
	NewPermission GroupPermission `json:"new_group_permission"`
}

// EventProjectPermissionDelete represents the event when deleting a project permission
type EventProjectPermissionDelete struct {
	Permission GroupPermission `json:"group_permission"`
}

// EventProjectKeyAdd represents the event when adding a project key
type EventProjectKeyAdd struct {
	Key ProjectKey `json:"key"`
}

// EventProjectKeyDelete represents the event when deleting a project key
type EventProjectKeyDelete struct {
	Key ProjectKey `json:"key"`
}

// EventProjectVCSServerAdd represents the event when adding a project vcs server
type EventProjectVCSServerAdd struct {
	VCSServerName string `json:"vcs_server"`
}

// EventProjectVCSServerDelete represents the event when deleting a project vcs server
type EventProjectVCSServerDelete struct {
	VCSServerName string `json:"vcs_server"`
}

// EventProjectPlatformAdd represents the event when adding a project platform
type EventProjectPlatformAdd struct {
	Platform ProjectPlatform `json:"platform"`
}

// EventProjectPlatformUpdate represents the event when updating a project platform
type EventProjectPlatformUpdate struct {
	OldPlatform  ProjectPlatform `json:"old_platform"`
	NewsPlatform ProjectPlatform `json:"new_platform"`
}

// EventProjectPlatformDelete represents the event when deleting a project platform
type EventProjectPlatformDelete struct {
	Platform ProjectPlatform `json:"platform"`
}

// ToEventProjectVariableAdd get the payload as EventProjectVariableAdd
func (e Event) ToEventProjectVariableAdd() (EventProjectVariableAdd, error) {
	var varEvent EventProjectVariableAdd
	if err := mapstructure.Decode(e.Payload, &varEvent); err != nil {
		return varEvent, WrapError(err, "ToEventProjectVariableAdd> Unable to decode EventProjectVariableAdd")
	}
	return varEvent, nil
}

// ToEventProjectVariableUpdate get the payload as EventProjectVariableUpdate
func (e Event) ToEventProjectVariableUpdate() (EventProjectVariableUpdate, error) {
	var varEvent EventProjectVariableUpdate
	if err := mapstructure.Decode(e.Payload, &varEvent); err != nil {
		return varEvent, WrapError(err, "ToEventProjectVariableUpdate> Unable to decode EventProjectVariableUpdate")
	}
	return varEvent, nil
}

// ToEventProjectVariableDelete get the payload as EventProjectVariableDelete
func (e Event) ToEventProjectVariableDelete() (EventProjectVariableDelete, error) {
	var varEvent EventProjectVariableDelete
	if err := mapstructure.Decode(e.Payload, &varEvent); err != nil {
		return varEvent, WrapError(err, "ToEventProjectVariableDelete> Unable to decode EventProjectVariableDelete")
	}
	return varEvent, nil
}

// ToEventProjectPermissionAdd get the payload as EventProjectPermissionAdd
func (e Event) ToEventProjectPermissionAdd() (EventProjectPermissionAdd, error) {
	var permEvent EventProjectPermissionAdd
	if err := mapstructure.Decode(e.Payload, &permEvent); err != nil {
		return permEvent, WrapError(err, "ToEventProjectPermissionAdd> Unable to decode EventProjectPermissionAdd")
	}
	return permEvent, nil
}

// ToEventProjectPermissionDelete get the payload as EventProjectPermissionDelete
func (e Event) ToEventProjectPermissionDelete() (EventProjectPermissionDelete, error) {
	var permEvent EventProjectPermissionDelete
	if err := mapstructure.Decode(e.Payload, &permEvent); err != nil {
		return permEvent, WrapError(err, "ToEventProjectPermissionDelete> Unable to decode EventProjectPermissionDelete")
	}
	return permEvent, nil
}

// ToEventProjectKeyAdd get the payload as EventProjectKeyAdd
func (e Event) ToEventProjectKeyAdd() (EventProjectKeyAdd, error) {
	var keyEvent EventProjectKeyAdd
	if err := mapstructure.Decode(e.Payload, &keyEvent); err != nil {
		return keyEvent, WrapError(err, "ToEventProjectKeyAdd> Unable to decode EventProjectKeyAdd")
	}
	return keyEvent, nil
}

// ToEventProjectKeyDelete get the payload as EventProjectKeyDelete
func (e Event) ToEventProjectKeyDelete() (EventProjectKeyDelete, error) {
	var keyEvent EventProjectKeyDelete
	if err := mapstructure.Decode(e.Payload, &keyEvent); err != nil {
		return keyEvent, WrapError(err, "ToEventProjectKeyDelete> Unable to decode EventProjectKeyDelete")
	}
	return keyEvent, nil
}

// ToEventProjectVCSServerAdd get the payload as EventProjectVCSServerAdd
func (e Event) ToEventProjectVCSServerAdd() (EventProjectVCSServerAdd, error) {
	var vcsEvent EventProjectVCSServerAdd
	if err := mapstructure.Decode(e.Payload, &vcsEvent); err != nil {
		return vcsEvent, WrapError(err, "ToEventProjectVCSServerAdd> Unable to decode EventProjectVCSServerAdd")
	}
	return vcsEvent, nil
}

// ToEventProjectVCSServerDelete get the payload as EventProjectVCSServerDelete
func (e Event) ToEventProjectVCSServerDelete() (EventProjectVCSServerDelete, error) {
	var vcsEvent EventProjectVCSServerDelete
	if err := mapstructure.Decode(e.Payload, &vcsEvent); err != nil {
		return vcsEvent, WrapError(err, "ToEventProjectVCSServerDelete> Unable to decode EventProjectVCSServerDelete")
	}
	return vcsEvent, nil
}
