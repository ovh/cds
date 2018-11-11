package workflowtemplate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCriteria(t *testing.T) {
	tests := []struct {
		criteria Criteria
		where    string
		args     interface{}
	}{
		{
			criteria: NewCriteria(),
			where:    "false",
			args: map[string]interface{}{
				"ids":      "",
				"groupIDs": "",
				"slugs":    "",
			},
		},
		{
			criteria: NewCriteria().IDs([]int64{}...).GroupIDs([]int64{}...).Slugs([]string{}...),
			where:    "(id = ANY(string_to_array(:ids, ',')::int[]) AND group_id = ANY(string_to_array(:groupIDs, ',')::int[]) AND slug = ANY(string_to_array(:slugs, ',')::text[]))",
			args: map[string]interface{}{
				"ids":      "",
				"groupIDs": "",
				"slugs":    "",
			},
		},
		{
			criteria: NewCriteria().IDs(1, 2).GroupIDs(3, 4).Slugs("four", "five"),
			where:    "(id = ANY(string_to_array(:ids, ',')::int[]) AND group_id = ANY(string_to_array(:groupIDs, ',')::int[]) AND slug = ANY(string_to_array(:slugs, ',')::text[]))",
			args: map[string]interface{}{
				"ids":      "1,2",
				"groupIDs": "3,4",
				"slugs":    "four,five",
			},
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.where, test.criteria.where())
		assert.Equal(t, test.args, test.criteria.args())
	}
}

func TestCriteriaInstance(t *testing.T) {
	tests := []struct {
		criteria CriteriaInstance
		where    string
		args     interface{}
	}{
		{
			criteria: NewCriteriaInstance(),
			where:    "false",
			args: map[string]interface{}{
				"workflowTemplateIDs": "",
				"workflowIDs":         "",
				"projectIDs":          "",
			},
		},
		{
			criteria: NewCriteriaInstance().WorkflowTemplateIDs([]int64{}...).WorkflowIDs([]int64{}...).ProjectIDs([]int64{}...),
			where:    "(workflow_template_id = ANY(string_to_array(:workflowTemplateIDs, ',')::int[]) AND workflow_id = ANY(string_to_array(:workflowIDs, ',')::int[]) AND project_id = ANY(string_to_array(:projectIDs, ',')::int[]))",
			args: map[string]interface{}{
				"workflowTemplateIDs": "",
				"workflowIDs":         "",
				"projectIDs":          "",
			},
		},
		{
			criteria: NewCriteriaInstance().WorkflowTemplateIDs(1, 2).WorkflowIDs(3, 4).ProjectIDs(5, 6),
			where:    "(workflow_template_id = ANY(string_to_array(:workflowTemplateIDs, ',')::int[]) AND workflow_id = ANY(string_to_array(:workflowIDs, ',')::int[]) AND project_id = ANY(string_to_array(:projectIDs, ',')::int[]))",
			args: map[string]interface{}{
				"workflowTemplateIDs": "1,2",
				"workflowIDs":         "3,4",
				"projectIDs":          "5,6",
			},
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.where, test.criteria.where())
		assert.Equal(t, test.args, test.criteria.args())
	}
}

func TestCriteriaAudit(t *testing.T) {
	tests := []struct {
		criteria CriteriaAudit
		where    string
		args     interface{}
	}{
		{
			criteria: NewCriteriaAudit(),
			where:    "false",
			args: map[string]interface{}{
				"eventTypes":          "",
				"workflowTemplateIDs": "",
			},
		},
		{
			criteria: NewCriteriaAudit().EventTypes([]string{}...).WorkflowTemplateIDs([]int64{}...),
			where:    "(event_type = ANY(string_to_array(:eventTypes, ',')::text[]) AND workflow_template_id = ANY(string_to_array(:workflowTemplateIDs, ',')::int[]))",
			args: map[string]interface{}{
				"eventTypes":          "",
				"workflowTemplateIDs": "",
			},
		},
		{
			criteria: NewCriteriaAudit().EventTypes("one", "two").WorkflowTemplateIDs(3, 4),
			where:    "(event_type = ANY(string_to_array(:eventTypes, ',')::text[]) AND workflow_template_id = ANY(string_to_array(:workflowTemplateIDs, ',')::int[]))",
			args: map[string]interface{}{
				"eventTypes":          "one,two",
				"workflowTemplateIDs": "3,4",
			},
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.where, test.criteria.where())
		assert.Equal(t, test.args, test.criteria.args())
	}
}

func TestCriteriaInstanceAudit(t *testing.T) {
	tests := []struct {
		criteria CriteriaInstanceAudit
		where    string
		args     interface{}
	}{
		{
			criteria: NewCriteriaInstanceAudit(),
			where:    "false",
			args: map[string]interface{}{
				"eventTypes":                  "",
				"workflowTemplateInstanceIDs": "",
			},
		},
		{
			criteria: NewCriteriaInstanceAudit().EventTypes([]string{}...).WorkflowTemplateInstanceIDs([]int64{}...),
			where:    "(event_type = ANY(string_to_array(:eventTypes, ',')::text[]) AND workflow_template_instance_id = ANY(string_to_array(:workflowTemplateInstanceIDs, ',')::int[]))",
			args: map[string]interface{}{
				"eventTypes":                  "",
				"workflowTemplateInstanceIDs": "",
			},
		},
		{
			criteria: NewCriteriaInstanceAudit().EventTypes("one", "two").WorkflowTemplateInstanceIDs(3, 4),
			where:    "(event_type = ANY(string_to_array(:eventTypes, ',')::text[]) AND workflow_template_instance_id = ANY(string_to_array(:workflowTemplateInstanceIDs, ',')::int[]))",
			args: map[string]interface{}{
				"eventTypes":                  "one,two",
				"workflowTemplateInstanceIDs": "3,4",
			},
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.where, test.criteria.where())
		assert.Equal(t, test.args, test.criteria.args())
	}
}
