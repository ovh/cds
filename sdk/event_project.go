package sdk

// EventAddProject represents the event when adding a project
type EventAddProject struct {
	ProjectKey  string            `json:"project_key"`
	Variables   []Variable        `json:"variables"`
	Permissions []GroupPermission `json:"groups_permission"`
	Keys        []ProjectKey      `json:"keys"`
	Metadata    Metadata          `json:"metadata"`
}

// EventUpdateProject represents the event when updating a project
type EventUpdateProject struct {
	ProjectKey  string   `json:"project_key"`
	OldName     string   `json:"old_name"`
	NewName     string   `json:"new_name"`
	OldMetadata Metadata `json:"old_metadata"`
	NewMetadata Metadata `json:"new_metadata"`
}

// EventDeleteProject represents the event when deleting a project
type EventDeleteProject struct {
	ProjectKey string `json:"project_key"`
}

// EventAddProjectVariable represents the event when adding a project variable
type EventAddProjectVariable struct {
	ProjectKey string   `json:"project_key"`
	Variable   Variable `json:"variable"`
}

// EventUpdateProjectVariable represents the event when updating a project variable
type EventUpdateProjectVariable struct {
	ProjectKey  string   `json:"project_key"`
	OldVariable Variable `json:"old_variable"`
	NewVariable Variable `json:"new_variable"`
}

// EventDeleteProjectVariable represents the event when deleting a project variable
type EventDeleteProjectVariable struct {
	ProjectKey string   `json:"project_key"`
	Variable   Variable `json:"variable"`
}

// EventAddProjectPermission represents the event when adding a project permission
type EventAddProjectPermission struct {
	ProjectKey string          `json:"project_key"`
	Permission GroupPermission `json:"group_permission"`
}

// EventUpdateProjectPermission represents the event when updating a project permission
type EventUpdateProjectPermission struct {
	ProjectKey    string          `json:"project_key"`
	OldPermission GroupPermission `json:"old_group_permission"`
	NewPermission GroupPermission `json:"new_group_permission"`
}

// EventDeleteProjectPermission represents the event when deleting a project permission
type EventDeleteProjectPermission struct {
	ProjectKey string          `json:"project_key"`
	Permission GroupPermission `json:"group_permission"`
}

// EventAddProjectKey represents the event when adding a project key
type EventAddProjectKey struct {
	ProjectKey string     `json:"project_key"`
	Key        ProjectKey `json:"key"`
}

// EventDeleteProjectKey represents the event when deleting a project key
type EventDeleteProjectKey struct {
	ProjectKey string     `json:"project_key"`
	Key        ProjectKey `json:"key"`
}

// EventAddProjectVCSServer represents the event when adding a project vcs server
type EventAddProjectVCSServer struct {
	ProjectKey    string `json:"project_key"`
	VCSServerName string `json:"vcs_server"`
}

// EventDeleteProjectVCSServer represents the event when deleting a project vcs server
type EventDeleteProjectVCSServer struct {
	ProjectKey    string `json:"project_key"`
	VCSServerName string `json:"vcs_server"`
}

// EventAddProjectPlatform represents the event when adding a project platform
type EventAddProjectPlatform struct {
	ProjectKey string          `json:"project_key"`
	Platform   ProjectPlatform `json:"platform"`
}

// EventUpdateProjectPlatform represents the event when updating a project platform
type EventUpdateProjectPlatform struct {
	ProjectKey   string          `json:"project_key"`
	OldPlatform  ProjectPlatform `json:"old_platform"`
	NewsPlatform ProjectPlatform `json:"new_platform"`
}

// EventDeleteProjectPlatform represents the event when deleting a project platform
type EventDeleteProjectPlatform struct {
	ProjectKey string          `json:"project_key"`
	Platform   ProjectPlatform `json:"platform"`
}
