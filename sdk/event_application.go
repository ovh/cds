package sdk

// EventApplicationAdd represents the event when adding an application
type EventApplicationAdd struct {
	Application
}

// EventApplicationUpdate represents the event when updating an application
type EventApplicationUpdate struct {
	OldName               string             `json:"old_name"`
	NewName               string             `json:"new_name"`
	OldMetadata           Metadata           `json:"old_metadata"`
	NewMetadata           Metadata           `json:"new_metadata"`
	OldRepositoryStrategy RepositoryStrategy `json:"old_vcs_strategy"`
	NewRepositoryStrategy RepositoryStrategy `json:"new_vcs_strategy"`
}

// EventApplicationDelete represents the event when deleting an application
type EventApplicationDelete struct {
}

// EventApplicationVariableAdd represents the event when adding an application variable
type EventApplicationVariableAdd struct {
	Variable Variable `json:"variable"`
}

// EventApplicationVariableUpdate represents the event when updating an application variable
type EventApplicationVariableUpdate struct {
	OldVariable Variable `json:"old_variable"`
	NewVariable Variable `json:"new_variable"`
}

// EventApplicationVariableDelete represents the event when deleting an application variable
type EventApplicationVariableDelete struct {
	Variable Variable `json:"variable"`
}

// EventApplicationPermissionAdd represents the event when adding an application permission
type EventApplicationPermissionAdd struct {
	Permission GroupPermission `json:"group_permission"`
}

// EventApplicationPermissionUpdate represents the event when updating an application permission
type EventApplicationPermissionUpdate struct {
	OldPermission GroupPermission `json:"old_group_permission"`
	NewPermission GroupPermission `json:"new_group_permission"`
}

// EventApplicationPermissionDelete represents the event when deleting an application permission
type EventApplicationPermissionDelete struct {
	Permission GroupPermission `json:"group_permission"`
}

// EventApplicationKeyAdd represents the event when adding an application key
type EventApplicationKeyAdd struct {
	Key ApplicationKey `json:"key"`
}

// EventApplicationKeyDelete represents the event when deleting an application key
type EventApplicationKeyDelete struct {
	Key ApplicationKey `json:"key"`
}
