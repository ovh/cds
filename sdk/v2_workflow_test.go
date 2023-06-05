package sdk

import (
	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLint(t *testing.T) {
	wk := `name: MyFirstWorkflow
jobs:
  myFirstjob:
    name: This is my first jobe
    contexts: [git]
    env:
      wname: ${{ cds.workflow }}
      proj: ${{ cds.project }}
    if: cds.workflow == 'MyFirstWorkflow'
    steps:
    - run: |-
        echo "Workflow: ${WNAME}"
    - uses: actions/PROJ/stash-build/SGU/cds-test-repo/test-child-action@tt7
      with:
        projectName: ${{ env.proj }}
    - run: |-
        echo "End""
`
	var w V2Workflow
	require.NoError(t, yaml.Unmarshal([]byte(wk), &w))
	errors := w.Lint()
	t.Logf("%+v", errors)
}
