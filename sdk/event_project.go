package sdk

// EventProjectAdd represents the event when adding a project
type EventProjectAdd struct {
	Variables   []ProjectVariable `json:"variables"`
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
	Variable ProjectVariable `json:"variable"`
}

// EventProjectVariableUpdate represents the event when updating a project variable
type EventProjectVariableUpdate struct {
	OldVariable ProjectVariable `json:"old_variable"`
	NewVariable ProjectVariable `json:"new_variable"`
}

// EventProjectVariableDelete represents the event when deleting a project variable
type EventProjectVariableDelete struct {
	Variable ProjectVariable `json:"variable"`
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

type EventProjectKeyDisable struct {
	Key ProjectKey `json:"key"`
}

type EventProjectKeyEnable struct {
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

// EventProjectIntegrationAdd represents the event when adding a project integration
type EventProjectIntegrationAdd struct {
	Integration ProjectIntegration `json:"integration"`
}

// EventProjectIntegrationUpdate represents the event when updating a project integration
type EventProjectIntegrationUpdate struct {
	OldIntegration  ProjectIntegration `json:"old_integration"`
	NewsIntegration ProjectIntegration `json:"new_integration"`
}

// EventProjectIntegrationDelete represents the event when deleting a project integration
type EventProjectIntegrationDelete struct {
	Integration ProjectIntegration `json:"integration"`
}

type EventProjectRepositoryDelete struct {
	Repository ProjectRepository `json:"repository"`
	VCS        VCSProject        `json:"vcs"`
}

type EventProjectRepositoryAdd struct {
	Repository ProjectRepository `json:"repository"`
	VCS        VCSProject        `json:"vcs"`
}

type EventProjectRepositoryAnalyze struct {
	RepositoryID string `json:"repository_id"`
	VCSID        string `json:"vcs_id"`
	AnalysisID   string `json:"analysis_id"`
	Status       string `json:"status"`
}
