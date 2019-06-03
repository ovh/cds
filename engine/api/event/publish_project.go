package event

import (
	"fmt"
	"time"

	"github.com/fatih/structs"

	"github.com/ovh/cds/sdk"
)

// PublishProjectEvent publish application event
func PublishProjectEvent(payload interface{}, key string, u sdk.Identifiable) {
	event := sdk.Event{
		Timestamp:  time.Now(),
		Hostname:   hostname,
		CDSName:    cdsname,
		EventType:  fmt.Sprintf("%T", payload),
		Payload:    structs.Map(payload),
		ProjectKey: key,
	}
	if u != nil {
		event.UserMail = u.GetEmail()
		event.Username = u.GetUsername()
	}
	publishEvent(event)
}

// PublishAddProject publishes an event for the creation of the given project
func PublishAddProject(p *sdk.Project, u sdk.Identifiable) {
	e := sdk.EventProjectAdd{
		Variables:   p.Variable,
		Permissions: p.ProjectGroups,
		Keys:        p.Keys,
		Metadata:    p.Metadata,
	}
	PublishProjectEvent(e, p.Key, u)
}

// PublishUpdateProject publishes an event for the modification of the project
func PublishUpdateProject(p *sdk.Project, oldProject *sdk.Project, u sdk.Identifiable) {
	e := sdk.EventProjectUpdate{
		NewName:     p.Name,
		NewMetadata: p.Metadata,
		OldMetadata: oldProject.Metadata,
		OldName:     oldProject.Name,
	}
	PublishProjectEvent(e, p.Key, u)
}

// PublishDeleteProject publishess an event for the deletion of the given project
func PublishDeleteProject(p *sdk.Project, u sdk.Identifiable) {
	e := sdk.EventProjectDelete{}
	PublishProjectEvent(e, p.Key, u)
}

// PublishAddProjectVariable publishes an event for the creation of the given variable
func PublishAddProjectVariable(p *sdk.Project, v sdk.Variable, u sdk.Identifiable) {
	if sdk.NeedPlaceholder(v.Type) {
		v.Value = sdk.PasswordPlaceholder
	}
	e := sdk.EventProjectVariableAdd{
		Variable: v,
	}
	PublishProjectEvent(e, p.Key, u)
}

// PublishUpdateProjectVariable publishes an event for the modification of a variable
func PublishUpdateProjectVariable(p *sdk.Project, newVar sdk.Variable, oldVar sdk.Variable, u sdk.Identifiable) {
	if sdk.NeedPlaceholder(newVar.Type) {
		newVar.Value = sdk.PasswordPlaceholder
	}
	if sdk.NeedPlaceholder(oldVar.Type) {
		oldVar.Value = sdk.PasswordPlaceholder
	}
	e := sdk.EventProjectVariableUpdate{
		NewVariable: newVar,
		OldVariable: oldVar,
	}
	PublishProjectEvent(e, p.Key, u)
}

// PublishDeleteProjectVariable publishes an event on project variable deletion
func PublishDeleteProjectVariable(p *sdk.Project, v sdk.Variable, u sdk.Identifiable) {
	if sdk.NeedPlaceholder(v.Type) {
		v.Value = sdk.PasswordPlaceholder
	}
	e := sdk.EventProjectVariableDelete{
		Variable: v,
	}
	PublishProjectEvent(e, p.Key, u)
}

// PublishAddProjectPermission publishes an event on adding a group permission on the project
func PublishAddProjectPermission(p *sdk.Project, gp sdk.GroupPermission, u sdk.Identifiable) {
	e := sdk.EventProjectPermissionAdd{
		Permission: gp,
	}
	PublishProjectEvent(e, p.Key, u)
}

// PublishUpdateProjectPermission publishes an event on updating a group permission on the project
func PublishUpdateProjectPermission(p *sdk.Project, gp sdk.GroupPermission, oldGP sdk.GroupPermission, u sdk.Identifiable) {
	e := sdk.EventProjectPermissionUpdate{
		NewPermission: gp,
		OldPermission: oldGP,
	}
	PublishProjectEvent(e, p.Key, u)
}

// PublishDeleteProjectPermission publishes an event on deleting a group permission on the project
func PublishDeleteProjectPermission(p *sdk.Project, gp sdk.GroupPermission, u sdk.Identifiable) {
	e := sdk.EventProjectPermissionDelete{
		Permission: gp,
	}
	PublishProjectEvent(e, p.Key, u)
}

// PublishAddProjectKey publishes an event on adding a project key
func PublishAddProjectKey(p *sdk.Project, k sdk.ProjectKey, u sdk.Identifiable) {
	k.Private = sdk.PasswordPlaceholder
	e := sdk.EventProjectKeyAdd{
		Key: k,
	}
	PublishProjectEvent(e, p.Key, u)
}

// PublishDeleteProjectKey publishes an event on deleting a project key
func PublishDeleteProjectKey(p *sdk.Project, k sdk.ProjectKey, u sdk.Identifiable) {
	if sdk.NeedPlaceholder(k.Type) {
		k.Private = sdk.PasswordPlaceholder
	}
	e := sdk.EventProjectKeyDelete{
		Key: k,
	}
	PublishProjectEvent(e, p.Key, u)
}

// PublishAddVCSServer publishes an event on adding a project server
func PublishAddVCSServer(p *sdk.Project, vcsServerName string, u sdk.Identifiable) {
	e := sdk.EventProjectVCSServerAdd{
		VCSServerName: vcsServerName,
	}
	PublishProjectEvent(e, p.Key, u)
}

// PublishDeleteVCSServer publishes an event on deleting a project server
func PublishDeleteVCSServer(p *sdk.Project, vcsServerName string, u sdk.Identifiable) {
	e := sdk.EventProjectVCSServerDelete{
		VCSServerName: vcsServerName,
	}
	PublishProjectEvent(e, p.Key, u)
}

// PublishAddProjectIntegration publishes an event on adding a integration
func PublishAddProjectIntegration(p *sdk.Project, pf sdk.ProjectIntegration, u sdk.Identifiable) {
	pf.HideSecrets()
	e := sdk.EventProjectIntegrationAdd{
		Integration: pf,
	}
	PublishProjectEvent(e, p.Key, u)
}

// PublishUpdateProjectIntegration publishes an event on updating a integration
func PublishUpdateProjectIntegration(p *sdk.Project, pf sdk.ProjectIntegration, pfOld sdk.ProjectIntegration, u sdk.Identifiable) {
	pf.HideSecrets()
	pfOld.HideSecrets()
	e := sdk.EventProjectIntegrationUpdate{
		NewsIntegration: pf,
		OldIntegration:  pfOld,
	}
	PublishProjectEvent(e, p.Key, u)
}

// PublishDeleteProjectIntegration publishes an event on deleting integration
func PublishDeleteProjectIntegration(p *sdk.Project, pf sdk.ProjectIntegration, u sdk.Identifiable) {
	e := sdk.EventProjectIntegrationDelete{
		Integration: pf,
	}
	PublishProjectEvent(e, p.Key, u)
}
