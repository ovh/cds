package sdk

import (
	"context"
	"testing"

	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/require"
)

func TestDefaultValue(t *testing.T) {
	wk := `name: myworkflow
from: library/myTemplate
parameters:
  keyNormal: myValue
  keyWithDefault: mySurchargedValue
  keyEmptyValue: ""`

	tmpl := `name: myTemplate
parameters:
- key: keyNormal
- key: keyWithDefault
  default: myDefaultValue
- key: keyEmptyValue
  default: not
- key: keyGoodDefault
  default: mySuperDefaultValue
- key: noDefault  
spec: |-
  on:
    push: {}
  jobs:
    [[- if eq .params.keyNormal "myValue" ]]
    normal: 
      runs-on: mymodel
    [[- end ]]	
    [[- if eq .params.keyWithDefault "mySurchargedValue" ]]
    withDefault: 
      runs-on: mymodel
    [[- end ]]	
    [[- if eq .params.keyEmptyValue "" ]]
    emptyValue: 
      runs-on: mymodel
    [[- end]]
    [[- if eq .params.keyGoodDefault "mySuperDefaultValue" ]]
    goodDefault: 
      runs-on: mymodel
    [[- end]]
    [[- if .params.noDefault ]]
    noDefault: 
      runs-on: mymodel
    [[- end]]`

	var work V2Workflow
	require.NoError(t, yaml.Unmarshal([]byte(wk), &work))

	var template V2WorkflowTemplate
	require.NoError(t, yaml.Unmarshal([]byte(tmpl), &template))

	yamlWorkflow, err := template.Resolve(context.TODO(), &work)
	require.NoError(t, err)

	var resolvedWorkflow V2Workflow
	require.NoError(t, yaml.Unmarshal([]byte(yamlWorkflow), &resolvedWorkflow))

	normal := work.Jobs["normal"]
	withDefault := work.Jobs["withDefault"]
	emptyValue := work.Jobs["emptyValue"]
	goodDefault := work.Jobs["goodDefault"]
	_, has := work.Jobs["noDefault"]

	require.Equal(t, "mymodel", normal.RunsOn.Model)
	require.Equal(t, "mymodel", withDefault.RunsOn.Model)
	require.Equal(t, "mymodel", emptyValue.RunsOn.Model)
	require.Equal(t, "mymodel", goodDefault.RunsOn.Model)
	require.False(t, has)

	require.Equal(t, 4, len(work.Jobs))
}
func TestOverrideWorkflowOnEmpty(t *testing.T) {
	wk := `name: myworkflow
from: library/myTemplate
`

	tmpl := `name: myTemplate
parameters:
- key: it_env
  type: json
spec: |-
  on:
    push: {}`

	var work V2Workflow
	require.NoError(t, yaml.Unmarshal([]byte(wk), &work))

	var template V2WorkflowTemplate
	require.NoError(t, yaml.Unmarshal([]byte(tmpl), &template))

	yamlWorkflow, err := template.Resolve(context.TODO(), &work)
	require.NoError(t, err)

	var resolvedWorkflow V2Workflow
	require.NoError(t, yaml.Unmarshal([]byte(yamlWorkflow), &resolvedWorkflow))

	require.NotNil(t, work.On)
	require.Nil(t, work.On.PullRequest)
	require.NotNil(t, work.On.Push)
	require.Equal(t, 0, len(work.On.Push.Branches))
}

func TestOverrideWorkflowOn(t *testing.T) {
	wk := `name: myworkflow
from: library/myTemplate
on: [push]
`

	tmpl := `name: myTemplate
parameters:
- key: it_env
  type: json
spec: |-
  on:
    push:
      branches: [master]
    pull-request:
      type: [opened]  `

	var work V2Workflow
	require.NoError(t, yaml.Unmarshal([]byte(wk), &work))

	var template V2WorkflowTemplate
	require.NoError(t, yaml.Unmarshal([]byte(tmpl), &template))

	yamlWorkflow, err := template.Resolve(context.TODO(), &work)
	require.NoError(t, err)

	var resolvedWorkflow V2Workflow
	require.NoError(t, yaml.Unmarshal([]byte(yamlWorkflow), &resolvedWorkflow))

	require.NotNil(t, work.On)
	require.Nil(t, work.On.PullRequest)
	require.NotNil(t, work.On.Push)
	require.Equal(t, 0, len(work.On.Push.Branches))
}

func TestWorkflowTemplate(t *testing.T) {
	wk := `name: myworkflow
from: library/myTemplate
annotations:
  type: override
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
  on:
    push:
      branches: [master]
  annotations:
    foo: bar
    type: baz
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

	require.Len(t, work.Annotations, 2)

	require.Contains(t, work.Annotations, "type")
	require.Contains(t, work.Annotations, "foo")

	if v, _ := work.Annotations["type"]; v != "override" {
		t.Errorf("annotations 'type' should have value 'override', got %s", v)
	}
	if v, _ := work.Annotations["foo"]; v != "bar" {
		t.Errorf("annotations 'foo' should have value 'bar', got %s", v)
	}

	require.NotNil(t, work.On)
	require.NotNil(t, work.On.Push)
	require.Equal(t, 1, len(work.On.Push.Branches))
	require.Equal(t, "master", work.On.Push.Branches[0])
}
