package exportentities

import (
	"github.com/ovh/cds/sdk"
)

// Application represents exported sdk.Application
type Application struct {
	Version        string                   `json:"version,omitempty" yaml:"version,omitempty"`
	Name           string                   `json:"name" yaml:"name"`
	VCSServer      string                   `json:"vcs_server,omitempty" yaml:"vcs_server,omitempty"`
	RepositoryName string                   `json:"repo,omitempty" yaml:"repo,omitempty"`
	Permissions    map[string]int           `json:"permissions,omitempty" yaml:"permissions,omitempty"`
	Variables      map[string]VariableValue `json:"variables,omitempty" yaml:"variables,omitempty"`
	Keys           map[string]VariableValue `json:"keys,omitempty" yaml:"keys,omitempty"`
}

type ApplicationVersion string

const ApplicationVersion1 = "v1.0"

// EncryptedKey represents an encrypted secret
type EncryptedKey struct {
	Type    string
	Name    string
	Content string
}

// NewApplication instanciance an exportable application from an sdk.Application
func NewApplication(app *sdk.Application, withPermissions bool, keys []EncryptedKey) (a Application, err error) {
	a.Version = ApplicationVersion1
	a.Name = app.Name

	if app.VCSServer != "" {
		a.VCSServer = app.VCSServer
		a.RepositoryName = app.RepositoryFullname
	}

	a.Variables = make(map[string]VariableValue, len(app.Variable))
	for _, v := range app.Variable {
		at := string(v.Type)
		if at == "string" {
			at = ""
		}
		a.Variables[v.Name] = VariableValue{
			Type:  at,
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
