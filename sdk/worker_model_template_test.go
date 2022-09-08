package sdk

import (
	"fmt"
	"testing"

	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/require"
)

func TestWorkerModelTemplateDockerWithoutCmd(t *testing.T) {
	tmpl := `name: debian9
description: "my debian worker model"
type: docker
spec:
  shell: sh -c
`

	var wmTemplate WorkerModelTemplate
	require.NoError(t, yaml.Unmarshal([]byte(tmpl), &wmTemplate))

	err := wmTemplate.Lint()
	require.NotEqual(t, 0, len(err))
	require.Contains(t, fmt.Sprintf("%v", err), "cmd is required")
}

func TestWorkerModelTemplateDockerOK(t *testing.T) {
	tmpl := `name: debian9
description: "my debian worker model"
type: docker
spec:
  shell: sh -c
  cmd: ./worker
`
	var wmTemplate WorkerModelTemplate
	require.NoError(t, yaml.Unmarshal([]byte(tmpl), &wmTemplate))

	require.Nil(t, wmTemplate.Lint())

}
