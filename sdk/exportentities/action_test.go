package exportentities

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestGitClone(t *testing.T) {
	content := `
  name: "test"
  steps:
  - checkout: '{{ .cds.workspace }}'
  - gitClone:
      commit: HEAD
      directory: '/external-packages'
      url: 'ssh://external-packages.git'
      privateKey: proj-ssh
      tag: ''`

	var jobAsCode Action

	require.NoError(t, yaml.Unmarshal([]byte(content), &jobAsCode))

	require.Len(t, jobAsCode.Steps, 2)

	job, err := jobAsCode.GetAction()
	require.NoError(t, err)

	require.Equal(t, "CheckoutApplication", job.Actions[0].Name)
	require.Equal(t, "{{ .cds.workspace }}", job.Actions[0].Parameters[0].Value)

	require.Equal(t, "GitClone", job.Actions[1].Name)
	require.Len(t, job.Actions[1].Parameters, 5)
	for _, p := range job.Actions[1].Parameters {
		t.Log(p.Name, p.Value)
		switch p.Name {
		case "commit":
			require.Equal(t, "HEAD", p.Value)
		case "directory":
			require.Equal(t, "/external-packages", p.Value)
		case "url":
			require.Equal(t, "ssh://external-packages.git", p.Value)
		case "privateKey":
			require.Equal(t, "proj-ssh", p.Value)
		case "tag":
			require.Equal(t, "", p.Value)
		}
	}
}
