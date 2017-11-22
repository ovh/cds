package exportentities

import (
	"fmt"

	"github.com/ovh/cds/sdk"
)

// Application represents exported sdk.Application
type Application struct {
	Name           string                   `json:"name" yaml:"name"`
	VCSServer      string                   `json:"vcs_server,omitempty" yaml:"vcs_server,omitempty"`
	RepositoryName string                   `json:"repo,omitempty" yaml:"repo,omitempty"`
	Permissions    map[string]int           `json:"permissions,omitempty" yaml:"permissions,omitempty"`
	Variables      map[string]VariableValue `json:"variables,omitempty" yaml:"variables,omitempty"`
	Keys           map[string]VariableValue `json:"keys,omitempty" yaml:"keys,omitempty"`
}

type EncryptedKey struct {
	Type    string
	Name    string
	Content string
}

// NewApplication instanciance an exportable application from an sdk.Application
func NewApplication(app *sdk.Application, withPermissions bool, keys []EncryptedKey) (a Application, err error) {
	a.Name = app.Name

	if app.VCSServer != "" {
		a.VCSServer = app.VCSServer
		a.RepositoryName = app.RepositoryFullname
	}

	a.Variables = make(map[string]VariableValue, len(app.Variable))
	for _, v := range app.Variable {
		fmt.Println(v)
		a.Variables[v.Name] = VariableValue{
			Type:  string(v.Type),
			Value: v.Value,
		}
	}

	a.Permissions = make(map[string]int, len(app.ApplicationGroups))
	for _, p := range app.ApplicationGroups {
		a.Permissions[p.Group.Name] = p.Permission
	}

	a.Keys = make(map[string]VariableValue, len(keys))
	for _, e := range keys {
		a.Keys[e.Name] = VariableValue{
			Type:  e.Type,
			Value: e.Content,
		}
	}
	return a, nil
}

// Application returns a new sdk.Application
func (a *Application) Application() (*sdk.Application, error) {
	app := new(sdk.Application)

	app.Name = a.Name
	app.VCSServer = a.VCSServer
	app.RepositoryFullname = a.RepositoryName

	//Compute permissions
	for g, p := range a.Permissions {
		perm := sdk.GroupPermission{
			Group:      sdk.Group{Name: g},
			Permission: p,
		}
		app.ApplicationGroups = append(app.ApplicationGroups, perm)
	}

	//Compute parameters
	for p, v := range a.Variables {
		param := sdk.Variable{
			Name:  p,
			Type:  v.Type,
			Value: v.Value,
		}
		app.Variable = append(app.Variable, param)
	}

	return app, nil
}
