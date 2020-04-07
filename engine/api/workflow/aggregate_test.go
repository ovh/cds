package workflow_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func TestAggregateOnWorkflowTemplateInstance(t *testing.T) {
	db := &test.SqlExecutorMock{}
	db.OnSelect = func(i interface{}) {
		gs := i.(*[]workflow.Workflow)
		*gs = append(*gs, workflow.Workflow{
			ID:   1,
			Name: "wkf-1",
		}, workflow.Workflow{
			ID:   2,
			Name: "wkf-2",
		})
	}

	ids := []int64{1, 2}
	wtis := []*sdk.WorkflowTemplateInstance{
		{WorkflowID: &ids[0]},
		{WorkflowID: &ids[1]},
	}

	assert.Nil(t, workflow.AggregateOnWorkflowTemplateInstance(context.TODO(), db, wtis...))

	assert.NotNil(t, wtis[0].Workflow)
	assert.Equal(t, "wkf-1", wtis[0].Workflow.Name)
	assert.NotNil(t, wtis[1].Workflow)
	assert.Equal(t, "wkf-2", wtis[1].Workflow.Name)
}
