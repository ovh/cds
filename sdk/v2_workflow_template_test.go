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
- key: keyWithDefaultJson
  type: json
  default: |-
    ["debian", "ubuntu"]
- key: keyEmptyValue
  default: not
- key: keyGoodDefault
  default: mySuperDefaultValue
- key: noDefault  
spec: |-
  semver:
    from: git
    schema:
      "refs/heads/master": ${{ git.version }}-rc-${{cds.run_number}}
      "**/*": ${{ git.version }}-snapshot-${{cds.run_number}} 
  on:
    push: {}
  jobs:
    [[- if eq .params.keyNormal "myValue" ]]
    normal: 
      runs-on: mymodel
      steps:
      - run: |-
          echo "[[.name]]"
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
    [[- end]]
    defaultJson:
      runs-on: mymodel
      strategy:
        matrix:
          os: 
          [[range $i, $a := .params.keyWithDefaultJson ]]
          - [[$a]]
          [[end ]]
      steps:
      - run: echo "${{ matrix.os}}"
`

	var work V2Workflow
	require.NoError(t, yaml.Unmarshal([]byte(wk), &work))

	var template V2WorkflowTemplate
	require.NoError(t, yaml.Unmarshal([]byte(tmpl), &template))

	yamlWorkflow, err := template.Resolve(context.TODO(), &work, nil)
	require.NoError(t, err)

	var resolvedWorkflow V2Workflow
	require.NoError(t, yaml.Unmarshal([]byte(yamlWorkflow), &resolvedWorkflow))

	normal := work.Jobs["normal"]
	withDefault := work.Jobs["withDefault"]
	emptyValue := work.Jobs["emptyValue"]
	goodDefault := work.Jobs["goodDefault"]
	defaultJson := work.Jobs["defaultJson"]
	_, has := work.Jobs["noDefault"]

	require.Equal(t, "mymodel", normal.RunsOn.Model)
	require.Equal(t, "echo \"myworkflow\"", normal.Steps[0].Run)
	require.Equal(t, "mymodel", withDefault.RunsOn.Model)
	require.Equal(t, "mymodel", emptyValue.RunsOn.Model)
	require.Equal(t, "mymodel", goodDefault.RunsOn.Model)
	require.Equal(t, []interface{}{"debian", "ubuntu"}, defaultJson.Strategy.Matrix["os"])
	require.False(t, has)

	require.Equal(t, 5, len(work.Jobs))

	require.NotNil(t, work.Semver)
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

	yamlWorkflow, err := template.Resolve(context.TODO(), &work, nil)
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

	yamlWorkflow, err := template.Resolve(context.TODO(), &work, nil)
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

	yamlWorkflow, err := template.Resolve(context.TODO(), &work, nil)
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

func TestTemplateVariableSetsExists(t *testing.T) {
	vars := NewTemplateVariableSets([]ProjectVariableSet{
		{Name: "vs1", Items: []ProjectVariableSetItem{{Name: "A"}, {Name: "B"}}},
		// EntityNamePattern allows a dash
		{Name: "vs-with-dash", Items: []ProjectVariableSetItem{{Name: "A"}}},
		{Name: "empty-vs"},
	})

	tests := []struct {
		name     string
		vars     TemplateVariableSets
		varset   string
		items    []string
		expected bool
	}{
		{name: "existing variable set", vars: vars, varset: "vs1", expected: true},
		{name: "unknown variable set", vars: vars, varset: "unknown", expected: false},
		{name: "name with a dash", vars: vars, varset: "vs-with-dash", expected: true},
		// An empty variable set still exists, where map truthiness would say false
		{name: "existing variable set without item", vars: vars, varset: "empty-vs", expected: true},
		{name: "existing item", vars: vars, varset: "vs1", items: []string{"A"}, expected: true},
		{name: "unknown item", vars: vars, varset: "vs1", items: []string{"MISSING"}, expected: false},
		{name: "all items must exist", vars: vars, varset: "vs1", items: []string{"A", "B"}, expected: true},
		{name: "one missing item is enough to fail", vars: vars, varset: "vs1", items: []string{"A", "MISSING"}, expected: false},
		{name: "item on unknown variable set", vars: vars, varset: "unknown", items: []string{"A"}, expected: false},
		// Callers without any project context (template preview) pass a nil map
		{name: "nil variable sets", vars: nil, varset: "vs1", expected: false},
		{name: "nil variable sets with item", vars: nil, varset: "vs1", items: []string{"A"}, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.vars.Exists(tt.varset, tt.items...))
		})
	}
}

// One job per case handled by TemplateVariableSets.Exists
const varsTemplate = `name: myTemplate
spec: |-
  on:
    push: {}
  jobs:
    always:
      runs-on: mymodel
    [[- if .vars.Exists "vs-present" ]]
    jobVarsetPresent:
      runs-on: mymodel
      vars: [vs-present]
    [[- end ]]
    [[- if .vars.Exists "vs-absent" ]]
    jobVarsetAbsent:
      runs-on: mymodel
    [[- end ]]
    [[- if .vars.Exists "vs-empty" ]]
    jobVarsetEmpty:
      runs-on: mymodel
    [[- end ]]
    [[- if .vars.Exists "vs-present" "TOKEN" ]]
    jobItemPresent:
      runs-on: mymodel
    [[- end ]]
    [[- if .vars.Exists "vs-present" "NOPE" ]]
    jobItemAbsent:
      runs-on: mymodel
    [[- end ]]
`

func TestResolveWithVariableSets(t *testing.T) {
	var work V2Workflow
	require.NoError(t, yaml.Unmarshal([]byte("name: myworkflow\nfrom: library/myTemplate"), &work))

	var template V2WorkflowTemplate
	require.NoError(t, yaml.Unmarshal([]byte(varsTemplate), &template))

	vars := NewTemplateVariableSets([]ProjectVariableSet{
		{Name: "vs-present", Items: []ProjectVariableSetItem{{Name: "TOKEN"}}},
		{Name: "vs-empty"},
	})

	_, err := template.Resolve(context.TODO(), &work, vars)
	require.NoError(t, err)

	require.Contains(t, work.Jobs, "always")
	require.Contains(t, work.Jobs, "jobVarsetPresent")
	require.Contains(t, work.Jobs, "jobVarsetEmpty")
	require.Contains(t, work.Jobs, "jobItemPresent")
	require.NotContains(t, work.Jobs, "jobVarsetAbsent")
	require.NotContains(t, work.Jobs, "jobItemAbsent")
	// No extra job must be generated
	require.Equal(t, 4, len(work.Jobs))

	require.Equal(t, []string{"vs-present"}, work.Jobs["jobVarsetPresent"].VariableSets)
}

// A nil TemplateVariableSets must not break the template execution: every gated block is dropped
// and existing templates keep resolving.
func TestResolveWithNilVariableSets(t *testing.T) {
	var work V2Workflow
	require.NoError(t, yaml.Unmarshal([]byte("name: myworkflow\nfrom: library/myTemplate"), &work))

	var template V2WorkflowTemplate
	require.NoError(t, yaml.Unmarshal([]byte(varsTemplate), &template))

	_, err := template.Resolve(context.TODO(), &work, nil)
	require.NoError(t, err)

	require.Contains(t, work.Jobs, "always")
	require.Equal(t, 1, len(work.Jobs))
}
