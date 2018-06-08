package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkflowRunTag(t *testing.T) {
	wfr := WorkflowRun{}

	assert.True(t, wfr.Tag("environment", "preproduction"))
	assert.False(t, wfr.Tag("environment", "production"))
	assert.Equal(t, 1, len(wfr.Tags))
	assert.Equal(t, "environment", wfr.Tags[0].Tag)
	assert.Equal(t, "preproduction,production", wfr.Tags[0].Value)
}
