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
				ID:        4,
				EventType: "EventWorkflowTemplateUpdate",
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
				ID:        2,
				EventType: "EventWorkflowTemplateAdd",
			},
			WorkflowTemplateID: 2,
		}, sdk.AuditWorkflowTemplate{
			AuditCommon: sdk.AuditCommon{
				ID:        1,
				EventType: "EventWorkflowTemplateAdd",
			},
			WorkflowTemplateID: 1,
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

func TestAggregateTemplateInstanceOnWorkflow(t *testing.T) {
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
	}

	ws := []*sdk.Workflow{{ID: 4}, {ID: 5}, {ID: 6}}

	assert.Nil(t, AggregateTemplateInstanceOnWorkflow(db, ws...))

	if !assert.NotNil(t, ws[0].TemplateInstance) {
		t.FailNow()
	}
	assert.Equal(t, int64(1), ws[0].TemplateInstance.WorkflowTemplateID)

	if !assert.NotNil(t, ws[1].TemplateInstance) {
		t.FailNow()
	}
	assert.Equal(t, int64(1), ws[1].TemplateInstance.WorkflowTemplateID)

	if !assert.NotNil(t, ws[2].TemplateInstance) {
		t.FailNow()
	}
	assert.Equal(t, int64(2), ws[2].TemplateInstance.WorkflowTemplateID)
}

func TestAggregateAggregateTemplateOnInstance(t *testing.T) {
	db := &test.SqlExecutorMock{}

	db.OnSelect = func(i interface{}) {
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

	wtis := []*sdk.WorkflowTemplateInstance{
		{WorkflowTemplateID: 1},
		{WorkflowTemplateID: 1},
		{WorkflowTemplateID: 2},
	}

	assert.Nil(t, AggregateTemplateOnInstance(db, wtis...))

	if !assert.NotNil(t, wtis[0].Template) || !assert.NotNil(t, wtis[0].Template.Group) {
		t.FailNow()
	}
	assert.Equal(t, "one/one", fmt.Sprintf("%s/%s", wtis[0].Template.Group.Name, wtis[0].Template.Slug))

	if !assert.NotNil(t, wtis[1].Template) || !assert.NotNil(t, wtis[1].Template.Group) {
		t.FailNow()
	}
	assert.Equal(t, "one/one", fmt.Sprintf("%s/%s", wtis[1].Template.Group.Name, wtis[1].Template.Slug))

	if !assert.NotNil(t, wtis[2].Template) || !assert.NotNil(t, wtis[2].Template.Group) {
		t.FailNow()
	}
	assert.Equal(t, "two/two", fmt.Sprintf("%s/%s", wtis[2].Template.Group.Name, wtis[2].Template.Slug))
}
