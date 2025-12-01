package sdk

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestYAMLGenerator_WorkerModel(t *testing.T) {
	schema := GetWorkerModelJsonSchema()

	gen := NewYAMLGenerator()
	var buf bytes.Buffer

	err := gen.Generate(&buf, schema)
	require.NoError(t, err)

	output := buf.String()
	t.Log("Generated YAML example:")
	t.Log(output)

	// Vérifications de base
	require.Contains(t, output, "name:")
	require.Contains(t, output, "#", "Should contain comments")
}

func TestYAMLGenerator_Action(t *testing.T) {
	schema := GetActionJsonSchema([]string{"checkout", "script"})

	gen := NewYAMLGenerator()
	var buf bytes.Buffer

	err := gen.Generate(&buf, schema)
	require.NoError(t, err)

	output := buf.String()
	t.Log("Generated YAML example:")
	t.Log(output)

	// Vérifications de base
	require.Contains(t, output, "name:")
	require.Contains(t, output, "#", "Should contain comments")
}

func TestYAMLGenerator_AllOfVariants(t *testing.T) {
	schema := GetWorkerModelJsonSchema()

	gen := NewYAMLGenerator()
	var buf bytes.Buffer

	err := gen.Generate(&buf, schema)
	require.NoError(t, err)

	output := buf.String()
	t.Log("Generated YAML with AllOf variants:")
	t.Log(output)

	// Vérifier la présence des 3 exemples
	require.Contains(t, output, "# Example 1: type=docker")
	require.Contains(t, output, "# Example 2: type=openstack")
	require.Contains(t, output, "# Example 3: type=vsphere")

	// Vérifier que spec est développé pour chaque type
	require.Contains(t, output, "spec:")
	require.Contains(t, output, "image:", "Docker spec should have image")
	require.Contains(t, output, "flavor:", "Openstack spec should have flavor")
	require.Contains(t, output, "envs:", "Docker spec should have envs")
}
