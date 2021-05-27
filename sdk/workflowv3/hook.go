package workflowv3

import (
	"fmt"
	"strings"
)

type Hook struct {
	Type   string            `json:"type,omitempty" yaml:"type,omitempty"`
	On     string            `json:"on,omitempty" yaml:"on,omitempty"` // can be @
	Config map[string]string `json:"config,omitempty" yaml:"config,omitempty"`
}

func (h Hook) Validate(w Workflow) (ExternalDependencies, error) {
	var extDep ExternalDependencies

	switch h.Type {
	case "repoWebhook":
		targetRepo := strings.TrimPrefix(h.On, "@")
		isExternal := targetRepo != h.On
		if isExternal {
			extDep.Repositories = append(extDep.Repositories, targetRepo)
		} else {
			if !w.Repositories.ExistRepo(targetRepo) {
				return extDep, fmt.Errorf("unknown repository %q", targetRepo)
			}
		}
	case "scheduler":
		return extDep, nil
	default:
		return extDep, fmt.Errorf("invalid hook type %q", h.Type)
	}

	return extDep, nil
}
