package workflowtemplate

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestAggregateTemplateInstanceOnWorkflow(t *testing.T) {
	db := &test.SqlExecutorMock{}

	ids := []int64{4, 5, 6}
	db.OnSelect = func(i interface{}) {
		fmt.Println(reflect.TypeOf(i))
		if wtis, ok := i.(*[]*sdk.WorkflowTemplateInstance); ok {
			*wtis = append(*wtis,
				&sdk.WorkflowTemplateInstance{
					WorkflowTemplateID: 1,
					WorkflowID:         &ids[0],
				},
				&sdk.WorkflowTemplateInstance{
					WorkflowTemplateID: 1,
					WorkflowID:         &ids[1],
				},
				&sdk.WorkflowTemplateInstance{
					WorkflowTemplateID: 2,
					WorkflowID:         &ids[2],
				})
		}
	}

	ws := []*sdk.Workflow{{ID: 4}, {ID: 5}, {ID: 6}}

	assert.Nil(t, AggregateTemplateInstanceOnWorkflow(context.TODO(), db, ws...))

	require.NotNil(t, ws[0].TemplateInstance)
	assert.Equal(t, int64(1), ws[0].TemplateInstance.WorkflowTemplateID)

	require.NotNil(t, ws[1].TemplateInstance)
	assert.Equal(t, int64(1), ws[1].TemplateInstance.WorkflowTemplateID)

	require.NotNil(t, ws[2].TemplateInstance)
	assert.Equal(t, int64(2), ws[2].TemplateInstance.WorkflowTemplateID)
}
