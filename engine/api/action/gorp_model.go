package action

import (
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

type actionParameter struct {
	ID          int64  `json:"id" yaml:"-" db:"id"`
	ActionID    int64  `json:"action_id" yaml:"-" db:"action_id"`
	Name        string `json:"name" db:"name"`
	Type        string `json:"type" db:"type"`
	Value       string `json:"value" db:"value"`
	Description string `json:"description,omitempty" yaml:"desc,omitempty" db:"description"`
	Advanced    bool   `json:"advanced,omitempty" yaml:"advanced,omitempty" db:"advanced"`
}

func actionParametersToParameters(aps []actionParameter) []sdk.Parameter {
	ps := make([]sdk.Parameter, len(aps))
	for i := range aps {
		ps[i] = sdk.Parameter{
			ID:          aps[i].ID,
			Name:        aps[i].Name,
			Type:        aps[i].Type,
			Value:       aps[i].Value,
			Description: aps[i].Description,
			Advanced:    aps[i].Advanced,
		}
	}
	return ps
}

type actionEdge struct {
	ID             int64  `db:"id"`
	ParentID       int64  `db:"parent_id"`
	ChildID        int64  `db:"child_id"`
	ExecOrder      int64  `db:"exec_order"`
	Enabled        bool   `db:"enabled"`
	Optional       bool   `db:"optional"`
	AlwaysExecuted bool   `db:"always_executed"`
	StepName       string `db:"step_name"`
	// aggregates
	Parameters []actionEdgeParameter `db:"-"`
	Child      *sdk.Action           `db:"-"`
}

func actionEdgesToIDs(aes []*actionEdge) []int64 {
	ids := make([]int64, len(aes))
	for i := range aes {
		ids[i] = aes[i].ID
	}
	return ids
}

func actionEdgesToChildIDs(aes []*actionEdge) []int64 {
	ids := make([]int64, len(aes))
	for i := range aes {
		ids[i] = aes[i].ChildID
	}
	return ids
}

type actionEdgeParameter struct {
	ID           int64  `json:"id" yaml:"-" db:"id"`
	ActionEdgeID int64  `json:"action_id" yaml:"-" db:"action_edge_id"`
	Name         string `json:"name" db:"name"`
	Type         string `json:"type" db:"type"`
	Value        string `json:"value" db:"value"`
	Description  string `json:"description,omitempty" yaml:"desc,omitempty" db:"description"`
	Advanced     bool   `json:"advanced,omitempty" yaml:"advanced,omitempty" db:"advanced"`
}

func init() {
	gorpmapping.Register(
		gorpmapping.New(sdk.Action{}, "action", true, "id"),
		gorpmapping.New(sdk.AuditAction{}, "action_audit", true, "id"),
		gorpmapping.New(actionParameter{}, "action_parameter", true, "id"),
		gorpmapping.New(sdk.Requirement{}, "action_requirement", true, "id"),
		gorpmapping.New(actionEdge{}, "action_edge", true, "id"),
		gorpmapping.New(actionEdgeParameter{}, "action_edge_parameter", true, "id"),
	)
}
