package workermodel

import (
	"fmt"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
)

var defaultEnvs = map[string]string{
	"CDS_SINGLE_USE":          "1",
	"CDS_TTL":                 "{{.TTL}}",
	"CDS_GRAYLOG_HOST":        "{{.GraylogHost}}",
	"CDS_GRAYLOG_PORT":        "{{.GraylogPort}}",
	"CDS_GRAYLOG_EXTRA_KEY":   "{{.GraylogExtraKey}}",
	"CDS_GRAYLOG_EXTRA_VALUE": "{{.GraylogExtraValue}}",
}

func MergeModelEnvsWithDefaultEnvs(envs map[string]string) map[string]string {
	if envs == nil {
		return defaultEnvs
	}
	for envName := range defaultEnvs {
		if _, ok := envs[envName]; !ok {
			envs[envName] = defaultEnvs[envName]
		}
	}

	return envs
}

// StateLoadOption represent load options to load worker model
type StateLoadOption string

func (s StateLoadOption) String() string {
	return string(s)
}

// IsValid returns an error if the state value is not valid.
func (s StateLoadOption) IsValid() error {
	switch s {
	case StateDisabled, StateOfficial, StateError, StateRegister, StateDeprecated, StateActive:
		return nil
	default:
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given state value")
	}
}

// List of const for state load option
const (
	StateError      StateLoadOption = "error"
	StateDisabled   StateLoadOption = "disabled"
	StateRegister   StateLoadOption = "register"
	StateDeprecated StateLoadOption = "deprecated"
	StateActive     StateLoadOption = "active"
	StateOfficial   StateLoadOption = "official"
)

func getAdditionalSQLFilters(opts *StateLoadOption) []string {
	var additionalFilters []string
	if opts != nil {
		switch {
		case *opts == StateError:
			additionalFilters = append(additionalFilters, "worker_model.nb_spawn_err > 0")
		case *opts == StateDisabled:
			additionalFilters = append(additionalFilters, "worker_model.disabled = true")
		case *opts == StateRegister:
			additionalFilters = append(additionalFilters, "worker_model.need_registration = true")
		case *opts == StateDeprecated:
			additionalFilters = append(additionalFilters, "worker_model.is_deprecated = true")
		case *opts == StateActive:
			additionalFilters = append(additionalFilters, "worker_model.is_deprecated = false")
		case *opts == StateOfficial:
			additionalFilters = append(additionalFilters, fmt.Sprintf("worker_model.group_id = %d", group.SharedInfraGroup.ID))
		}
	}
	return additionalFilters
}

// Constant for worker model.
const (
	CacheTTLInSeconds = 30
)
