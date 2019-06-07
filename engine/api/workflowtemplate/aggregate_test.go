package workflowtemplate

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

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

	assert.Nil(t, AggregateTemplateInstanceOnWorkflow(context.TODO(), db, ws...))

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
