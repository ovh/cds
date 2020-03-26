package workflowtemplate_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/sdk"
)

func TestLoadInstanceAudits(t *testing.T) {
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

	assert.Nil(t, workflowtemplate.LoadInstanceOptions.WithAudits(context.TODO(), db, wtis...))

	assert.Equal(t, int64(1), wtis[0].FirstAudit.ID)
	assert.Equal(t, int64(3), wtis[0].LastAudit.ID)
	assert.Equal(t, int64(2), wtis[1].FirstAudit.ID)
	assert.Equal(t, int64(4), wtis[1].LastAudit.ID)
}
