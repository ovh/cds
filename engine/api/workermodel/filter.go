package workermodel

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
)

// LoadFilter struct for worker model query.
type LoadFilter struct {
	Binary string
	State  StateFilter
}

// SQL returns the raw sql for current filter.
func (l LoadFilter) SQL() string {
	var conds []string

	if l.Binary != "" {
		conds = append(conds, "worker_capability.type = 'binary'")
		conds = append(conds, "worker_capability.argument = :binary")
	}

	switch l.State {
	case StateError:
		conds = append(conds, "worker_model.nb_spawn_err > 0")
	case StateDisabled:
		conds = append(conds, "worker_model.disabled = true")
	case StateRegister:
		conds = append(conds, "worker_model.need_registration = true")
	case StateDeprecated:
		conds = append(conds, "worker_model.is_deprecated = true")
	case StateActive:
		conds = append(conds, "worker_model.is_deprecated = false")
	case StateOfficial:
		conds = append(conds, "worker_model.group_id = :sharedInfraGroupID")
	}

	return gorpmapping.And(conds...)
}

// Args returns sql args for current filter.
func (l LoadFilter) Args() gorpmapping.ArgsMap {
	return gorpmapping.ArgsMap{
		"binary":             l.Binary,
		"sharedInfraGroupID": group.SharedInfraGroup.ID,
	}
}

// StateFilter for worker model.
type StateFilter string

// IsValid returns an error if the state value is not valid.
func (s StateFilter) IsValid() error {
	switch s {
	case StateDisabled, StateOfficial, StateError, StateRegister, StateDeprecated, StateActive:
		return nil
	default:
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given state filter")
	}
}

// List of const for state filter.
const (
	StateError      StateFilter = "error"
	StateDisabled   StateFilter = "disabled"
	StateRegister   StateFilter = "register"
	StateDeprecated StateFilter = "deprecated"
	StateActive     StateFilter = "active"
	StateOfficial   StateFilter = "official"
)
