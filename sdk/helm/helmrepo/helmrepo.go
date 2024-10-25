/*
Copyright The Helm-push Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package helmrepo

import (
	"fmt"
	"os"

	"helm.sh/helm/v3/pkg/helmpath"
	"helm.sh/helm/v3/pkg/repo"
)

type (
	// Repo represents a collection of parameters for chart repository
	Repo struct {
		*repo.Entry
	}
)

// GetRepoByName returns repository by name
func GetRepoByName(name string) (*Repo, error) {
	r, err := repoFile()
	if err != nil {
		return nil, err
	}
	entry, exists := findRepoEntry(name, r)
	if !exists {
		return nil, fmt.Errorf("no repo named %q found", name)
	}
	return &Repo{entry}, nil
}

func repoFile() (*repo.File, error) {
	repositoryFile := getRepositoryFile()
	return repo.LoadFile(repositoryFile)
}

func getRepositoryFile() string {
	var helmRepoFilePath string
	if v, ok := os.LookupEnv("HELM_REPOSITORY_CONFIG"); ok {
		helmRepoFilePath = v
	} else {
		helmRepoFilePath = helmpath.ConfigPath("repositories.yaml")
	}
	return helmRepoFilePath
}

func findRepoEntry(name string, r *repo.File) (*repo.Entry, bool) {
	var entry *repo.Entry
	exists := false
	for _, re := range r.Repositories {
		if re.Name == name {
			entry = re
			exists = true
			break
		}
	}
	return entry, exists
}
