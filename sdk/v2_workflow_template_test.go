package sdk

import (
	"context"
	"testing"

	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/require"
)

func TestTempalte(t *testing.T) {
	wk := `name: myworkflow
from: library/myTemplate
parameters:
  it_env: |-
   [{
      "name": "MY_VAR_1",
      "value": "${{vars.myvarset.myvalue}}"
    },{
      "name": "MY_VAR_2",
      "value": "${{vars.myvarset.myvalue2}}"
    }]`

	tmpl := `name: myTemplate
parameters:
- key: it_env
  type: json
spec: |-
  jobs:
   myJob:
      [[- if .params.it_env]]
      env: 
        [[- range .params.it_env]]
        [[.name]]: [[.value]]
        [[- end]]	
      [[- end ]]
      steps:
      - uses: actions/checkout`

	var work V2Workflow
	require.NoError(t, yaml.Unmarshal([]byte(wk), &work))

	var template V2WorkflowTemplate
	require.NoError(t, yaml.Unmarshal([]byte(tmpl), &template))

	yamlWorkflow, err := template.Resolve(context.TODO(), &work)
	require.NoError(t, err)

	var resolvedWorkflow V2Workflow
	require.NoError(t, yaml.Unmarshal([]byte(yamlWorkflow), &resolvedWorkflow))

	require.Equal(t, 2, len(resolvedWorkflow.Jobs["myJob"].Env))

	value1 := resolvedWorkflow.Jobs["myJob"].Env["MY_VAR_1"]
	require.Equal(t, "${{vars.myvarset.myvalue}}", value1)
	value2 := resolvedWorkflow.Jobs["myJob"].Env["MY_VAR_2"]
	require.Equal(t, "${{vars.myvarset.myvalue2}}", value2)
}
