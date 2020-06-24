package sdk

// EventEnvironmentAdd represents the event when adding an environment
type EventEnvironmentAdd struct {
	Environment
}

// EventEnvironmentUpdate represents the event when updating an environment
type EventEnvironmentUpdate struct {
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
}

// EventEnvironmentDelete represents the event when deleting an environment
type EventEnvironmentDelete struct {
}

// EventEnvironmentVariableAdd represents the event when adding an environment variable
type EventEnvironmentVariableAdd struct {
	Variable EnvironmentVariable `json:"variable"`
}

// EventEnvironmentVariableUpdate represents the event when updating an environment variable
type EventEnvironmentVariableUpdate struct {
	OldVariable EnvironmentVariable `json:"old_variable"`
	NewVariable EnvironmentVariable `json:"new_variable"`
}

// EventEnvironmentVariableDelete represents the event when deleting an environment variable
type EventEnvironmentVariableDelete struct {
	Variable EnvironmentVariable `json:"variable"`
}

// EventEnvironmentPermissionAdd represents the event when adding an environment permission
type EventEnvironmentPermissionAdd struct {
	Permission GroupPermission `json:"group_permission"`
}

// EventEnvironmentPermissionUpdate represents the event when updating an environment permission
type EventEnvironmentPermissionUpdate struct {
	OldPermission GroupPermission `json:"old_group_permission"`
	NewPermission GroupPermission `json:"new_group_permission"`
}

// EventEnvironmentPermissionDelete represents the event when deleting an environment permission
type EventEnvironmentPermissionDelete struct {
	Permission GroupPermission `json:"group_permission"`
}

// EventEnvironmentKeyAdd represents the event when adding an environment key
type EventEnvironmentKeyAdd struct {
	Key EnvironmentKey `json:"key"`
}

// EventEnvironmentKeyDelete represents the event when deleting an environment key
type EventEnvironmentKeyDelete struct {
	Key EnvironmentKey `json:"key"`
}
