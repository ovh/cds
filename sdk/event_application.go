package sdk

import "github.com/mitchellh/mapstructure"

// EventApplicationAdd represents the event when adding an application
//easyjson:json
type EventApplicationAdd struct {
	Application
}

// EventApplicationUpdate represents the event when updating an application
//easyjson:json
type EventApplicationUpdate struct {
	OldName               string             `json:"old_name"`
	NewName               string             `json:"new_name"`
	OldMetadata           Metadata           `json:"old_metadata"`
	NewMetadata           Metadata           `json:"new_metadata"`
	OldRepositoryStrategy RepositoryStrategy `json:"old_vcs_strategy"`
	NewRepositoryStrategy RepositoryStrategy `json:"new_vcs_strategy"`
}

// EventApplicationDelete represents the event when deleting an application
//easyjson:json
type EventApplicationDelete struct {
}

// EventApplicationVariableAdd represents the event when adding an application variable
//easyjson:json
type EventApplicationVariableAdd struct {
	Variable Variable `json:"variable"`
}

// EventApplicationVariableUpdate represents the event when updating an application variable
//easyjson:json
type EventApplicationVariableUpdate struct {
	OldVariable Variable `json:"old_variable"`
	NewVariable Variable `json:"new_variable"`
}

// EventApplicationVariableDelete represents the event when deleting an application variable
//easyjson:json
type EventApplicationVariableDelete struct {
	Variable Variable `json:"variable"`
}

// EventApplicationPermissionAdd represents the event when adding an application permission
//easyjson:json
type EventApplicationPermissionAdd struct {
	Permission GroupPermission `json:"group_permission"`
}

// EventApplicationPermissionUpdate represents the event when updating an application permission
//easyjson:json
type EventApplicationPermissionUpdate struct {
	OldPermission GroupPermission `json:"old_group_permission"`
	NewPermission GroupPermission `json:"new_group_permission"`
}

// EventApplicationPermissionDelete represents the event when deleting an application permission
//easyjson:json
type EventApplicationPermissionDelete struct {
	Permission GroupPermission `json:"group_permission"`
}

// EventApplicationKeyAdd represents the event when adding an application key
//easyjson:json
type EventApplicationKeyAdd struct {
	Key ApplicationKey `json:"key"`
}

// EventApplicationKeyDelete represents the event when deleting an application key
//easyjson:json
type EventApplicationKeyDelete struct {
	Key ApplicationKey `json:"key"`
}

// EventApplicationRepositoryAdd represents the event when adding a repository to an application
//easyjson:json
type EventApplicationRepositoryAdd struct {
	VCSServer  string `json:"vcs_server"`
	Repository string `json:"repository"`
}

// EventApplicationRepositoryDelete represents the event when deleting a repository to an application
//easyjson:json
type EventApplicationRepositoryDelete struct {
	VCSServer  string `json:"vcs_server"`
	Repository string `json:"repository"`
}

// EventApplicationVulnerabilityUpdate represents the event when updating a vulnerability
//easyjson:json
type EventApplicationVulnerabilityUpdate struct {
	OldVulnerability Vulnerability `json:"old_vulnerability"`
	NewVulnerability Vulnerability `json:"new_vulnerability"`
}

// ToEventApplicationRepositoryAdd get the payload as EventApplicationRepositoryAdd
func (e Event) ToEventApplicationRepositoryAdd() (EventApplicationRepositoryAdd, error) {
	var vcsEvent EventApplicationRepositoryAdd
	if err := mapstructure.Decode(e.Payload, &vcsEvent); err != nil {
		return vcsEvent, WrapError(err, "ToEventApplicationRepositoryAdd> Unable to decode EventApplicationRepositoryAdd")
	}
	return vcsEvent, nil
}

// ToEventApplicationRepositoryDelete get the payload as EventApplicationRepositoryDelete
func (e Event) ToEventApplicationRepositoryDelete() (EventApplicationRepositoryDelete, error) {
	var vcsEvent EventApplicationRepositoryDelete
	if err := mapstructure.Decode(e.Payload, &vcsEvent); err != nil {
		return vcsEvent, WrapError(err, "ToEventApplicationRepositoryDelete> Unable to decode EventApplicationRepositoryDelete")
	}
	return vcsEvent, nil
}

// ToEventApplicationVariableAdd get the payload as EventApplicationVariableAdd
func (e Event) ToEventApplicationVariableAdd() (EventApplicationVariableAdd, error) {
	var varEvent EventApplicationVariableAdd
	if err := mapstructure.Decode(e.Payload, &varEvent); err != nil {
		return varEvent, WrapError(err, "ToEventApplicationVariableAdd> Unable to decode EventApplicationVariableAdd")
	}
	return varEvent, nil
}

// ToEventApplicationVariableUpdate get the payload as EventApplicationVariableUpdate
func (e Event) ToEventApplicationVariableUpdate() (EventApplicationVariableUpdate, error) {
	var varEvent EventApplicationVariableUpdate
	if err := mapstructure.Decode(e.Payload, &varEvent); err != nil {
		return varEvent, WrapError(err, "ToEventApplicationVariableUpdate> Unable to decode EventApplicationVariableUpdate")
	}
	return varEvent, nil
}

// ToEventApplicationVariableDelete get the payload as EventApplicationVariableDelete
func (e Event) ToEventApplicationVariableDelete() (EventApplicationVariableDelete, error) {
	var varEvent EventApplicationVariableDelete
	if err := mapstructure.Decode(e.Payload, &varEvent); err != nil {
		return varEvent, WrapError(err, "ToEventApplicationVariableDelete> Unable to decode EventApplicationVariableDelete")
	}
	return varEvent, nil
}
