package workflow_test

import (
	"context"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/stretchr/testify/assert"
)

func TestLoadAllWorkflows(t *testing.T) {
	db, _, end := test.SetupPG(t)
	defer end()

	var opts workflow.LoadAllWorkflowsOptions

	_, err := workflow.LoadAllWorkflows(context.TODO(), db, opts)
	assert.NoError(t, err)
}
