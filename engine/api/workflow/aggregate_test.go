package workflow

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestAggregateOnWorkflowTemplateInstance(t *testing.T) {
	db := &test.SqlExecutorMock{}
	db.OnSelect = func(i interface{}) {
		gs := i.(*[]sdk.Workflow)
		*gs = append(*gs, sdk.Workflow{
			ID:   1,
			Name: "wkf-1",
		}, sdk.Workflow{
			ID:   2,
			Name: "wkf-2",
		})
	}

	wtis := []*sdk.WorkflowTemplateInstance{
		{WorkflowID: 1},
		{WorkflowID: 2},
	}

	assert.Nil(t, AggregateOnWorkflowTemplateInstance(db, wtis...))

	assert.NotNil(t, wtis[0].Workflow)
	assert.Equal(t, "wkf-1", wtis[0].Workflow.Name)
	assert.NotNil(t, wtis[1].Workflow)
	assert.Equal(t, "wkf-2", wtis[1].Workflow.Name)
}
