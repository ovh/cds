package workflowtemplate

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestAggregateOnWorkflowTemplate(t *testing.T) {
	db := &test.SqlExecutorMock{}
	db.OnSelect = func(i interface{}) {
		gs := i.(*[]sdk.Group)
		*gs = append(*gs, sdk.Group{
			ID:   1,
			Name: "grp-1",
		}, sdk.Group{
			ID:   2,
			Name: "grp-2",
		})
	}

	wts := []*sdk.WorkflowTemplate{
		{GroupID: 1},
		{GroupID: 2},
	}

	assert.Nil(t, AggregateOnWorkflowTemplate(db, wts...))

	assert.NotNil(t, wts[0].Group)
	assert.Equal(t, "grp-1", wts[0].Group.Name)
	assert.NotNil(t, wts[1].Group)
	assert.Equal(t, "grp-2", wts[1].Group.Name)
}
