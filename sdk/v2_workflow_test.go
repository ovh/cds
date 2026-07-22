package sdk

import (
	"slices"
	"testing"

	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/require"
)

func TestV2JobNeedsTemplateResolution(t *testing.T) {
	tests := []struct {
		name string
		job  V2Job
		want bool
	}{
		{
			name: "from only is an unresolved reference",
			job:  V2Job{From: ".cds/workflow-templates/build.yml"},
			want: true,
		},
		{
			name: "from with matrix but no content is an unresolved reference",
			job: V2Job{
				From:     "proj/vcs/my/repo/tmpl@refs/heads/main",
				Strategy: &V2JobStrategy{Matrix: map[string]interface{}{"region": []string{"r1", "r2"}}},
			},
			want: true,
		},
		{
			name: "from with steps is already resolved",
			job:  V2Job{From: "tmpl", Steps: []ActionStep{{Run: "echo"}}},
			want: false,
		},
		{
			name: "from with runs-on is already resolved",
			job:  V2Job{From: "tmpl", RunsOn: V2JobRunsOn{Model: "docker-debian"}},
			want: false,
		},
		{
			name: "job without from has nothing to resolve",
			job:  V2Job{Steps: []ActionStep{{Run: "echo"}}},
			want: false,
		},
		{
			name: "empty job without from has nothing to resolve",
			job:  V2Job{},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.job.NeedsTemplateResolution())
		})
	}
}

func TestV2WorkflowLintJobTemplateReference(t *testing.T) {
	jobFromOnly := V2Job{From: ".cds/workflow-templates/build.yml"}
	jobFromAndRunsOn := V2Job{From: ".cds/workflow-templates/build.yml", RunsOn: V2JobRunsOn{Model: "docker-debian"}}
	jobFromAndSteps := V2Job{From: ".cds/workflow-templates/build.yml", Steps: []ActionStep{{Run: "echo"}}}

	t.Run("yaml definition accepts a job template reference", func(t *testing.T) {
		w := V2Workflow{Name: "w", Jobs: map[string]V2Job{"myjob": jobFromOnly}}
		require.Empty(t, w.LintYamlDefinition())
	})
	t.Run("yaml definition rejects from combined with runs-on", func(t *testing.T) {
		w := V2Workflow{Name: "w", Jobs: map[string]V2Job{"myjob": jobFromAndRunsOn}}
		errs := w.LintYamlDefinition()
		require.Len(t, errs, 1)
		require.Contains(t, errs[0].Error(), "from cannot be combined with steps or runs-on")
	})
	t.Run("yaml definition rejects from combined with steps", func(t *testing.T) {
		w := V2Workflow{Name: "w", Jobs: map[string]V2Job{"myjob": jobFromAndSteps}}
		errs := w.LintYamlDefinition()
		require.Len(t, errs, 1)
		require.Contains(t, errs[0].Error(), "from cannot be combined with steps or runs-on")
	})
	t.Run("workflow run data accepts from as provenance on resolved jobs", func(t *testing.T) {
		w := V2Workflow{Name: "w", Jobs: map[string]V2Job{"myjob": jobFromAndRunsOn}}
		require.Empty(t, w.LintWorkflowRunData())
	})
	t.Run("workflow from a template skips both lints", func(t *testing.T) {
		w := V2Workflow{Name: "w", From: "proj/vcs/my/repo/tmpl", Jobs: map[string]V2Job{"myjob": jobFromAndRunsOn}}
		require.Empty(t, w.LintYamlDefinition())
		require.Empty(t, w.LintWorkflowRunData())
	})
}

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
