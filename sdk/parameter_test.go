package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParameterMapMerge(t *testing.T) {
	original := map[string]string{
		"test":     "value1",
		"default":  "ok",
		"git.hash": "XXX",
	}
	override := map[string]string{
		"test":     "override",
		"git.hash": "XXXBIS",
	}

	res := ParametersMapMerge(original, override)
	assert.Equal(t, "override", res["test"])
	assert.Equal(t, "ok", res["default"])
	assert.Equal(t, "XXXBIS", res["git.hash"])
}

func TestParameterMapMerge_WithExcludeGitParams(t *testing.T) {
	original := map[string]string{
		"test":     "value1",
		"default":  "ok",
		"git.hash": "XXX",
	}
	override := map[string]string{
		"test":     "override",
		"git.hash": "XXXBIS",
	}

	res := ParametersMapMerge(original, override, MapMergeOptions.ExcludeGitParams)
	assert.Equal(t, "override", res["test"])
	assert.Equal(t, "ok", res["default"])
	assert.Equal(t, "XXX", res["git.hash"])
}

func TestParametersMerge(t *testing.T) {
	original := []Parameter{
		Parameter{Name: "test", Value: "value1"},
		Parameter{Name: "default", Value: "ok"},
	}
	override := []Parameter{
		Parameter{Name: "test", Value: "override"},
	}

	res := ParametersMerge(original, override)
	for _, param := range res {
		if param.Name == "test" {
			assert.Equal(t, "override", param.Value)
			continue
		}
		if param.Name == "default" {
			assert.Equal(t, "ok", param.Value)
		}
	}
}
