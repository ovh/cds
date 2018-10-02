package workflowtemplate_test

import (
	"fmt"
	"testing"

	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/stretchr/testify/assert"
)

func TestExecuteTemplate(t *testing.T) {
	tmpl := workflowtemplate.GetAll()

	req := workflowtemplate.Request{
		Name: "my-workflow",
		Parameters: map[string]string{
			"withDeploy": "true",
			"deployWhen": "failure",
		},
	}

	res, err := tmpl[0].Execute(req)
	assert.Nil(t, err)

	fmt.Println(res.Workflow)
	for _, p := range res.Pipelines {
		fmt.Println(p)
	}

	assert.Equal(t, true, true)
}
