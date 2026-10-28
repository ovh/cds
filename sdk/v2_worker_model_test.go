package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/require"
)

func TestWorkerDockerModelWithoutImage(t *testing.T) {
	dockerWM := `
    name: debian9
    description: "my debian worker model"
    type: docker
    spec:
      envs:
        CDS_GRAYLOG_EXTRA_KEY: '{{.GraylogExtraKey}}'
        CDS_GRAYLOG_EXTRA_VALUE: '{{.GraylogExtraValue}}'
        CDS_GRAYLOG_HOST: '{{.GraylogHost}}'
        CDS_GRAYLOG_PORT: '{{.GraylogPort}}'
  `

	var dockerModel V2WorkerModel
	require.NoError(t, yaml.Unmarshal([]byte(dockerWM), &dockerModel))

	err := dockerModel.LintYamlDefinition()
	require.NotEqual(t, 0, len(err))
	require.Contains(t, fmt.Sprintf("%v", err), "image is required")
}

func TestWorkerDockerModelWrongType(t *testing.T) {
	dockerWM := `
    name: debian9
    description: "my debian worker model"
    type: marathon
    spec:
      envs:
        CDS_GRAYLOG_EXTRA_KEY: '{{.GraylogExtraKey}}'
        CDS_GRAYLOG_EXTRA_VALUE: '{{.GraylogExtraValue}}'
        CDS_GRAYLOG_HOST: '{{.GraylogHost}}'
        CDS_GRAYLOG_PORT: '{{.GraylogPort}}'
  `

	var dockerModel V2WorkerModel
	require.NoError(t, yaml.Unmarshal([]byte(dockerWM), &dockerModel))

	err := dockerModel.LintYamlDefinition()
	require.NotEqual(t, 0, len(err))
	require.Contains(t, fmt.Sprintf("%v", err), "type must be one of the following")
}

const microVMSHA = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func lintMicroVMModel(t *testing.T, spec string) []error {
	wm := `
    name: ubuntu-microvm
    osarch: linux/amd64
    type: microvm
    spec:
` + spec
	var model V2WorkerModel
	require.NoError(t, yaml.Unmarshal([]byte(wm), &model))
	return model.LintYamlDefinition()
}

func TestWorkerMicroVMModelOK(t *testing.T) {
	errs := lintMicroVMModel(t, `
      kernel:
        url: https://example.org/vmlinux
        sha256: `+microVMSHA+`
      rootfs:
        url: https://example.org/rootfs.squashfs
        sha256: `+microVMSHA+`
        format: squashfs
        size: 4096
      vcpus: 2
      memory: 2048
`)
	require.Nil(t, errs)
}

// The size is optional: the hatchery applies its defaults.
func TestWorkerMicroVMModelWithoutSizeOK(t *testing.T) {
	errs := lintMicroVMModel(t, `
      kernel:
        url: https://example.org/vmlinux
        sha256: `+microVMSHA+`
      rootfs:
        url: https://example.org/rootfs.ext4
        sha256: `+microVMSHA+`
`)
	require.Nil(t, errs)
}

// The platform refuses to boot an artifact it cannot check, so the lint must
// catch a missing or malformed checksum at analysis time, not at spawn time.
func TestWorkerMicroVMModelInvalid(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		wantErr string
	}{
		{
			name: "missing kernel",
			spec: `
      rootfs:
        url: https://example.org/rootfs.ext4
        sha256: ` + microVMSHA + `
`,
			wantErr: "kernel is required",
		},
		{
			name: "kernel without checksum",
			spec: `
      kernel:
        url: https://example.org/vmlinux
      rootfs:
        url: https://example.org/rootfs.ext4
        sha256: ` + microVMSHA + `
`,
			wantErr: "sha256 is required",
		},
		{
			name: "uppercase checksum",
			spec: `
      kernel:
        url: https://example.org/vmlinux
        sha256: ` + strings.ToUpper(microVMSHA) + `
      rootfs:
        url: https://example.org/rootfs.ext4
        sha256: ` + microVMSHA + `
`,
			wantErr: "Does not match pattern",
		},
		{
			name: "unknown rootfs format",
			spec: `
      kernel:
        url: https://example.org/vmlinux
        sha256: ` + microVMSHA + `
      rootfs:
        url: https://example.org/rootfs.qcow2
        sha256: ` + microVMSHA + `
        format: qcow2
`,
			wantErr: "format must be one of the following",
		},
		{
			name: "too many vcpus",
			spec: `
      kernel:
        url: https://example.org/vmlinux
        sha256: ` + microVMSHA + `
      rootfs:
        url: https://example.org/rootfs.ext4
        sha256: ` + microVMSHA + `
      vcpus: 64
`,
			wantErr: "less than or equal to 32",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := lintMicroVMModel(t, tt.spec)
			require.NotEmpty(t, errs)
			require.Contains(t, fmt.Sprintf("%v", errs), tt.wantErr)
		})
	}
}

// Git variables in the artifact URLs let a repository boot the image built
// from the same commit; the checksums are never interpolated.
func TestEntityInterpolateMicroVMModel(t *testing.T) {
	e := EntityWithObject{
		Entity: Entity{Ref: GitRefBranchPrefix + "main", Commit: "abcdef"},
		Model: V2WorkerModel{
			Type: WorkerModelTypeMicroVM,
			Spec: json.RawMessage(`{"kernel":{"url":"https://example.org/${{ git.ref_name }}/vmlinux","sha256":"` + microVMSHA + `"},"rootfs":{"url":"https://example.org/${{ git.sha }}/rootfs.ext4","sha256":"` + microVMSHA + `"},"vcpus":2,"memory":1024}`),
		},
	}
	require.NoError(t, e.Interpolate(context.TODO()))

	var spec V2WorkerModelMicroVMSpec
	require.NoError(t, json.Unmarshal(e.Model.Spec, &spec))
	require.Equal(t, "https://example.org/main/vmlinux", spec.Kernel.URL)
	require.Equal(t, "https://example.org/abcdef/rootfs.ext4", spec.Rootfs.URL)
	require.Equal(t, microVMSHA, spec.Kernel.SHA256)
	require.Equal(t, 2, spec.VCPUs)
	require.Equal(t, 1024, spec.Memory)
}

func TestWorkerDockerModelOK(t *testing.T) {
	dockerWM := `
    name: debian9
    description: "my debian worker model"
    osarch: linux/amd64
    type: docker
    spec:
      image: myimage
  `

	var dockerModel V2WorkerModel
	require.NoError(t, yaml.Unmarshal([]byte(dockerWM), &dockerModel))

	require.Nil(t, dockerModel.LintYamlDefinition())
}
