package sdk

import (
	"testing"

	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalV2Job(t *testing.T) {
	src := `jobs:
  myFirstJob:
    name: This is my first  job
    region: build
    runs-on: docker-debian
    steps:
      - run: 'echo "Workflow: ${{cds.workflow}}"'
name: MyDistantWorkflow
`
	var w V2Workflow
	require.NoError(t, yaml.Unmarshal([]byte(src), &w))

	require.Equal(t, "docker-debian", w.Jobs["myFirstJob"].RunsOn.Model)

	bts, err := yaml.Marshal(w)
	require.NoError(t, err)

	require.Equal(t, src, string(bts))
}

func TestUnmarshalV2JobFullRunsOn(t *testing.T) {
	src := `jobs:
  myFirstJob:
    name: This is my first  job
    region: build
    runs-on:
      flavor: b2-7
      memory: 4096
      model: docker-debian
    steps:
      - run: 'echo "Workflow: ${{cds.workflow}}"'
name: MyDistantWorkflow
`
	var w V2Workflow
	require.NoError(t, yaml.Unmarshal([]byte(src), &w))

	require.Equal(t, "docker-debian", w.Jobs["myFirstJob"].RunsOn.Model)

	bts, err := yaml.Marshal(w)
	require.NoError(t, err)

	require.Equal(t, src, string(bts))
}

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
  model-update:
    models:
      - mymodel
    target_branch: develop
  push:
    branches:
      - master
  workflow-update:
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
  - workflow-update
  - model-update
`
	var w V2Workflow
	require.NoError(t, yaml.Unmarshal([]byte(src), &w))

	bts, err := yaml.Marshal(w)
	require.NoError(t, err)

	require.Equal(t, src, string(bts))
}
