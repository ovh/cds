package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatrixParallelism(t *testing.T) {
	cases := []struct {
		name                         string
		defaultValue, maxValue       int64
		expectedDefault, expectedMax int64
	}{
		{"unset", 0, 0, 20, 40},
		{"configured", 10, 30, 10, 30},
		{"negative", -1, -1, 20, 40},
		{"max lower than default", 30, 10, 30, 30},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			api := &API{}
			api.Config.WorkflowV2.MatrixDefaultParallelism = c.defaultValue
			api.Config.WorkflowV2.MatrixMaxParallelism = c.maxValue
			defaultParallelism, maxParallelism := api.matrixParallelism()
			require.Equal(t, c.expectedDefault, defaultParallelism)
			require.Equal(t, c.expectedMax, maxParallelism)
		})
	}
}
