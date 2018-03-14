package event

import (
	"github.com/ovh/cds/sdk"
)

// PublishAddProject publishes an event for the creation of the given project
func PublishAddProject(p *sdk.Project, u *sdk.User) {
	e := sdk.EventAddProject{
		ProjectKey:  p.Key,
		Variables:   p.Variable,
		Permissions: p.ProjectGroups,
		Keys:        p.Keys,
		Metadata:    p.Metadata,
	}
	Publish(e, u)
}

// PublishUpdateProject publishes an event for the modification of the project
func PublishUpdateProject(p *sdk.Project, oldProject *sdk.Project, u *sdk.User) {
	e := sdk.EventUpdateProject{
		ProjectKey:  p.Key,
		NewName:     p.Name,
		NewMetadata: p.Metadata,
		OldMetadata: oldProject.Metadata,
		OldName:     oldProject.Name,
	}
	Publish(e, u)
}

// PublishDeleteProject publishess an event for the deletion of the given project
func PublishDeleteProject(p *sdk.Project, u *sdk.User) {
	e := sdk.EventDeleteProject{
		ProjectKey: p.Key,
	}
	Publish(e, u)
}

// PublishAddProjectVariable publishes an event for the creation of the given variable
func PublishAddProjectVariable(p *sdk.Project, v sdk.Variable, u *sdk.User) {
	if sdk.NeedPlaceholder(v.Type) {
		v.Value = sdk.PasswordPlaceholder
	}
	e := sdk.EventAddProjectVariable{
		Variable:   v,
		ProjectKey: p.Key,
	}
	Publish(e, u)
}

// PublishUpdateProjectVariable publishes an event for the modification of a variable
func PublishUpdateProjectVariable(p *sdk.Project, newVar sdk.Variable, oldVar sdk.Variable, u *sdk.User) {
	if sdk.NeedPlaceholder(newVar.Type) {
		newVar.Value = sdk.PasswordPlaceholder
	}
	if sdk.NeedPlaceholder(oldVar.Type) {
		oldVar.Value = sdk.PasswordPlaceholder
	}
	e := sdk.EventUpdateProjectVariable{
		ProjectKey:  p.Key,
		NewVariable: newVar,
		OldVariable: oldVar,
	}
	Publish(e, u)
}

// PublishDeleteProjectVariable publishes an event on project variable deletion
func PublishDeleteProjectVariable(p *sdk.Project, v sdk.Variable, u *sdk.User) {
	if sdk.NeedPlaceholder(v.Type) {
		v.Value = sdk.PasswordPlaceholder
	}
	e := sdk.EventDeleteProjectVariable{
		ProjectKey: p.Key,
		Variable:   v,
	}
	Publish(e, u)
}

// PublishAddProjectPermission publishes an event on adding a group permission on the project
func PublishAddProjectPermission(p *sdk.Project, gp sdk.GroupPermission, u *sdk.User) {
	e := sdk.EventAddProjectPermission{
		ProjectKey: p.Key,
		Permission: gp,
	}
	Publish(e, u)
}

// PublishUpdateProjectPermission publishes an event on updating a group permission on the project
func PublishUpdateProjectPermission(p *sdk.Project, gp sdk.GroupPermission, oldGP sdk.GroupPermission, u *sdk.User) {
	e := sdk.EventUpdateProjectPermission{
		ProjectKey:    p.Key,
		NewPermission: gp,
		OldPermission: oldGP,
	}
	Publish(e, u)
}

// PublishDeleteProjectPermission publishes an event on deleting a group permission on the project
func PublishDeleteProjectPermission(p *sdk.Project, gp sdk.GroupPermission, u *sdk.User) {
	e := sdk.EventDeleteProjectPermission{
		ProjectKey: p.Key,
		Permission: gp,
	}
	Publish(e, u)
}

// PublishAddProjectKey publishes an event on adding a project key
func PublishAddProjectKey(p *sdk.Project, k sdk.ProjectKey, u *sdk.User) {
	k.Private = sdk.PasswordPlaceholder
	e := sdk.EventAddProjectKey{
		ProjectKey: p.Key,
		Key:        k,
	}
	Publish(e, u)
}

// PublishDeleteProjectKey publishes an event on deleting a project key
func PublishDeleteProjectKey(p *sdk.Project, k sdk.ProjectKey, u *sdk.User) {
	if sdk.NeedPlaceholder(k.Type) {
		k.Private = sdk.PasswordPlaceholder
	}
	e := sdk.EventAddProjectKey{
		ProjectKey: p.Key,
		Key:        k,
	}
	Publish(e, u)
}
