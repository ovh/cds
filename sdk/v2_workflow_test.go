package sdk

import (
	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUnmarshalV2WorkflowHooksDetailed(t *testing.T) {
	src := `jobs:
  myFirstJob:
    name: This is my first  job
    region: build
    runs-on: docker-debian
    steps:
      - run: 'echo "Workflow: ${{cds.workflow}}"'
name: MyDistantWorkflow
"on":
  model_update:
    models:
      - mymodel
    target_branch: develop
  push:
    branches:
      - master
  workflow_update:
    target_branch: master
`
	var w V2Workflow
	require.NoError(t, yaml.Unmarshal([]byte(src), &w))
	bts, err := yaml.Marshal(w)
	require.NoError(t, err)

	require.Equal(t, src, string(bts))
}

func TestUnmarshalV2WorkflowHooksShort(t *testing.T) {
	src := `jobs:
  myFirstJob:
    name: This is my first  job
    region: build
    runs-on: docker-debian
    steps:
      - run: 'echo "Workflow: ${{cds.workflow}}"'
name: MyDistantWorkflow
"on":
  - push
  - workflow_update
  - model_update
`
	var w V2Workflow
	require.NoError(t, yaml.Unmarshal([]byte(src), &w))

	bts, err := yaml.Marshal(w)
	require.NoError(t, err)

	require.Equal(t, src, string(bts))
}
