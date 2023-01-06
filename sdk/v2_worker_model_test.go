package sdk

import (
	"fmt"
	"testing"

	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/require"
)

func TestWorkerDockerModelWithoutImage(t *testing.T) {
	dockerWM := `name: debian9
image: debian:9
description: "my debian worker model"
type: docker
spec:
  envs:
    CDS_GRAYLOG_EXTRA_KEY: '{{.GraylogExtraKey}}'
    CDS_GRAYLOG_EXTRA_VALUE: '{{.GraylogExtraValue}}'
    CDS_GRAYLOG_HOST: '{{.GraylogHost}}'
    CDS_GRAYLOG_PORT: '{{.GraylogPort}}'
  cmd: curl {{.API}}/download/worker/linux/$(uname -m) -o worker && chmod +x worker && exec ./worker`

	var dockerModel V2WorkerModel
	require.NoError(t, yaml.Unmarshal([]byte(dockerWM), &dockerModel))

	err := dockerModel.Lint()
	require.NotEqual(t, 0, len(err))
	require.Contains(t, fmt.Sprintf("%v", err), "image is required")
}

func TestWorkerDockerModelWrongType(t *testing.T) {
	dockerWM := `name: debian9
image: debian:9
description: "my debian worker model"
type: marathon
spec:
  envs:
    CDS_GRAYLOG_EXTRA_KEY: '{{.GraylogExtraKey}}'
    CDS_GRAYLOG_EXTRA_VALUE: '{{.GraylogExtraValue}}'
    CDS_GRAYLOG_HOST: '{{.GraylogHost}}'
    CDS_GRAYLOG_PORT: '{{.GraylogPort}}'
  cmd: curl {{.API}}/download/worker/linux/$(uname -m) -o worker && chmod +x worker && exec ./worker`

	var dockerModel V2WorkerModel
	require.NoError(t, yaml.Unmarshal([]byte(dockerWM), &dockerModel))

	err := dockerModel.Lint()
	require.NotEqual(t, 0, len(err))
	require.Contains(t, fmt.Sprintf("%v", err), "type must be one of the following")
}

func TestWorkerDockerModelOK(t *testing.T) {
	dockerWM := `name: debian9
image: debian:9
description: "my debian worker model"
type: docker
spec:
  image: myimage'
  cmd: curl {{.API}}/download/worker/linux/$(uname -m) -o worker && chmod +x worker && exec ./worker
  shell: sh -c`

	var dockerModel V2WorkerModel
	require.NoError(t, yaml.Unmarshal([]byte(dockerWM), &dockerModel))

	require.Nil(t, dockerModel.Lint())
}
