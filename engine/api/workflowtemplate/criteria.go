package workflowtemplate

import (
	"github.com/ovh/cds/engine/api/database"
)

func NewCriteria() Criteria { return Criteria{} }

type Criteria struct {
	ids, groupIDs []int64
}

func (c Criteria) IDs(ids ...int64) Criteria {
	c.ids = ids
	return c
}

func (c Criteria) GroupIDs(ids ...int64) Criteria {
	c.groupIDs = ids
	return c
}

func (c Criteria) where() string {
	var reqs []string

	if c.ids != nil {
		reqs = append(reqs, "id = ANY(string_to_array(:ids, ',')::int[])")
	}
	if c.groupIDs != nil {
		reqs = append(reqs, "group_id = ANY(string_to_array(:groupIDs, ',')::int[])")
	}

	if len(reqs) == 0 {
		return "false"
	}

	return database.And(reqs...)
}

func (c Criteria) args() interface{} {
	return map[string]interface{}{
		"ids":      database.IDsToQueryString(c.ids),
		"groupIDs": database.IDsToQueryString(c.groupIDs),
	}
}

func NewCriteriaInstance() CriteriaInstance { return CriteriaInstance{} }

type CriteriaInstance struct {
	workflowTemplateIDs []int64
	workflowIDs         []int64
}

func (c CriteriaInstance) WorkflowTemplateIDs(ids ...int64) CriteriaInstance {
	c.workflowTemplateIDs = ids
	return c
}

func (c CriteriaInstance) WorkflowIDs(ids ...int64) CriteriaInstance {
	c.workflowIDs = ids
	return c
}

func (c CriteriaInstance) where() string {
	var reqs []string

	if c.workflowTemplateIDs != nil {
		reqs = append(reqs, "workflow_template_id = ANY(string_to_array(:workflowTemplateIDs, ',')::int[])")
	}

	if c.workflowIDs != nil {
		reqs = append(reqs, "workflow_id = ANY(string_to_array(:workflowIDs, ',')::int[])")
	}

	if len(reqs) == 0 {
		return "false"
	}

	return database.And(reqs...)
}

func (c CriteriaInstance) args() interface{} {
	return map[string]interface{}{
		"workflowTemplateIDs": database.IDsToQueryString(c.workflowTemplateIDs),
		"workflowIDs":         database.IDsToQueryString(c.workflowIDs),
	}
}
