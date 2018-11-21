package workflowtemplate

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestAggregateAuditsOnWorkflowTemplate(t *testing.T) {
	db := &test.SqlExecutorMock{}
	db.OnSelect = func(i interface{}) {
		awts := i.(*[]sdk.AuditWorkflowTemplate)
		*awts = append(*awts, sdk.AuditWorkflowTemplate{
			AuditCommon: sdk.AuditCommon{
				ID:        1,
				EventType: "EventWorkflowTemplateAdd",
			},
			WorkflowTemplateID: 1,
		}, sdk.AuditWorkflowTemplate{
			AuditCommon: sdk.AuditCommon{
				ID:        2,
				EventType: "EventWorkflowTemplateAdd",
			},
			WorkflowTemplateID: 2,
		}, sdk.AuditWorkflowTemplate{
			AuditCommon: sdk.AuditCommon{
				ID:        3,
				EventType: "EventWorkflowTemplateUpdate",
			},
			WorkflowTemplateID: 1,
		}, sdk.AuditWorkflowTemplate{
			AuditCommon: sdk.AuditCommon{
				ID:        4,
				EventType: "EventWorkflowTemplateUpdate",
			},
			WorkflowTemplateID: 2,
		})
	}

	wts := []*sdk.WorkflowTemplate{
		{ID: 1},
		{ID: 2},
	}

	assert.Nil(t, AggregateAuditsOnWorkflowTemplate(db, wts...))

	assert.Equal(t, int64(1), wts[0].FirstAudit.ID)
	assert.Equal(t, int64(3), wts[0].LastAudit.ID)
	assert.Equal(t, int64(2), wts[1].FirstAudit.ID)
	assert.Equal(t, int64(4), wts[1].LastAudit.ID)
}

func TestAggregateAuditsOnWorkflowTemplateInstance(t *testing.T) {
	db := &test.SqlExecutorMock{}
	db.OnSelect = func(i interface{}) {
		awtis := i.(*[]sdk.AuditWorkflowTemplateInstance)
		*awtis = append(*awtis, sdk.AuditWorkflowTemplateInstance{
			AuditCommon: sdk.AuditCommon{
				ID:        1,
				EventType: "EventWorkflowTemplateInstanceAdd",
			},
			WorkflowTemplateInstanceID: 1,
		}, sdk.AuditWorkflowTemplateInstance{
			AuditCommon: sdk.AuditCommon{
				ID:        2,
				EventType: "EventWorkflowTemplateInstanceAdd",
			},
			WorkflowTemplateInstanceID: 2,
		}, sdk.AuditWorkflowTemplateInstance{
			AuditCommon: sdk.AuditCommon{
				ID:        3,
				EventType: "EventWorkflowTemplateInstanceUpdate",
			},
			WorkflowTemplateInstanceID: 1,
		}, sdk.AuditWorkflowTemplateInstance{
			AuditCommon: sdk.AuditCommon{
				ID:        4,
				EventType: "EventWorkflowTemplateInstanceUpdate",
			},
			WorkflowTemplateInstanceID: 2,
		})
	}

	wtis := []*sdk.WorkflowTemplateInstance{
		{ID: 1},
		{ID: 2},
	}

	assert.Nil(t, AggregateAuditsOnWorkflowTemplateInstance(db, wtis...))

	assert.Equal(t, int64(1), wtis[0].FirstAudit.ID)
	assert.Equal(t, int64(3), wtis[0].LastAudit.ID)
	assert.Equal(t, int64(2), wtis[1].FirstAudit.ID)
	assert.Equal(t, int64(4), wtis[1].LastAudit.ID)
}

func TestAggregateWorkflowTemplateInstance(t *testing.T) {
	db := &test.SqlExecutorMock{}

	ids := []int64{4, 5, 6}
	db.OnSelect = func(i interface{}) {
		if wtis, ok := i.(*[]sdk.WorkflowTemplateInstance); ok {
			*wtis = append(*wtis, sdk.WorkflowTemplateInstance{},
				sdk.WorkflowTemplateInstance{
					WorkflowTemplateID: 1,
					WorkflowID:         &ids[0],
				},
				sdk.WorkflowTemplateInstance{
					WorkflowTemplateID: 1,
					WorkflowID:         &ids[1],
				},
				sdk.WorkflowTemplateInstance{
					WorkflowTemplateID: 2,
					WorkflowID:         &ids[2],
				})
		}
		if wts, ok := i.(*[]sdk.WorkflowTemplate); ok {
			*wts = append(*wts, sdk.WorkflowTemplate{},
				sdk.WorkflowTemplate{
					ID:    1,
					Slug:  "one",
					Group: &sdk.Group{Name: "one"},
				},
				sdk.WorkflowTemplate{
					ID:    2,
					Slug:  "two",
					Group: &sdk.Group{Name: "two"},
				})
		}
	}

	ws := []*sdk.Workflow{{ID: 4}, {ID: 5}, {ID: 6}}

	assert.Nil(t, AggregateTemplateOnWorkflow(db, ws...))

	if !assert.NotNil(t, ws[0].Template) || !assert.NotNil(t, ws[0].Template.Group) {
		t.FailNow()
	}
	assert.Equal(t, "one/one", fmt.Sprintf("%s/%s", ws[0].Template.Group.Name, ws[0].Template.Slug))

	if !assert.NotNil(t, ws[1].Template) || !assert.NotNil(t, ws[1].Template.Group) {
		t.FailNow()
	}
	assert.Equal(t, "one/one", fmt.Sprintf("%s/%s", ws[1].Template.Group.Name, ws[1].Template.Slug))

	if !assert.NotNil(t, ws[2].Template) && !assert.NotNil(t, ws[2].Template.Group) {
		t.FailNow()
	}
	assert.Equal(t, "two/two", fmt.Sprintf("%s/%s", ws[2].Template.Group.Name, ws[2].Template.Slug))
}
