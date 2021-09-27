package workflowv3

import (
	"fmt"
	"strings"
)

type Deployments map[string]Deployment

func (d Deployments) ExistDeployment(deploymentName string) bool {
	_, ok := d[deploymentName]
	return ok
}

type Deployment struct {
	Integration string           `json:"integration,omitempty" yaml:"integration,omitempty"` // should be @
	Config      DeploymentConfig `json:"config,omitempty" yaml:"config,omitempty"`
}

func (d Deployment) Validate() (ExternalDependencies, error) {
	var extDep ExternalDependencies

	integration := strings.TrimPrefix(d.Integration, "@")
	if integration == d.Integration {
		return extDep, fmt.Errorf("integration should be external")
	}
	extDep.Integrations = append(extDep.Integrations, integration)

	return extDep, nil
}

type DeploymentConfig map[string]DeploymentConfigValue

type DeploymentConfigValue struct {
	Type  string `json:"type,omitempty" yaml:"type,omitempty"`
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
}
