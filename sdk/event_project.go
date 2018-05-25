package sdk

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
