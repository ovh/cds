package sdk

// EventAddApplication represents the event when adding an application
type EventAddApplication struct {
	Application
}

// EventUpdateApplication represents the event when updating an application
type EventUpdateApplication struct {
	OldName               string             `json:"old_name"`
	NewName               string             `json:"new_name"`
	OldMetadata           Metadata           `json:"old_metadata"`
	NewMetadata           Metadata           `json:"new_metadata"`
	OldRepositoryStrategy RepositoryStrategy `json:"old_vcs_strategy"`
	NewRepositoryStrategy RepositoryStrategy `json:"new_vcs_strategy"`
}

// EventDeleteApplication represents the event when deleting an application
type EventDeleteApplication struct {
}

// EventAddApplicationVariable represents the event when adding an application variable
type EventAddApplicationVariable struct {
	Variable Variable `json:"variable"`
}

// EventUpdateApplicationVariable represents the event when updating an application variable
type EventUpdateApplicationVariable struct {
	OldVariable Variable `json:"old_variable"`
	NewVariable Variable `json:"new_variable"`
}

// EventDeleteApplicationVariable represents the event when deleting an application variable
type EventDeleteApplicationVariable struct {
	Variable Variable `json:"variable"`
}

// EventAddApplicationPermission represents the event when adding an application permission
type EventAddApplicationPermission struct {
	Permission GroupPermission `json:"group_permission"`
}

// EventUpdateApplicationPermission represents the event when updating an application permission
type EventUpdateApplicationPermission struct {
	OldPermission GroupPermission `json:"old_group_permission"`
	NewPermission GroupPermission `json:"new_group_permission"`
}

// EventDeleteApplicationPermission represents the event when deleting an application permission
type EventDeleteApplicationPermission struct {
	Permission GroupPermission `json:"group_permission"`
}

// EventAddApplicationKey represents the event when adding an application key
type EventAddApplicationKey struct {
	Key ApplicationKey `json:"key"`
}

// EventDeleteApplicationKey represents the event when deleting an application key
type EventDeleteApplicationKey struct {
	Key ApplicationKey `json:"key"`
}

/*
type Application struct {
	ID          int64  `json:"id" db:"id"`
	Name        string `json:"name" db:"name" cli:"name,key"`
	Description string `json:"description"  db:"description"`



	VCSServer          string              `json:"vcs_server,omitempty" db:"vcs_server"`
	RepositoryFullname string              `json:"repository_fullname,omitempty" db:"repo_fullname"`

	RepositoryStrategy RepositoryStrategy  `json:"vcs_strategy,omitempty" db:"-"`


	Metadata           Metadata            `json:"metadata" yaml:"metadata" db:"-"`

	Keys               []ApplicationKey    `json:"keys" yaml:"keys" db:"-"`
}
*/
