package workflowtemplate_test

import (
	"fmt"
	"testing"

	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/stretchr/testify/assert"
)

func TestExecuteTemplate(t *testing.T) {
	tmpl := workflowtemplate.GetAll()

	res, err := tmpl[0].Execute()
	assert.Nil(t, err)

	fmt.Println(res.Workflow)
	for _, p := range res.Pipelines {
		fmt.Println(p)
	}

	assert.Equal(t, true, true)
}
