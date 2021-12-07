package workflowv3

import (
	"fmt"
	"strings"
)

type ArtifactManager string

func (a ArtifactManager) Validate() (ExternalDependencies, error) {
	var extDep ExternalDependencies

	if a == "" {
		return extDep, nil
	}

	integration := strings.TrimPrefix(string(a), "@")
	if integration == string(a) {
		return extDep, fmt.Errorf("should be external")
	}
	extDep.Integrations = append(extDep.Integrations, integration)

	return extDep, nil
}
