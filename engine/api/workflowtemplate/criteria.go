package workflowtemplate

import (
	"strings"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
)

func NewCriteria() Criteria { return Criteria{} }

type Criteria struct {
	ids, groupIDs []int64
	slugs         []string
}

func (c Criteria) IDs(ids ...int64) Criteria {
	c.ids = ids
	return c
}

func (c Criteria) GroupIDs(ids ...int64) Criteria {
	c.groupIDs = ids
	return c
}

func (c Criteria) Slugs(ss ...string) Criteria {
	c.slugs = ss
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

	if c.slugs != nil {
		reqs = append(reqs, "slug = ANY(string_to_array(:slugs, ',')::text[])")
	}

	if len(reqs) == 0 {
		return "false"
	}

	return gorpmapping.And(reqs...)
}

func (c Criteria) args() interface{} {
	return map[string]interface{}{
		"ids":      gorpmapping.IDsToQueryString(c.ids),
		"groupIDs": gorpmapping.IDsToQueryString(c.groupIDs),
		"slugs":    strings.Join(c.slugs, ","),
	}
}

func NewCriteriaInstance() CriteriaInstance { return CriteriaInstance{} }

type CriteriaInstance struct {
	workflowTemplateIDs []int64
	workflowIDs         []int64
	projectIDs          []int64
}

func (c CriteriaInstance) WorkflowTemplateIDs(ids ...int64) CriteriaInstance {
	c.workflowTemplateIDs = ids
	return c
}

func (c CriteriaInstance) WorkflowIDs(ids ...int64) CriteriaInstance {
	c.workflowIDs = ids
	return c
}

func (c CriteriaInstance) ProjectIDs(ids ...int64) CriteriaInstance {
	c.projectIDs = ids
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

	if c.projectIDs != nil {
		reqs = append(reqs, "project_id = ANY(string_to_array(:projectIDs, ',')::int[])")
	}

	if len(reqs) == 0 {
		return "false"
	}

	return gorpmapping.And(reqs...)
}

func (c CriteriaInstance) args() interface{} {
	return map[string]interface{}{
		"workflowTemplateIDs": gorpmapping.IDsToQueryString(c.workflowTemplateIDs),
		"workflowIDs":         gorpmapping.IDsToQueryString(c.workflowIDs),
		"projectIDs":          gorpmapping.IDsToQueryString(c.projectIDs),
	}
}

func NewCriteriaAudit() CriteriaAudit { return CriteriaAudit{} }

type CriteriaAudit struct {
	workflowTemplateIDs []int64
	eventTypes          []string
}

func (c CriteriaAudit) EventTypes(ets ...string) CriteriaAudit {
	c.eventTypes = ets
	return c
}

func (c CriteriaAudit) WorkflowTemplateIDs(ids ...int64) CriteriaAudit {
	c.workflowTemplateIDs = ids
	return c
}

func (c CriteriaAudit) where() string {
	var reqs []string

	if c.eventTypes != nil {
		reqs = append(reqs, "event_type = ANY(string_to_array(:eventTypes, ',')::text[])")
	}

	if c.workflowTemplateIDs != nil {
		reqs = append(reqs, "workflow_template_id = ANY(string_to_array(:workflowTemplateIDs, ',')::int[])")
	}

	if len(reqs) == 0 {
		return "false"
	}

	return gorpmapping.And(reqs...)
}

func (c CriteriaAudit) args() interface{} {
	return map[string]interface{}{
		"eventTypes":          strings.Join(c.eventTypes, ","),
		"workflowTemplateIDs": gorpmapping.IDsToQueryString(c.workflowTemplateIDs),
	}
}

func NewCriteriaInstanceAudit() CriteriaInstanceAudit { return CriteriaInstanceAudit{} }

type CriteriaInstanceAudit struct {
	workflowTemplateInstanceIDs []int64
	eventTypes                  []string
}

func (c CriteriaInstanceAudit) EventTypes(ets ...string) CriteriaInstanceAudit {
	c.eventTypes = ets
	return c
}

func (c CriteriaInstanceAudit) WorkflowTemplateInstanceIDs(ids ...int64) CriteriaInstanceAudit {
	c.workflowTemplateInstanceIDs = ids
	return c
}

func (c CriteriaInstanceAudit) where() string {
	var reqs []string

	if c.eventTypes != nil {
		reqs = append(reqs, "event_type = ANY(string_to_array(:eventTypes, ',')::text[])")
	}

	if c.workflowTemplateInstanceIDs != nil {
		reqs = append(reqs, "workflow_template_instance_id = ANY(string_to_array(:workflowTemplateInstanceIDs, ',')::int[])")
	}

	if len(reqs) == 0 {
		return "false"
	}

	return gorpmapping.And(reqs...)
}

func (c CriteriaInstanceAudit) args() interface{} {
	return map[string]interface{}{
		"eventTypes":                  strings.Join(c.eventTypes, ","),
		"workflowTemplateInstanceIDs": gorpmapping.IDsToQueryString(c.workflowTemplateInstanceIDs),
	}
}
