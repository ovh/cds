package workflowtemplate

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestAggregateAuditsOnWorkflowTemplate(t *testing.T) {
	db := &test.SqlExecutorMock{}
	db.OnSelect = func(i interface{}) {
		awts := i.(*[]*sdk.AuditWorkflowTemplate)
		*awts = append(*awts, &sdk.AuditWorkflowTemplate{
			AuditCommon: sdk.AuditCommon{
				ID:        1,
				EventType: "EventWorkflowTemplateAdd",
			},
			WorkflowTemplateID: 1,
		}, &sdk.AuditWorkflowTemplate{
			AuditCommon: sdk.AuditCommon{
				ID:        2,
				EventType: "EventWorkflowTemplateAdd",
			},
			WorkflowTemplateID: 2,
		}, &sdk.AuditWorkflowTemplate{
			AuditCommon: sdk.AuditCommon{
				ID:        3,
				EventType: "EventWorkflowTemplateUpdate",
			},
			WorkflowTemplateID: 1,
		}, &sdk.AuditWorkflowTemplate{
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
		awtis := i.(*[]*sdk.AuditWorkflowTemplateInstance)
		*awtis = append(*awtis, &sdk.AuditWorkflowTemplateInstance{
			AuditCommon: sdk.AuditCommon{
				ID:        1,
				EventType: "EventWorkflowTemplateInstanceAdd",
			},
			WorkflowTemplateInstanceID: 1,
		}, &sdk.AuditWorkflowTemplateInstance{
			AuditCommon: sdk.AuditCommon{
				ID:        2,
				EventType: "EventWorkflowTemplateInstanceAdd",
			},
			WorkflowTemplateInstanceID: 2,
		}, &sdk.AuditWorkflowTemplateInstance{
			AuditCommon: sdk.AuditCommon{
				ID:        3,
				EventType: "EventWorkflowTemplateInstanceUpdate",
			},
			WorkflowTemplateInstanceID: 1,
		}, &sdk.AuditWorkflowTemplateInstance{
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
