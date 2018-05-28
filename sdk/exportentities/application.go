package exportentities

import (
	"github.com/ovh/cds/sdk"
)

// Application represents exported sdk.Application
type Application struct {
	Version              string                              `json:"version,omitempty" yaml:"version,omitempty"`
	Name                 string                              `json:"name" yaml:"name"`
	VCSServer            string                              `json:"vcs_server,omitempty" yaml:"vcs_server,omitempty"`
	RepositoryName       string                              `json:"repo,omitempty" yaml:"repo,omitempty"`
	Permissions          map[string]int                      `json:"permissions,omitempty" yaml:"permissions,omitempty"`
	Variables            map[string]VariableValue            `json:"variables,omitempty" yaml:"variables,omitempty"`
	Keys                 map[string]KeyValue                 `json:"keys,omitempty" yaml:"keys,omitempty"`
	VCSConnectionType    string                              `json:"vcs_connection_type,omitempty" yaml:"vcs_connection_type,omitempty"`
	VCSSSHKey            string                              `json:"vcs_ssh_key,omitempty" yaml:"vcs_ssh_key,omitempty"`
	VCSUser              string                              `json:"vcs_user,omitempty" yaml:"vcs_user,omitempty"`
	VCSPassword          string                              `json:"vcs_password,omitempty" yaml:"vcs_password,omitempty"`
	VCSBranch            string                              `json:"vcs_branch,omitempty" yaml:"vcs_branch,omitempty"`
	VCSDefaultBranch     string                              `json:"vcs_default_branch,omitempty" yaml:"vcs_default_branch,omitempty"`
	VCSPGPKey            string                              `json:"vcs_pgp_key,omitempty" yaml:"vcs_pgp_key,omitempty"`
	DeploymentStrategies map[string]map[string]VariableValue `json:"deployments,omitempty" yaml:"deployments,omitempty"`
}

// ApplicationVersion is a version
type ApplicationVersion string

// There are the supported versions
const (
	ApplicationVersion1 = "v1.0"
)

// EncryptedKey represents an encrypted secret
type EncryptedKey struct {
	Type    string
	Name    string
	Content string
}

// NewApplication instanciance an exportable application from an sdk.Application
func NewApplication(app sdk.Application, withPermissions bool, keys []EncryptedKey) (a Application, err error) {
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

	if withPermissions {
		a.Permissions = make(map[string]int, len(app.ApplicationGroups))
		for _, p := range app.ApplicationGroups {
			a.Permissions[p.Group.Name] = p.Permission
		}
	}

	a.Keys = make(map[string]KeyValue, len(keys))
	for _, e := range keys {
		a.Keys[e.Name] = KeyValue{
			Type:  e.Type,
			Value: e.Content,
		}
	}

	a.VCSBranch = app.RepositoryStrategy.Branch
	if app.RepositoryStrategy.ConnectionType != "https" {
		a.VCSConnectionType = app.RepositoryStrategy.ConnectionType
	}
	a.VCSDefaultBranch = app.RepositoryStrategy.DefaultBranch
	a.VCSPGPKey = app.RepositoryStrategy.PGPKey
	a.VCSSSHKey = app.RepositoryStrategy.SSHKey
	a.VCSUser = app.RepositoryStrategy.User
	a.VCSPassword = app.RepositoryStrategy.Password

	a.DeploymentStrategies = make(map[string]map[string]VariableValue, len(app.DeploymentStrategies))
	for name, config := range app.DeploymentStrategies {
		vars := make(map[string]VariableValue, len(config))
		for k, v := range config {
			vars[k] = VariableValue{
				Type:  v.Type,
				Value: v.Value,
			}
		}
		a.DeploymentStrategies[name] = vars
	}

	return a, nil
}
