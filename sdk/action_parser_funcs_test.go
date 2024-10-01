package sdk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

func TestHashFiles(t *testing.T) {
	path := filepath.Join(os.TempDir(), "testdata", t.Name())
	defer os.RemoveAll(path)
	require.NoError(t, os.MkdirAll(path, os.FileMode(0755)))

	log.Factory = log.NewTestingWrapper(t)
	a := ActionParser{
		contexts: map[string]interface{}{
			"cds": map[string]interface{}{
				"workspace": "/home/sguiheux/src/github.com/ovh/cds/sdk",
			},
		},
	}

	d1 := []byte("I'm file 1")
	err := os.WriteFile(path+"/file1", d1, 0755)
	require.NoError(t, err)

	d2 := []byte("I'm file 2")
	err = os.WriteFile(path+"/file2", d2, 0755)
	require.NoError(t, err)

	hashSum1, err := hashFiles(context.TODO(), &a, path+"/file1", path+"/file2")
	require.NoError(t, err)

	hashSum2, err := hashFiles(context.TODO(), &a, path+"/file2", path+"/file1")
	require.NoError(t, err)

	t.Logf("%s", hashSum1)
	t.Logf("%s", hashSum2)

	require.Equal(t, fmt.Sprintf("%s", hashSum1), fmt.Sprintf("%s", hashSum2))
}
