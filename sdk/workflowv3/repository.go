package workflowv3

import (
	"fmt"
	"strings"
)

type Repositories map[string]Repository

func (r Repositories) ExistRepo(repoName string) bool {
	_, ok := r[repoName]
	return ok
}

type Repository struct {
	Slug       string `json:"slug,omitempty" yaml:"slug,omitempty"`
	Server     string `json:"server,omitempty" yaml:"server,omitempty"` // should be @
	Connection string `json:"connection,omitempty" yaml:"connection,omitempty"`
	SSHKey     string `json:"ssh_key,omitempty" yaml:"ssh_key,omitempty"` // can be @
	PGPKey     string `json:"pgp_key,omitempty" yaml:"pgp_key,omitempty"` // can be @
}

func (r Repository) Validate(w Workflow) (ExternalDependencies, error) {
	var extDep ExternalDependencies

	server := strings.TrimPrefix(r.Server, "@")
	if server == r.Server {
		return extDep, fmt.Errorf("vcs server should be external")
	}
	extDep.VCSServers = append(extDep.VCSServers, server)

	if r.SSHKey != "" {
		key := strings.TrimPrefix(r.SSHKey, "@")
		isExternal := key != r.SSHKey
		if isExternal {
			extDep.SSHKeys = append(extDep.SSHKeys, key)
		} else {
			if !w.Keys.ExistKey(key, SSHKeyType) {
				return extDep, fmt.Errorf("unknown ssh key %q", key)
			}
		}
	}
	if r.PGPKey != "" {
		key := strings.TrimPrefix(r.PGPKey, "@")
		isExternal := key != r.PGPKey
		if isExternal {
			extDep.PGPKeys = append(extDep.PGPKeys, key)
		} else {
			if !w.Keys.ExistKey(key, PGPKeyType) {
				return extDep, fmt.Errorf("unknown pgp key %q", key)
			}
		}
	}

	return extDep, nil
}
