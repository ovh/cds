package sdk

import (
	"context"
	"testing"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
)

func Test_result_as_annotation_expression(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	// Usage as annotations expression
	a := ActionParser{
		contexts: map[string]interface{}{
			"jobs": map[string]interface{}{
				"myJob": map[string]interface{}{
					"results": map[string]interface{}{
						"JobRunResults": map[string]interface{}{
							"generic:foo.txt": V2WorkflowRunResultGenericDetail{},
						},
					},
				},
			},
		},
	}

	r, err := result(context.TODO(), &a, "generic", "foo.*")
	require.NoError(t, err)
	require.NotNil(t, r)
}

func Test_result_as_script_expression(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)

	// Usage as expression in script
	a := ActionParser{
		contexts: map[string]interface{}{
			"jobs": map[string]interface{}{
				"myJob": map[string]interface{}{
					"JobRunResults": map[string]interface{}{
						"generic:foo.txt": V2WorkflowRunResultGenericDetail{},
					},
				},
			},
		},
	}

	r, err := result(context.TODO(), &a, "generic", "foo.*")
	require.NoError(t, err)
	require.NotNil(t, r)
}
