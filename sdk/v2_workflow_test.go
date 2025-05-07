package sdk

import (
	"slices"
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
      memory: "4096"
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

func TestAncestor(t *testing.T) {
	w := V2Workflow{
		Stages: map[string]WorkflowStage{
			"stage1": {},
			"stage2": {},
			"stage3": {
				Needs: []string{"stage1", "stage2"},
			},
			"stage4": {
				Needs: []string{"stage3"},
			},
		},
		Jobs: map[string]V2Job{
			"job1": {
				Stage: "stage1",
			},
			"job11": {
				Stage: "stage1",
			},
			"job111": {
				Stage: "stage1",
				Needs: []string{"job1", "job11"},
			},
			"job2": {
				Stage: "stage2",
			},
			"job22": {
				Stage: "stage2",
			},
			"job222": {
				Stage: "stage2",
				Needs: []string{"job2", "job22"},
			},
			"job3": {
				Stage: "stage3",
			},
			"job33": {
				Stage: "stage3",
			},
			"job333": {
				Stage: "stage3",
				Needs: []string{"job3", "job33"},
			},
			"job4": {
				Stage: "stage4",
			},
		},
	}

	parents := WorkflowJobParents(w, "job333")
	require.True(t, slices.Contains(parents, "job1"))
	require.True(t, slices.Contains(parents, "job11"))
	require.True(t, slices.Contains(parents, "job111"))
	require.True(t, slices.Contains(parents, "job2"))
	require.True(t, slices.Contains(parents, "job22"))
	require.True(t, slices.Contains(parents, "job222"))
	require.True(t, slices.Contains(parents, "job3"))
	require.True(t, slices.Contains(parents, "job33"))
	require.Len(t, parents, 8)

	parents = WorkflowJobParents(w, "job22")
	require.Len(t, parents, 0)

	parents = WorkflowJobParents(w, "job111")
	require.Len(t, parents, 2)
	require.True(t, slices.Contains(parents, "job1"))
	require.True(t, slices.Contains(parents, "job11"))

	parents = WorkflowJobParents(w, "job4")
	require.True(t, slices.Contains(parents, "job1"))
	require.True(t, slices.Contains(parents, "job11"))
	require.True(t, slices.Contains(parents, "job111"))
	require.True(t, slices.Contains(parents, "job2"))
	require.True(t, slices.Contains(parents, "job22"))
	require.True(t, slices.Contains(parents, "job222"))
	require.True(t, slices.Contains(parents, "job3"))
	require.True(t, slices.Contains(parents, "job33"))
	require.True(t, slices.Contains(parents, "job333"))
	require.Len(t, parents, 9)
}
